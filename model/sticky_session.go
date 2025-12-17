package model

// StickySession represents a sticky session binding
type StickySession struct {
	SessionHash string `json:"session_hash"`
	ChannelId   int    `json:"channel_id"`
	Group       string `json:"group"`
	Model       string `json:"model"`
	CreatedAt   int64  `json:"created_at"`
	TTL         int64  `json:"ttl"` // remaining TTL in seconds
}

// StickySessionInfo for API responses
type StickySessionInfo struct {
	ChannelId    int              `json:"channel_id"`
	ChannelName  string           `json:"channel_name"`
	SessionCount int              `json:"session_count"`
	MaxCount     int              `json:"max_count"`
	TTLMinutes   int              `json:"ttl_minutes"`
	Sessions     []StickySession  `json:"sessions,omitempty"`
}
