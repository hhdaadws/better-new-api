package service

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

// ChannelAuditResult 渠道审核结果
type ChannelAuditResult struct {
	Passed       bool     // 是否通过审核
	FailedType   string   // 失败类型: "header" 或 "content"
	FailedReason string   // 失败原因
	MatchedItems []string // 匹配到的项目（header名称或关键词）
}

// CheckChannelAudit 执行渠道审核检查
// 包括请求头审核和内容审核
func CheckChannelAudit(headers http.Header, contentText string, channelSetting dto.ChannelSettings) *ChannelAuditResult {
	// 检查请求头审核
	if channelSetting.HeaderAuditEnabled && channelSetting.HeaderAuditRules != "" {
		result := checkHeaderAudit(headers, channelSetting.HeaderAuditRules)
		if !result.Passed {
			return result
		}
	}

	// 检查内容审核
	if channelSetting.ContentAuditEnabled && channelSetting.ContentAuditKeywords != "" {
		result := checkContentAudit(contentText, channelSetting.ContentAuditKeywords)
		if !result.Passed {
			return result
		}
	}

	return &ChannelAuditResult{Passed: true}
}

// checkHeaderAudit 检查请求头是否符合审核规则
// rulesJSON 格式: {"header-name": "regex-pattern", ...}
// 规则含义：指定的请求头必须存在且其值必须匹配对应的正则表达式
func checkHeaderAudit(headers http.Header, rulesJSON string) *ChannelAuditResult {
	if rulesJSON == "" {
		return &ChannelAuditResult{Passed: true}
	}

	// 解析 JSON 规则
	var rules map[string]string
	if err := common.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		return &ChannelAuditResult{
			Passed:       false,
			FailedType:   "header",
			FailedReason: fmt.Sprintf("invalid header audit rules JSON: %v", err),
		}
	}

	var failedHeaders []string
	for headerName, pattern := range rules {
		// 获取请求头值（HTTP 头名称不区分大小写）
		headerValue := headers.Get(headerName)

		// 编译正则表达式
		re, err := regexp.Compile(pattern)
		if err != nil {
			return &ChannelAuditResult{
				Passed:       false,
				FailedType:   "header",
				FailedReason: fmt.Sprintf("invalid regex pattern for header '%s': %v", headerName, err),
			}
		}

		// 检查是否匹配
		if !re.MatchString(headerValue) {
			failedHeaders = append(failedHeaders, headerName)
		}
	}

	if len(failedHeaders) > 0 {
		return &ChannelAuditResult{
			Passed:       false,
			FailedType:   "header",
			FailedReason: fmt.Sprintf("request headers do not match audit rules: %s", strings.Join(failedHeaders, ", ")),
			MatchedItems: failedHeaders,
		}
	}

	return &ChannelAuditResult{Passed: true}
}

// checkContentAudit 检查内容是否包含禁止的关键词
// keywords 格式：换行分隔的关键词列表
func checkContentAudit(contentText string, keywords string) *ChannelAuditResult {
	if keywords == "" || contentText == "" {
		return &ChannelAuditResult{Passed: true}
	}

	// 解析关键词列表
	keywordList := parseKeywordList(keywords)
	if len(keywordList) == 0 {
		return &ChannelAuditResult{Passed: true}
	}

	// 使用 AC 算法检查（不区分大小写）
	lowerContent := strings.ToLower(contentText)
	found, matchedWords := AcSearch(lowerContent, keywordList, false)

	if found {
		return &ChannelAuditResult{
			Passed:       false,
			FailedType:   "content",
			FailedReason: fmt.Sprintf("content contains blocked keywords: %s", strings.Join(matchedWords, ", ")),
			MatchedItems: matchedWords,
		}
	}

	return &ChannelAuditResult{Passed: true}
}

// parseKeywordList 解析换行分隔的关键词列表
func parseKeywordList(keywords string) []string {
	lines := strings.Split(keywords, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			// 转换为小写以便后续不区分大小写匹配
			result = append(result, strings.ToLower(trimmed))
		}
	}
	return result
}
