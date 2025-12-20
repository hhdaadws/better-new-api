package dto

type ChannelSettings struct {
	ForceFormat            bool   `json:"force_format,omitempty"`
	ThinkingToContent      bool   `json:"thinking_to_content,omitempty"`
	Proxy                  string `json:"proxy"`
	PassThroughBodyEnabled bool   `json:"pass_through_body_enabled,omitempty"`
	SystemPrompt           string `json:"system_prompt,omitempty"`
	SystemPromptOverride   bool   `json:"system_prompt_override,omitempty"`
	// 渠道审核功能
	HeaderAuditEnabled   bool   `json:"header_audit_enabled,omitempty"`   // 是否启用请求头审核
	HeaderAuditRules     string `json:"header_audit_rules,omitempty"`     // 请求头审核规则，JSON格式：{"header-name": "regex-pattern"}
	ContentAuditEnabled  bool   `json:"content_audit_enabled,omitempty"`  // 是否启用内容审核
	ContentAuditKeywords string `json:"content_audit_keywords,omitempty"` // 内容审核关键词，换行分隔
	// 粘性会话功能
	StickySessionEnabled    bool `json:"sticky_session_enabled,omitempty"`     // 是否启用粘性会话
	StickySessionMaxCount   int  `json:"sticky_session_max_count,omitempty"`   // 最大会话数(0=无限制)
	StickySessionTTLMinutes int  `json:"sticky_session_ttl_minutes,omitempty"` // 会话过期时间(分钟，默认60)
	// Session并发错误自动排除
	SessionConcurrencyAutoExclude    bool `json:"session_concurrency_auto_exclude,omitempty"`    // 并发错误自动排除
	SessionConcurrencyExcludeMinutes int  `json:"session_concurrency_exclude_minutes,omitempty"` // 排除时间(分钟,默认2)
	// Claude 缓存计费设置
	CacheCreation1hAs5m bool `json:"cache_creation_1h_as_5m,omitempty"` // 1小时缓存创建按5分钟倍率计费
	// Claude Code 测试伪装
	ClaudeCodeTestEnabled bool `json:"claude_code_test_enabled,omitempty"` // 是否启用 Claude Code 测试伪装
}

type VertexKeyType string

const (
	VertexKeyTypeJSON   VertexKeyType = "json"
	VertexKeyTypeAPIKey VertexKeyType = "api_key"
)

type AwsKeyType string

const (
	AwsKeyTypeAKSK   AwsKeyType = "ak_sk" // 默认
	AwsKeyTypeApiKey AwsKeyType = "api_key"
)

type ChannelOtherSettings struct {
	AzureResponsesVersion string        `json:"azure_responses_version,omitempty"`
	VertexKeyType         VertexKeyType `json:"vertex_key_type,omitempty"` // "json" or "api_key"
	OpenRouterEnterprise  *bool         `json:"openrouter_enterprise,omitempty"`
	AllowServiceTier      bool          `json:"allow_service_tier,omitempty"`      // 是否允许 service_tier 透传（默认过滤以避免额外计费）
	DisableStore          bool          `json:"disable_store,omitempty"`           // 是否禁用 store 透传（默认允许透传，禁用后可能导致 Codex 无法使用）
	AllowSafetyIdentifier bool          `json:"allow_safety_identifier,omitempty"` // 是否允许 safety_identifier 透传（默认过滤以保护用户隐私）
	AwsKeyType            AwsKeyType    `json:"aws_key_type,omitempty"`
	PassThroughHeaders    bool          `json:"pass_through_headers,omitempty"` // 是否透传全部客户端请求头（默认false，仅透传Content-Type和Accept）
}

func (s *ChannelOtherSettings) IsOpenRouterEnterprise() bool {
	if s == nil || s.OpenRouterEnterprise == nil {
		return false
	}
	return *s.OpenRouterEnterprise
}
