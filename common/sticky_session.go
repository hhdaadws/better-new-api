package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	StickySessionPrefix          = "sticky_session:"
	StickySessionByChannelPrefix = "sticky_sessions_by_channel:"
	SessionChannelUsagePrefix    = "session_channel_usage:"
)

// SessionChannelUsage 记录会话在渠道上的使用历史
type SessionChannelUsage struct {
	ChannelId  int   `json:"channel_id"`   // 渠道ID
	Priority   int64 `json:"priority"`     // 渠道优先级
	LastUsedAt int64 `json:"last_used_at"` // 最后使用时间戳
}

// StickySessionData stores the session binding information
type StickySessionData struct {
	ChannelId int    `json:"channel_id"`
	Group     string `json:"group"`
	Model     string `json:"model"`
	CreatedAt int64  `json:"created_at"`
	Username  string `json:"username"`
	TokenName string `json:"token_name"`
}

// GetStickySessionKey returns the Redis key for session->channel mapping
func GetStickySessionKey(group, model, sessionHash string) string {
	return fmt.Sprintf("%s%s:%s:%s", StickySessionPrefix, group, model, sessionHash)
}

// GetStickySessionByChannelKey returns the Redis key for channel's sessions index
func GetStickySessionByChannelKey(channelId int) string {
	return fmt.Sprintf("%s%d", StickySessionByChannelPrefix, channelId)
}

// GetStickySessionChannel retrieves the channel ID for a session
func GetStickySessionChannel(group, model, sessionHash string) (int, error) {
	if !RedisEnabled {
		return 0, nil
	}
	ctx := context.Background()
	key := GetStickySessionKey(group, model, sessionHash)
	val, err := RDB.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	// Try to parse as JSON first (new format)
	var data StickySessionData
	if jsonErr := json.Unmarshal([]byte(val), &data); jsonErr == nil {
		return data.ChannelId, nil
	}

	// Fallback: parse as plain integer (legacy format)
	channelId, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return channelId, nil
}

// SetStickySession creates or updates a sticky session binding
// Optimized: only writes to Redis when necessary (new session or missing user info)
func SetStickySession(group, model, sessionHash string, channelId int, ttlMinutes int, username, tokenName string) error {
	if !RedisEnabled {
		return nil
	}
	ctx := context.Background()
	key := GetStickySessionKey(group, model, sessionHash)
	ttl := time.Duration(ttlMinutes) * time.Minute

	// Check if session already exists
	existingVal, err := RDB.Get(ctx, key).Result()
	if err == nil && existingVal != "" {
		// Session exists, check if we need to update
		var existingData StickySessionData
		if jsonErr := json.Unmarshal([]byte(existingVal), &existingData); jsonErr == nil {
			// If user info already exists and channel matches, just renew TTL (fast path)
			if existingData.Username != "" && existingData.ChannelId == channelId {
				return RDB.Expire(ctx, key, ttl).Err()
			}
			// User info missing or channel changed, need to update with preserved CreatedAt
			data := StickySessionData{
				ChannelId: channelId,
				Group:     group,
				Model:     model,
				CreatedAt: existingData.CreatedAt, // Preserve original creation time
				Username:  username,
				TokenName: tokenName,
			}
			jsonData, _ := json.Marshal(data)
			return RDB.Set(ctx, key, string(jsonData), ttl).Err()
		}
		// Legacy format (plain integer), need to upgrade
	}

	// New session or legacy format upgrade
	data := StickySessionData{
		ChannelId: channelId,
		Group:     group,
		Model:     model,
		CreatedAt: time.Now().Unix(),
		Username:  username,
		TokenName: tokenName,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Use pipeline for atomic operation
	pipe := RDB.TxPipeline()

	// Set session -> channel mapping with TTL
	pipe.Set(ctx, key, string(jsonData), ttl)

	// Add to channel's session index (Sorted Set with timestamp as score)
	indexKey := GetStickySessionByChannelKey(channelId)
	score := float64(time.Now().Unix())
	// Store full session key as member for later lookup
	memberData := fmt.Sprintf("%s:%s:%s", group, model, sessionHash)
	pipe.ZAdd(ctx, indexKey, &redis.Z{Score: score, Member: memberData})

	_, err = pipe.Exec(ctx)
	return err
}

// DeleteStickySession removes a sticky session binding
func DeleteStickySession(group, model, sessionHash string, channelId int) error {
	if !RedisEnabled {
		return nil
	}
	ctx := context.Background()

	pipe := RDB.TxPipeline()

	// Delete session -> channel mapping
	key := GetStickySessionKey(group, model, sessionHash)
	pipe.Del(ctx, key)

	// Remove from channel's session index
	indexKey := GetStickySessionByChannelKey(channelId)
	memberData := fmt.Sprintf("%s:%s:%s", group, model, sessionHash)
	pipe.ZRem(ctx, indexKey, memberData)

	_, err := pipe.Exec(ctx)
	return err
}

// GetChannelStickySessionCount returns the number of active sessions for a channel
func GetChannelStickySessionCount(channelId int) (int, error) {
	if !RedisEnabled {
		return 0, nil
	}
	ctx := context.Background()
	indexKey := GetStickySessionByChannelKey(channelId)

	// First, clean up expired sessions
	cleanupExpiredSessions(ctx, channelId)

	count, err := RDB.ZCard(ctx, indexKey).Result()
	return int(count), err
}

// GetChannelStickySessions returns all sessions for a channel with their TTL
func GetChannelStickySessions(channelId int) ([]map[string]interface{}, error) {
	if !RedisEnabled {
		return nil, nil
	}
	ctx := context.Background()
	indexKey := GetStickySessionByChannelKey(channelId)

	// Get all members with scores
	results, err := RDB.ZRangeWithScores(ctx, indexKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	sessions := make([]map[string]interface{}, 0)
	expiredMembers := make([]interface{}, 0)

	for _, z := range results {
		memberData := z.Member.(string)
		// Parse member data: group:model:sessionHash
		var group, model, sessionHash string
		if _, err := fmt.Sscanf(memberData, "%s", &memberData); err == nil {
			// Split by :
			parts := splitMemberData(memberData)
			if len(parts) >= 3 {
				group = parts[0]
				model = parts[1]
				sessionHash = parts[2]
			}
		}

		if sessionHash == "" {
			continue
		}

		// Get TTL and data for this session
		key := GetStickySessionKey(group, model, sessionHash)
		ttl, err := RDB.TTL(ctx, key).Result()
		if err != nil || ttl < 0 {
			// Session expired, mark for removal
			expiredMembers = append(expiredMembers, memberData)
			continue
		}

		// Get session data to retrieve user info
		username := ""
		tokenName := ""
		val, err := RDB.Get(ctx, key).Result()
		if err == nil {
			var data StickySessionData
			if jsonErr := json.Unmarshal([]byte(val), &data); jsonErr == nil {
				username = data.Username
				tokenName = data.TokenName
			}
		}

		sessions = append(sessions, map[string]interface{}{
			"session_hash": sessionHash,
			"group":        group,
			"model":        model,
			"created_at":   int64(z.Score),
			"ttl":          int64(ttl.Seconds()),
			"username":     username,
			"token_name":   tokenName,
		})
	}

	// Clean up expired members
	if len(expiredMembers) > 0 {
		RDB.ZRem(ctx, indexKey, expiredMembers...)
	}

	return sessions, nil
}

// splitMemberData splits the member data string by colon
func splitMemberData(data string) []string {
	result := make([]string, 0)
	current := ""
	for _, c := range data {
		if c == ':' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// RenewStickySessionTTL renews the TTL if remaining time is below threshold
func RenewStickySessionTTL(group, model, sessionHash string, ttlMinutes int) error {
	if !RedisEnabled {
		return nil
	}
	ctx := context.Background()
	key := GetStickySessionKey(group, model, sessionHash)

	// Check remaining TTL
	ttl, err := RDB.TTL(ctx, key).Result()
	if err != nil {
		return err
	}

	// Renew if remaining time is less than half of the configured TTL
	threshold := time.Duration(ttlMinutes/2) * time.Minute
	if ttl > 0 && ttl < threshold {
		newTTL := time.Duration(ttlMinutes) * time.Minute
		return RDB.Expire(ctx, key, newTTL).Err()
	}
	return nil
}

// ReleaseAllChannelStickySessions releases all sticky sessions for a channel
func ReleaseAllChannelStickySessions(channelId int) error {
	if !RedisEnabled {
		return nil
	}
	ctx := context.Background()
	indexKey := GetStickySessionByChannelKey(channelId)

	// Get all members
	members, err := RDB.ZRange(ctx, indexKey, 0, -1).Result()
	if err != nil {
		return err
	}

	if len(members) == 0 {
		return nil
	}

	pipe := RDB.TxPipeline()

	// Delete each session key
	for _, memberData := range members {
		parts := splitMemberData(memberData)
		if len(parts) >= 3 {
			group := parts[0]
			model := parts[1]
			sessionHash := parts[2]
			key := GetStickySessionKey(group, model, sessionHash)
			pipe.Del(ctx, key)
		}
	}

	// Delete the index
	pipe.Del(ctx, indexKey)

	_, err = pipe.Exec(ctx)
	return err
}

// ReleaseStickySessionByChannelAndHash releases a specific sticky session
func ReleaseStickySessionByChannelAndHash(channelId int, sessionHash string) error {
	if !RedisEnabled {
		return nil
	}
	ctx := context.Background()
	indexKey := GetStickySessionByChannelKey(channelId)

	// Find the session in the index
	members, err := RDB.ZRange(ctx, indexKey, 0, -1).Result()
	if err != nil {
		return err
	}

	for _, memberData := range members {
		parts := splitMemberData(memberData)
		if len(parts) >= 3 && parts[2] == sessionHash {
			group := parts[0]
			model := parts[1]
			return DeleteStickySession(group, model, sessionHash, channelId)
		}
	}

	return nil
}

// cleanupExpiredSessions removes expired sessions from the channel index
func cleanupExpiredSessions(ctx context.Context, channelId int) {
	indexKey := GetStickySessionByChannelKey(channelId)

	members, err := RDB.ZRange(ctx, indexKey, 0, -1).Result()
	if err != nil {
		return
	}

	expiredMembers := make([]interface{}, 0)

	for _, memberData := range members {
		parts := splitMemberData(memberData)
		if len(parts) >= 3 {
			group := parts[0]
			model := parts[1]
			sessionHash := parts[2]
			key := GetStickySessionKey(group, model, sessionHash)

			// Check if session still exists
			exists, _ := RDB.Exists(ctx, key).Result()
			if exists == 0 {
				expiredMembers = append(expiredMembers, memberData)
			}
		}
	}

	if len(expiredMembers) > 0 {
		RDB.ZRem(ctx, indexKey, expiredMembers...)
	}
}

// GetStickySessionTTL returns the remaining TTL for a session in seconds
func GetStickySessionTTL(group, model, sessionHash string) int64 {
	if !RedisEnabled {
		return 0
	}
	ctx := context.Background()
	key := GetStickySessionKey(group, model, sessionHash)
	ttl, err := RDB.TTL(ctx, key).Result()
	if err != nil || ttl < 0 {
		return 0
	}
	return int64(ttl.Seconds())
}

// GetSessionChannelUsageKey returns the Redis key for session channel usage tracking
func GetSessionChannelUsageKey(group, model, sessionId string) string {
	return fmt.Sprintf("%s%s:%s:%s", SessionChannelUsagePrefix, group, model, sessionId)
}

// GetSessionChannelUsage retrieves the channel usage history for a session
func GetSessionChannelUsage(group, model, sessionId string) (*SessionChannelUsage, error) {
	if !RedisEnabled {
		return nil, nil
	}
	ctx := context.Background()
	key := GetSessionChannelUsageKey(group, model, sessionId)
	val, err := RDB.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var usage SessionChannelUsage
	if err := json.Unmarshal([]byte(val), &usage); err != nil {
		return nil, err
	}
	return &usage, nil
}

// SetSessionChannelUsage saves the channel usage history for a session
func SetSessionChannelUsage(group, model, sessionId string, channelId int, priority int64) error {
	if !RedisEnabled {
		return nil
	}
	ctx := context.Background()
	key := GetSessionChannelUsageKey(group, model, sessionId)

	usage := SessionChannelUsage{
		ChannelId:  channelId,
		Priority:   priority,
		LastUsedAt: time.Now().Unix(),
	}

	jsonData, err := json.Marshal(usage)
	if err != nil {
		return err
	}

	// TTL 设为 10 分钟，稍大于 5 分钟的免费缓存阈值
	ttl := 10 * time.Minute
	return RDB.Set(ctx, key, string(jsonData), ttl).Err()
}

// CheckChannelSwitchForFreeCache checks if the channel switch qualifies for free cache creation
// Returns: eligible - whether the switch qualifies, prevChannelId - the previous channel ID
// Conditions: switching from lower priority to higher priority channel, and previous usage within 5 minutes
func CheckChannelSwitchForFreeCache(group, model, sessionId string, newChannelId int, newPriority int64) (bool, int) {
	if !RedisEnabled {
		return false, 0
	}

	usage, err := GetSessionChannelUsage(group, model, sessionId)
	if err != nil || usage == nil {
		return false, 0
	}

	// Check if it's a different channel
	if usage.ChannelId == newChannelId {
		return false, 0
	}

	// Check if new channel has higher priority (indicates user was forced to use lower priority before)
	if newPriority <= usage.Priority {
		return false, 0
	}

	// Check if previous channel usage was within 5 minutes
	const cacheValidDuration = 5 * 60 // 5 minutes in seconds
	if time.Now().Unix()-usage.LastUsedAt > cacheValidDuration {
		return false, 0
	}

	return true, usage.ChannelId
}
