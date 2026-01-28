package claude

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const (
	RequestModeCompletion = 1
	RequestModeMessage    = 2
)

type Adaptor struct {
	RequestMode int
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	// 处理 tools 字段，确保 web_search 工具格式正确
	if request.Tools != nil {
		request.Tools = processClaudeTools(request.Tools)
	}
	return request, nil
}

// processClaudeTools 处理工具列表，确保 web_search 工具的 type 字段正确
func processClaudeTools(tools any) []any {
	toolsSlice, ok := tools.([]any)
	if !ok {
		return nil
	}

	result := make([]any, 0, len(toolsSlice))
	for _, tool := range toolsSlice {
		toolMap, ok := tool.(map[string]any)
		if !ok {
			result = append(result, tool)
			continue
		}

		toolType, _ := toolMap["type"].(string)
		// 检查是否是 web_search 工具（type 包含 web_search）
		if strings.Contains(toolType, "web_search") {
			// 创建新的 web_search 工具，确保 type 是 "web_search_20250305"
			webSearchTool := map[string]any{
				"type": "web_search_20250305",
				"name": "web_search",
			}
			// 复制其他字段
			if maxUses, exists := toolMap["max_uses"]; exists {
				webSearchTool["max_uses"] = maxUses
			}
			if userLocation, exists := toolMap["user_location"]; exists {
				webSearchTool["user_location"] = userLocation
			}
			result = append(result, webSearchTool)
		} else {
			// 普通工具，直接保留
			result = append(result, tool)
		}
	}
	return result
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	if strings.HasPrefix(info.UpstreamModelName, "claude-2") || strings.HasPrefix(info.UpstreamModelName, "claude-instant") {
		a.RequestMode = RequestModeCompletion
	} else {
		a.RequestMode = RequestModeMessage
	}
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := ""
	if a.RequestMode == RequestModeMessage {
		baseURL = fmt.Sprintf("%s/v1/messages", info.ChannelBaseUrl)
	} else {
		baseURL = fmt.Sprintf("%s/v1/complete", info.ChannelBaseUrl)
	}
	if info.IsClaudeBetaQuery {
		baseURL = baseURL + "?beta=true"
	}
	return baseURL, nil
}

func CommonClaudeHeadersOperation(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) {
	// common headers operation
	anthropicBeta := c.Request.Header.Get("anthropic-beta")
	if anthropicBeta != "" {
		req.Set("anthropic-beta", anthropicBeta)
	}
	model_setting.GetClaudeSettings().WriteHeaders(info.OriginModelName, req)
}

// SetupClaudeCodeTestHeaders 设置 Claude Code 测试请求头（保留用于测试接口）
func SetupClaudeCodeTestHeaders(req *http.Header) {
	// Claude Code SDK 指纹头
	req.Set("x-stainless-retry-count", "0")
	req.Set("x-stainless-timeout", "60")
	req.Set("x-stainless-lang", "js")
	req.Set("x-stainless-package-version", "0.55.1")
	req.Set("x-stainless-os", "Windows")
	req.Set("x-stainless-arch", "x64")
	req.Set("x-stainless-runtime", "node")
	req.Set("x-stainless-runtime-version", "v20.19.2")
	req.Set("anthropic-dangerous-direct-browser-access", "true")
	req.Set("x-app", "cli")
	req.Set("User-Agent", "claude-cli/2.0.72 (external, cli)")
	req.Set("accept-language", "*")
	req.Set("sec-fetch-mode", "cors")
	// anthropic-beta 特性
	req.Set("anthropic-beta", "oauth-2025-04-20,claude-code-20250219,interleaved-thinking-2025-05-14")
}

// SetupClaudeCodeStandardHeaders 注入标准 Claude Code 请求头
// 使用固定的标准值，确保请求不被 CRS 拦截
func SetupClaudeCodeStandardHeaders(c *gin.Context, req *http.Header) {
	// 1. anthropic-beta: 强制使用标准值（必须包含 oauth-2025-04-20）
	// 客户端可能缺少 oauth-2025-04-20，导致被 CRS 拦截
	req.Set("anthropic-beta", "oauth-2025-04-20,claude-code-20250219,interleaved-thinking-2025-05-14")

	// anthropic-version: 保留客户端的值
	if version := c.Request.Header.Get("anthropic-version"); version != "" {
		req.Set("anthropic-version", version)
	} else {
		req.Set("anthropic-version", "2023-06-01")
	}

	// 2. 标准 Claude Code SDK 指纹头（固定值）
	req.Set("x-stainless-retry-count", "0")
	req.Set("x-stainless-timeout", "600")
	req.Set("x-stainless-lang", "js")
	req.Set("x-stainless-package-version", "0.70.0")
	req.Set("x-stainless-os", "Windows")
	req.Set("x-stainless-arch", "x64")
	req.Set("x-stainless-runtime", "node")
	req.Set("x-stainless-runtime-version", "v24.7.0")

	// 3. 必要的身份标识头
	req.Set("anthropic-dangerous-direct-browser-access", "true")
	req.Set("x-app", "cli")
	req.Set("User-Agent", "claude-cli/2.1.22 (external, cli)")

	// 4. 其他必要头
	req.Set("accept-language", "*")
	req.Set("sec-fetch-mode", "cors")
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	// 基础请求头
	req.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	req.Set("Accept", c.Request.Header.Get("Accept"))
	req.Set("x-api-key", info.ApiKey)

	// 检查是否启用 Claude Code 伪装模式（从渠道设置读取）
	if info.ChannelSetting.ClaudeCodeTestEnabled {
		// 注入标准 Claude Code 请求头
		SetupClaudeCodeStandardHeaders(c, req)
	} else {
		// 原有的正常模式逻辑
		channel.SetupApiRequestHeader(info, c, req)
		anthropicVersion := c.Request.Header.Get("anthropic-version")
		if anthropicVersion == "" {
			anthropicVersion = "2023-06-01"
		}
		req.Set("anthropic-version", anthropicVersion)
		CommonClaudeHeadersOperation(c, req, info)
	}
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	if a.RequestMode == RequestModeCompletion {
		return RequestOpenAI2ClaudeComplete(*request), nil
	} else {
		return RequestOpenAI2ClaudeMessage(c, *request)
	}
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	// TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	if info.IsStream {
		return ClaudeStreamHandler(c, resp, info, a.RequestMode)
	} else {
		return ClaudeHandler(c, resp, info, a.RequestMode)
	}
	return
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
