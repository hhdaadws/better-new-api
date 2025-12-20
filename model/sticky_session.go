package model

// StickySession represents a sticky session binding
type StickySession struct {
	SessionHash string `json:"session_hash"`
	ChannelId   int    `json:"channel_id"`
	Group       string `json:"group"`
	Model       string `json:"model"`
	Username    string `json:"username"`
	TokenName   string `json:"token_name"`
	CreatedAt   int64  `json:"created_at"`
	TTL         int64  `json:"ttl"` // remaining TTL in seconds
}

// StickySessionInfo for API responses
type StickySessionInfo struct {
	ChannelId      int             `json:"channel_id"`
	ChannelName    string          `json:"channel_name"`
	SessionCount   int             `json:"session_count"`
	MaxCount       int             `json:"max_count"`
	TTLMinutes     int             `json:"ttl_minutes"`
	DailyBindLimit int             `json:"daily_bind_limit"` // 每日绑定上限
	DailyBindCount int             `json:"daily_bind_count"` // 今日已绑定数量
	Sessions       []StickySession `json:"sessions,omitempty"`
}
