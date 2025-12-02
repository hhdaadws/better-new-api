package service

import (
	"context"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	// Redis key prefixes for check-in
	CheckinQuotaKeyPrefix  = "checkin:quota:"  // 用户当日签到额度
	CheckinRecordKeyPrefix = "checkin:record:" // 用户当日签到记录
)

// Singapore timezone (UTC+8)
var SingaporeLocation *time.Location

func init() {
	var err error
	SingaporeLocation, err = time.LoadLocation("Asia/Singapore")
	if err != nil {
		// Fallback to fixed offset if timezone data not available
		SingaporeLocation = time.FixedZone("SGT", 8*60*60)
	}
}

// GetSingaporeNow returns current time in Singapore timezone
func GetSingaporeNow() time.Time {
	return time.Now().In(SingaporeLocation)
}

// GetSingaporeDate returns current date string in Singapore timezone (YYYY-MM-DD)
func GetSingaporeDate() string {
	return GetSingaporeNow().Format("2006-01-02")
}

// GetTTLUntilSingaporeMidnight returns duration until midnight in Singapore timezone
func GetTTLUntilSingaporeMidnight() time.Duration {
	now := GetSingaporeNow()
	// Calculate midnight of next day
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, SingaporeLocation)
	return midnight.Sub(now)
}

// GetCheckinQuotaKey returns Redis key for user's daily check-in quota
func GetCheckinQuotaKey(userId int) string {
	return fmt.Sprintf("%s%d:%s", CheckinQuotaKeyPrefix, userId, GetSingaporeDate())
}

// GetCheckinRecordKey returns Redis key for user's daily check-in record
func GetCheckinRecordKey(userId int) string {
	return fmt.Sprintf("%s%d:%s", CheckinRecordKeyPrefix, userId, GetSingaporeDate())
}

// CheckinStatus represents the check-in status for a user
type CheckinStatus struct {
	HasCheckedIn   bool  `json:"has_checked_in"`   // 今日是否已签到
	QuotaRemaining int   `json:"quota_remaining"`  // 剩余签到额度
	QuotaUsed      int   `json:"quota_used"`       // 已使用签到额度
	QuotaTotal     int   `json:"quota_total"`      // 签到获得的总额度
	ExpiresAt      int64 `json:"expires_at"`       // 过期时间戳 (UTC)
}

// GetCheckinStatus returns the current check-in status for a user
func GetCheckinStatus(userId int) (*CheckinStatus, error) {
	if !common.RedisEnabled {
		return nil, fmt.Errorf("Redis is not enabled")
	}

	ctx := context.Background()
	status := &CheckinStatus{}

	// Check if user has checked in today
	recordKey := GetCheckinRecordKey(userId)
	_, err := common.RDB.Get(ctx, recordKey).Result()
	if err == nil {
		status.HasCheckedIn = true
	}

	// Get remaining quota
	quotaKey := GetCheckinQuotaKey(userId)
	quotaStr, err := common.RDB.Get(ctx, quotaKey).Result()
	if err == nil {
		fmt.Sscanf(quotaStr, "%d", &status.QuotaRemaining)
	}

	// Get total quota from record (stored as value)
	if status.HasCheckedIn {
		totalStr, err := common.RDB.Get(ctx, recordKey).Result()
		if err == nil {
			fmt.Sscanf(totalStr, "%d", &status.QuotaTotal)
		}
		status.QuotaUsed = status.QuotaTotal - status.QuotaRemaining
	}

	// Calculate expiration time
	ttl := GetTTLUntilSingaporeMidnight()
	status.ExpiresAt = time.Now().Add(ttl).Unix()

	return status, nil
}

// PerformCheckin performs daily check-in for a user and grants quota
func PerformCheckin(userId int, quotaAmount int) (*CheckinStatus, error) {
	if !common.RedisEnabled {
		return nil, fmt.Errorf("Redis is not enabled")
	}

	ctx := context.Background()
	recordKey := GetCheckinRecordKey(userId)

	// Check if already checked in today
	exists, err := common.RDB.Exists(ctx, recordKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to check check-in status: %v", err)
	}
	if exists > 0 {
		return nil, fmt.Errorf("今日已签到，请明天再来")
	}

	// Calculate TTL until Singapore midnight
	ttl := GetTTLUntilSingaporeMidnight()

	// Set check-in record (store total quota as value)
	err = common.RDB.Set(ctx, recordKey, fmt.Sprintf("%d", quotaAmount), ttl).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to record check-in: %v", err)
	}

	// Set check-in quota
	quotaKey := GetCheckinQuotaKey(userId)
	err = common.RDB.Set(ctx, quotaKey, fmt.Sprintf("%d", quotaAmount), ttl).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to set check-in quota: %v", err)
	}

	return &CheckinStatus{
		HasCheckedIn:   true,
		QuotaRemaining: quotaAmount,
		QuotaUsed:      0,
		QuotaTotal:     quotaAmount,
		ExpiresAt:      time.Now().Add(ttl).Unix(),
	}, nil
}

// GetCheckinQuota returns the remaining check-in quota for a user
func GetCheckinQuota(userId int) (int, error) {
	if !common.RedisEnabled {
		return 0, nil
	}

	ctx := context.Background()
	quotaKey := GetCheckinQuotaKey(userId)

	quotaStr, err := common.RDB.Get(ctx, quotaKey).Result()
	if err != nil {
		return 0, nil // No quota or key doesn't exist
	}

	var quota int
	fmt.Sscanf(quotaStr, "%d", &quota)
	return quota, nil
}

// ConsumeCheckinQuota consumes check-in quota for a user
// Returns the amount actually consumed (may be less than requested if not enough quota)
func ConsumeCheckinQuota(userId int, amount int) (int, error) {
	if !common.RedisEnabled {
		return 0, nil
	}

	if amount <= 0 {
		return 0, nil
	}

	ctx := context.Background()
	quotaKey := GetCheckinQuotaKey(userId)

	// Get current quota
	quotaStr, err := common.RDB.Get(ctx, quotaKey).Result()
	if err != nil {
		return 0, nil // No quota available
	}

	var currentQuota int
	fmt.Sscanf(quotaStr, "%d", &currentQuota)

	if currentQuota <= 0 {
		return 0, nil
	}

	// Calculate actual consumption
	consumed := amount
	if consumed > currentQuota {
		consumed = currentQuota
	}

	// Get remaining TTL
	ttl, err := common.RDB.TTL(ctx, quotaKey).Result()
	if err != nil || ttl <= 0 {
		return 0, nil
	}

	// Decrease quota
	newQuota := currentQuota - consumed
	err = common.RDB.Set(ctx, quotaKey, fmt.Sprintf("%d", newQuota), ttl).Err()
	if err != nil {
		return 0, fmt.Errorf("failed to consume check-in quota: %v", err)
	}

	return consumed, nil
}

// ReturnCheckinQuota returns quota back to check-in pool (for failed requests)
func ReturnCheckinQuota(userId int, amount int) error {
	if !common.RedisEnabled {
		return nil
	}

	if amount <= 0 {
		return nil
	}

	ctx := context.Background()
	quotaKey := GetCheckinQuotaKey(userId)

	// Check if key exists
	exists, err := common.RDB.Exists(ctx, quotaKey).Result()
	if err != nil || exists == 0 {
		return nil // Key doesn't exist, nothing to return
	}

	// Get remaining TTL
	ttl, err := common.RDB.TTL(ctx, quotaKey).Result()
	if err != nil || ttl <= 0 {
		return nil
	}

	// Increment quota
	_, err = common.RDB.IncrBy(ctx, quotaKey, int64(amount)).Result()
	if err != nil {
		return fmt.Errorf("failed to return check-in quota: %v", err)
	}

	return nil
}
