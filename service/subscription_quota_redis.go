package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// Redis key patterns for subscription quota tracking
// Key format: subscription:quota:{userSubscriptionId}:{period}:{periodKey}
// Example: subscription:quota:123:daily:2024-01-15
//          subscription:quota:123:weekly:2024-W03
//          subscription:quota:123:total:total (不重置，订阅期内累计)
const (
	SubscriptionQuotaKeyPrefix = "subscription:quota:"
)

// GetSubscriptionQuotaKey generates Redis key for subscription quota tracking
func GetSubscriptionQuotaKey(userSubscriptionId int, period string, periodKey string) string {
	return fmt.Sprintf("%s%d:%s:%s", SubscriptionQuotaKeyPrefix, userSubscriptionId, period, periodKey)
}

// GetCurrentPeriodKeys returns the current period keys for daily, weekly, and total quotas
func GetCurrentPeriodKeys() (daily, weekly, total string) {
	now := GetSingaporeNow()

	// Daily: YYYY-MM-DD
	daily = now.Format("2006-01-02")

	// Weekly: YYYY-WNN (ISO week number)
	year, week := now.ISOWeek()
	weekly = fmt.Sprintf("%d-W%02d", year, week)

	// Total: 固定值，不会变化，不会重置
	total = "total"

	return
}

// GetTTLForPeriod returns the TTL duration for each period type
func GetTTLForPeriod(period string) time.Duration {
	now := GetSingaporeNow()

	switch period {
	case "daily":
		// Until midnight Singapore time
		return GetTTLUntilSingaporeMidnight()

	case "weekly":
		// Until next Monday 00:00 Singapore time
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7
		}
		daysToMonday := 8 - weekday // Days until next Monday
		nextMonday := time.Date(now.Year(), now.Month(), now.Day()+daysToMonday, 0, 0, 0, 0, SingaporeLocation)
		return nextMonday.Sub(now)

	case "total":
		// 总限额不过期，设置一个很长的 TTL（365天），订阅到期时由系统清理
		return 365 * 24 * time.Hour

	default:
		return 24 * time.Hour // Default to 1 day
	}
}

// SubscriptionQuotaRedis handles Redis-based subscription quota operations
type SubscriptionQuotaRedis struct {
	UserSubscriptionId int
	Subscription       *model.Subscription
}

// NewSubscriptionQuotaRedis creates a new SubscriptionQuotaRedis instance
func NewSubscriptionQuotaRedis(userSubscriptionId int, subscription *model.Subscription) *SubscriptionQuotaRedis {
	return &SubscriptionQuotaRedis{
		UserSubscriptionId: userSubscriptionId,
		Subscription:       subscription,
	}
}

// GetQuotaUsed returns the current quota used for each period from Redis
// Returns (dailyUsed, weeklyUsed, totalUsed, error)
func (s *SubscriptionQuotaRedis) GetQuotaUsed() (int, int, int, error) {
	if !common.RedisEnabled {
		return 0, 0, 0, fmt.Errorf("Redis is not enabled")
	}

	ctx := context.Background()
	daily, weekly, total := GetCurrentPeriodKeys()

	// Get all quota values in parallel using pipeline
	pipe := common.RDB.Pipeline()

	dailyCmd := pipe.Get(ctx, GetSubscriptionQuotaKey(s.UserSubscriptionId, "daily", daily))
	weeklyCmd := pipe.Get(ctx, GetSubscriptionQuotaKey(s.UserSubscriptionId, "weekly", weekly))
	totalCmd := pipe.Get(ctx, GetSubscriptionQuotaKey(s.UserSubscriptionId, "total", total))

	_, _ = pipe.Exec(ctx) // Ignore errors, keys may not exist

	dailyUsed, _ := strconv.Atoi(dailyCmd.Val())
	weeklyUsed, _ := strconv.Atoi(weeklyCmd.Val())
	totalUsed, _ := strconv.Atoi(totalCmd.Val())

	return dailyUsed, weeklyUsed, totalUsed, nil
}

// CheckQuotaAvailable checks if there's enough quota available for all periods
// Returns nil if available, error with reason if not
func (s *SubscriptionQuotaRedis) CheckQuotaAvailable(quota int) error {
	dailyUsed, weeklyUsed, totalUsed, err := s.GetQuotaUsed()
	if err != nil {
		return err
	}

	// Check daily limit
	if s.Subscription.DailyQuotaLimit > 0 && dailyUsed+quota > s.Subscription.DailyQuotaLimit {
		return fmt.Errorf("超出每日订阅限额（已用: %d, 需要: %d, 限额: %d）",
			dailyUsed, quota, s.Subscription.DailyQuotaLimit)
	}

	// Check weekly limit
	if s.Subscription.WeeklyQuotaLimit > 0 && weeklyUsed+quota > s.Subscription.WeeklyQuotaLimit {
		return fmt.Errorf("超出每周订阅限额（已用: %d, 需要: %d, 限额: %d）",
			weeklyUsed, quota, s.Subscription.WeeklyQuotaLimit)
	}

	// Check total limit
	if s.Subscription.TotalQuotaLimit > 0 && totalUsed+quota > s.Subscription.TotalQuotaLimit {
		return fmt.Errorf("超出订阅总限额（已用: %d, 需要: %d, 限额: %d）",
			totalUsed, quota, s.Subscription.TotalQuotaLimit)
	}

	return nil
}

// ConsumeQuota atomically consumes quota using Redis INCR with TTL
// This is the core function that replaces database row locks
func (s *SubscriptionQuotaRedis) ConsumeQuota(quota int) error {
	if !common.RedisEnabled {
		return fmt.Errorf("Redis is not enabled")
	}

	if quota <= 0 {
		return nil
	}

	// First check if quota is available
	if err := s.CheckQuotaAvailable(quota); err != nil {
		return err
	}

	ctx := context.Background()
	daily, weekly, total := GetCurrentPeriodKeys()

	// Use Lua script for atomic check-and-increment
	// This ensures we don't exceed limits between check and consume
	script := `
		local dailyKey = KEYS[1]
		local weeklyKey = KEYS[2]
		local totalKey = KEYS[3]
		local quota = tonumber(ARGV[1])
		local dailyLimit = tonumber(ARGV[2])
		local weeklyLimit = tonumber(ARGV[3])
		local totalLimit = tonumber(ARGV[4])
		local dailyTTL = tonumber(ARGV[5])
		local weeklyTTL = tonumber(ARGV[6])
		local totalTTL = tonumber(ARGV[7])

		-- Get current values (0 if not exists)
		local dailyUsed = tonumber(redis.call('GET', dailyKey) or '0')
		local weeklyUsed = tonumber(redis.call('GET', weeklyKey) or '0')
		local totalUsed = tonumber(redis.call('GET', totalKey) or '0')

		-- Check limits (0 means no limit)
		if dailyLimit > 0 and dailyUsed + quota > dailyLimit then
			return {'daily', dailyUsed, dailyLimit}
		end
		if weeklyLimit > 0 and weeklyUsed + quota > weeklyLimit then
			return {'weekly', weeklyUsed, weeklyLimit}
		end
		if totalLimit > 0 and totalUsed + quota > totalLimit then
			return {'total', totalUsed, totalLimit}
		end

		-- Increment all counters atomically
		local newDaily = redis.call('INCRBY', dailyKey, quota)
		local newWeekly = redis.call('INCRBY', weeklyKey, quota)
		local newTotal = redis.call('INCRBY', totalKey, quota)

		-- Set TTL if key was just created (TTL returns -1 for new keys)
		if redis.call('TTL', dailyKey) == -1 then
			redis.call('EXPIRE', dailyKey, dailyTTL)
		end
		if redis.call('TTL', weeklyKey) == -1 then
			redis.call('EXPIRE', weeklyKey, weeklyTTL)
		end
		if redis.call('TTL', totalKey) == -1 then
			redis.call('EXPIRE', totalKey, totalTTL)
		end

		return {'ok', newDaily, newWeekly, newTotal}
	`

	dailyKey := GetSubscriptionQuotaKey(s.UserSubscriptionId, "daily", daily)
	weeklyKey := GetSubscriptionQuotaKey(s.UserSubscriptionId, "weekly", weekly)
	totalKey := GetSubscriptionQuotaKey(s.UserSubscriptionId, "total", total)

	dailyTTL := int64(GetTTLForPeriod("daily").Seconds())
	weeklyTTL := int64(GetTTLForPeriod("weekly").Seconds())
	totalTTL := int64(GetTTLForPeriod("total").Seconds())

	result, err := common.RDB.Eval(ctx, script,
		[]string{dailyKey, weeklyKey, totalKey},
		quota,
		s.Subscription.DailyQuotaLimit,
		s.Subscription.WeeklyQuotaLimit,
		s.Subscription.TotalQuotaLimit,
		dailyTTL,
		weeklyTTL,
		totalTTL,
	).Result()

	if err != nil {
		return fmt.Errorf("Redis Lua script error: %v", err)
	}

	// Parse result
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) == 0 {
		return fmt.Errorf("unexpected Redis result format")
	}

	status := resultSlice[0].(string)
	if status != "ok" {
		used := int64(0)
		limit := int64(0)
		if len(resultSlice) > 1 {
			used, _ = resultSlice[1].(int64)
		}
		if len(resultSlice) > 2 {
			limit, _ = resultSlice[2].(int64)
		}
		return fmt.Errorf("超出%s订阅限额（已用: %d, 限额: %d）", status, used, limit)
	}

	return nil
}

// ReturnQuota returns quota back when request fails
// Uses Redis DECRBY for atomic decrement
func (s *SubscriptionQuotaRedis) ReturnQuota(quota int) error {
	if !common.RedisEnabled {
		return fmt.Errorf("Redis is not enabled")
	}

	if quota <= 0 {
		return nil
	}

	ctx := context.Background()
	daily, weekly, total := GetCurrentPeriodKeys()

	// Use Lua script for atomic decrement with floor at 0
	script := `
		local dailyKey = KEYS[1]
		local weeklyKey = KEYS[2]
		local totalKey = KEYS[3]
		local quota = tonumber(ARGV[1])

		-- Decrement but don't go below 0
		local dailyUsed = tonumber(redis.call('GET', dailyKey) or '0')
		local weeklyUsed = tonumber(redis.call('GET', weeklyKey) or '0')
		local totalUsed = tonumber(redis.call('GET', totalKey) or '0')

		local newDaily = math.max(0, dailyUsed - quota)
		local newWeekly = math.max(0, weeklyUsed - quota)
		local newTotal = math.max(0, totalUsed - quota)

		-- Only update if key exists (has TTL)
		if redis.call('TTL', dailyKey) > 0 then
			redis.call('SET', dailyKey, newDaily, 'KEEPTTL')
		end
		if redis.call('TTL', weeklyKey) > 0 then
			redis.call('SET', weeklyKey, newWeekly, 'KEEPTTL')
		end
		if redis.call('TTL', totalKey) > 0 then
			redis.call('SET', totalKey, newTotal, 'KEEPTTL')
		end

		return {newDaily, newWeekly, newTotal}
	`

	dailyKey := GetSubscriptionQuotaKey(s.UserSubscriptionId, "daily", daily)
	weeklyKey := GetSubscriptionQuotaKey(s.UserSubscriptionId, "weekly", weekly)
	totalKey := GetSubscriptionQuotaKey(s.UserSubscriptionId, "total", total)

	_, err := common.RDB.Eval(ctx, script,
		[]string{dailyKey, weeklyKey, totalKey},
		quota,
	).Result()

	if err != nil {
		return fmt.Errorf("failed to return subscription quota: %v", err)
	}

	return nil
}

// GetQuotaStatus returns detailed quota status for display
type SubscriptionQuotaStatus struct {
	DailyUsed      int   `json:"daily_used"`
	DailyLimit     int   `json:"daily_limit"`
	DailyRemaining int   `json:"daily_remaining"`
	DailyExpiresAt int64 `json:"daily_expires_at"`

	WeeklyUsed      int   `json:"weekly_used"`
	WeeklyLimit     int   `json:"weekly_limit"`
	WeeklyRemaining int   `json:"weekly_remaining"`
	WeeklyExpiresAt int64 `json:"weekly_expires_at"`

	TotalUsed      int   `json:"total_used"`
	TotalLimit     int   `json:"total_limit"`
	TotalRemaining int   `json:"total_remaining"`
}

func (s *SubscriptionQuotaRedis) GetQuotaStatus() (*SubscriptionQuotaStatus, error) {
	dailyUsed, weeklyUsed, totalUsed, err := s.GetQuotaUsed()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	status := &SubscriptionQuotaStatus{
		DailyUsed:   dailyUsed,
		DailyLimit:  s.Subscription.DailyQuotaLimit,
		WeeklyUsed:  weeklyUsed,
		WeeklyLimit: s.Subscription.WeeklyQuotaLimit,
		TotalUsed:   totalUsed,
		TotalLimit:  s.Subscription.TotalQuotaLimit,
	}

	// Calculate remaining
	if s.Subscription.DailyQuotaLimit > 0 {
		status.DailyRemaining = s.Subscription.DailyQuotaLimit - dailyUsed
		if status.DailyRemaining < 0 {
			status.DailyRemaining = 0
		}
	}
	if s.Subscription.WeeklyQuotaLimit > 0 {
		status.WeeklyRemaining = s.Subscription.WeeklyQuotaLimit - weeklyUsed
		if status.WeeklyRemaining < 0 {
			status.WeeklyRemaining = 0
		}
	}
	if s.Subscription.TotalQuotaLimit > 0 {
		status.TotalRemaining = s.Subscription.TotalQuotaLimit - totalUsed
		if status.TotalRemaining < 0 {
			status.TotalRemaining = 0
		}
	}

	// Calculate expiration times
	status.DailyExpiresAt = now.Add(GetTTLForPeriod("daily")).Unix()
	status.WeeklyExpiresAt = now.Add(GetTTLForPeriod("weekly")).Unix()

	return status, nil
}
