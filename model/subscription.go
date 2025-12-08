package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/bytedance/gopkg/util/gopool"
)

// 套餐状态常量
const (
	SubscriptionStatusEnabled  = 1
	SubscriptionStatusDisabled = 2
)

// 用户订阅状态常量
const (
	UserSubscriptionStatusActive   = 1 // 激活
	UserSubscriptionStatusExpired  = 2 // 已过期
	UserSubscriptionStatusCanceled = 3 // 已取消
	UserSubscriptionStatusReplaced = 4 // 已被新订阅替换
)

// Subscription 订阅套餐
type Subscription struct {
	Id                 int    `json:"id"`
	Name               string `json:"name" gorm:"type:varchar(64);not null"`
	Description        string `json:"description" gorm:"type:text"`
	DailyQuotaLimit    int    `json:"daily_quota_limit" gorm:"default:0"`
	WeeklyQuotaLimit   int    `json:"weekly_quota_limit" gorm:"default:0"`
	MonthlyQuotaLimit  int    `json:"monthly_quota_limit" gorm:"default:0"`
	AllowedGroups      string `json:"allowed_groups" gorm:"type:text;not null"` // JSON array
	DurationDays       int    `json:"duration_days" gorm:"default:30"`
	Status             int    `json:"status" gorm:"default:1"`
	CreatedTime        int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime        int64  `json:"updated_time" gorm:"bigint"`
}

// UserSubscription 用户订阅
type UserSubscription struct {
	Id               int    `json:"id"`
	UserId           int    `json:"user_id" gorm:"index"`
	SubscriptionId   int    `json:"subscription_id" gorm:"index"`
	RedemptionId     *int   `json:"redemption_id"`
	Status           int    `json:"status" gorm:"default:1;index"`
	StartTime        int64  `json:"start_time" gorm:"bigint"`
	ExpireTime       int64  `json:"expire_time" gorm:"bigint;index"`
	DailyQuotaUsed   int    `json:"daily_quota_used" gorm:"default:0"`
	WeeklyQuotaUsed  int    `json:"weekly_quota_used" gorm:"default:0"`
	MonthlyQuotaUsed int    `json:"monthly_quota_used" gorm:"default:0"`
	DailyResetTime   int64  `json:"daily_reset_time" gorm:"bigint"`
	WeeklyResetTime  int64  `json:"weekly_reset_time" gorm:"bigint"`
	MonthlyResetTime int64  `json:"monthly_reset_time" gorm:"bigint"`
	CreatedTime      int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime      int64  `json:"updated_time" gorm:"bigint"`

	// 关联数据（不存数据库）
	SubscriptionInfo *Subscription `json:"subscription_info,omitempty" gorm:"-"`
}

// SubscriptionLog 订阅额度使用日志
type SubscriptionLog struct {
	Id                 int64  `json:"id"`
	UserSubscriptionId int    `json:"user_subscription_id" gorm:"index"`
	UserId             int    `json:"user_id" gorm:"index"`
	QuotaUsed          int    `json:"quota_used"`
	ChannelId          int    `json:"channel_id"`
	ModelName          string `json:"model_name" gorm:"type:varchar(128)"`
	TokenName          string `json:"token_name" gorm:"type:varchar(128)"`
	PromptTokens       int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens   int    `json:"completion_tokens" gorm:"default:0"`
	CreatedTime        int64  `json:"created_time" gorm:"bigint;index"`
}

// ========== Subscription 方法 ==========

func (s *Subscription) Insert() error {
	s.CreatedTime = common.GetTimestamp()
	s.UpdatedTime = s.CreatedTime
	return DB.Create(s).Error
}

func (s *Subscription) Update() error {
	s.UpdatedTime = common.GetTimestamp()
	return DB.Model(s).Updates(s).Error
}

func (s *Subscription) Delete() error {
	return DB.Delete(s).Error
}

func GetSubscriptionById(id int) (*Subscription, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	var sub Subscription
	err := DB.First(&sub, id).Error
	return &sub, err
}

func GetAllSubscriptions(startIdx int, num int) ([]*Subscription, int64, error) {
	var subs []*Subscription
	var total int64

	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err := tx.Model(&Subscription{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&subs).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return subs, total, nil
}

func GetEnabledSubscriptions() ([]*Subscription, error) {
	var subs []*Subscription
	err := DB.Where("status = ?", SubscriptionStatusEnabled).Find(&subs).Error
	return subs, err
}

// IsGroupAllowed 检查分组是否在允许列表中
func (s *Subscription) IsGroupAllowed(group string) bool {
	var groups []string
	if err := json.Unmarshal([]byte(s.AllowedGroups), &groups); err != nil {
		return false
	}
	for _, g := range groups {
		if g == group {
			return true
		}
	}
	return false
}

// GetAllowedGroupsList 获取允许的分组列表
func (s *Subscription) GetAllowedGroupsList() []string {
	var groups []string
	json.Unmarshal([]byte(s.AllowedGroups), &groups)
	return groups
}

// ========== UserSubscription 方法 ==========

func (us *UserSubscription) Insert() error {
	us.CreatedTime = common.GetTimestamp()
	us.UpdatedTime = us.CreatedTime
	err := DB.Create(us).Error
	if err == nil {
		// 清除缓存
		gopool.Go(func() {
			CacheDeleteUserSubscription(us.UserId)
		})
	}
	return err
}

func (us *UserSubscription) Update() error {
	us.UpdatedTime = common.GetTimestamp()
	err := DB.Save(us).Error
	if err == nil {
		// 清除缓存
		gopool.Go(func() {
			CacheDeleteUserSubscription(us.UserId)
		})
	}
	return err
}

// CheckAndResetQuota 检查并重置过期的额度
func (us *UserSubscription) CheckAndResetQuota() bool {
	now := time.Now()
	needsUpdate := false

	// 每日重置（当天0点）
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	if us.DailyResetTime < todayStart {
		us.DailyQuotaUsed = 0
		us.DailyResetTime = todayStart
		needsUpdate = true
	}

	// 每周重置（周一0点）
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	daysToMonday := weekday - 1
	weekStart := time.Date(now.Year(), now.Month(), now.Day()-daysToMonday, 0, 0, 0, 0, now.Location()).Unix()
	if us.WeeklyResetTime < weekStart {
		us.WeeklyQuotaUsed = 0
		us.WeeklyResetTime = weekStart
		needsUpdate = true
	}

	// 每月重置（1号0点）
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Unix()
	if us.MonthlyResetTime < monthStart {
		us.MonthlyQuotaUsed = 0
		us.MonthlyResetTime = monthStart
		needsUpdate = true
	}

	return needsUpdate
}

// ConsumeQuota 消费额度（带事务和锁）
func (us *UserSubscription) ConsumeQuota(quota int, subscription *Subscription) error {
	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 加锁查询
	err := tx.Set("gorm:query_option", "FOR UPDATE").First(us, us.Id).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	// 检查并重置
	if us.CheckAndResetQuota() {
		// 重置时间已更新，需要保存
	}

	// 检查三个维度限额
	if subscription.DailyQuotaLimit > 0 && us.DailyQuotaUsed+quota > subscription.DailyQuotaLimit {
		tx.Rollback()
		return errors.New("超出每日订阅限额")
	}
	if subscription.WeeklyQuotaLimit > 0 && us.WeeklyQuotaUsed+quota > subscription.WeeklyQuotaLimit {
		tx.Rollback()
		return errors.New("超出每周订阅限额")
	}
	if subscription.MonthlyQuotaLimit > 0 && us.MonthlyQuotaUsed+quota > subscription.MonthlyQuotaLimit {
		tx.Rollback()
		return errors.New("超出每月订阅限额")
	}

	// 消费额度
	us.DailyQuotaUsed += quota
	us.WeeklyQuotaUsed += quota
	us.MonthlyQuotaUsed += quota
	us.UpdatedTime = common.GetTimestamp()

	err = tx.Save(us).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	err = tx.Commit().Error
	if err != nil {
		return err
	}

	// 清除缓存
	gopool.Go(func() {
		CacheDeleteUserSubscription(us.UserId)
		// 更新 Redis 缓存
		if common.RedisEnabled {
			CacheIncrSubscriptionQuota(us.Id, "daily", quota)
			CacheIncrSubscriptionQuota(us.Id, "weekly", quota)
			CacheIncrSubscriptionQuota(us.Id, "monthly", quota)
		}
	})

	return nil
}

// ReturnQuota 返还额度（请求失败时退还预扣的额度）
func (us *UserSubscription) ReturnQuota(quota int) error {
	if quota <= 0 {
		return nil
	}

	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 加锁查询
	err := tx.Set("gorm:query_option", "FOR UPDATE").First(us, us.Id).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	// 返还额度（减少已使用量，但不能小于0）
	us.DailyQuotaUsed -= quota
	if us.DailyQuotaUsed < 0 {
		us.DailyQuotaUsed = 0
	}
	us.WeeklyQuotaUsed -= quota
	if us.WeeklyQuotaUsed < 0 {
		us.WeeklyQuotaUsed = 0
	}
	us.MonthlyQuotaUsed -= quota
	if us.MonthlyQuotaUsed < 0 {
		us.MonthlyQuotaUsed = 0
	}
	us.UpdatedTime = common.GetTimestamp()

	err = tx.Save(us).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	err = tx.Commit().Error
	if err != nil {
		return err
	}

	// 清除缓存
	gopool.Go(func() {
		CacheDeleteUserSubscription(us.UserId)
		// 更新 Redis 缓存（负数表示返还）
		if common.RedisEnabled {
			CacheIncrSubscriptionQuota(us.Id, "daily", -quota)
			CacheIncrSubscriptionQuota(us.Id, "weekly", -quota)
			CacheIncrSubscriptionQuota(us.Id, "monthly", -quota)
		}
	})

	return nil
}

// GetActiveUserSubscription 获取用户激活的订阅（支持指定分组）
func GetActiveUserSubscription(userId int, group string) (*UserSubscription, *Subscription, error) {
	// 先尝试从缓存获取
	cached, err := CacheGetUserSubscription(userId)
	if err == nil && cached != nil {
		sub, _ := GetSubscriptionById(cached.SubscriptionId)
		if sub != nil && sub.IsGroupAllowed(group) && cached.Status == UserSubscriptionStatusActive {
			// 检查是否过期
			now := common.GetTimestamp()
			if cached.ExpireTime > now {
				return cached, sub, nil
			}
		}
	}

	// 从数据库查询
	var us UserSubscription
	now := common.GetTimestamp()
	err = DB.Where("user_id = ? AND status = ? AND expire_time > ?",
		userId, UserSubscriptionStatusActive, now).First(&us).Error
	if err != nil {
		return nil, nil, err
	}

	// 检查是否过期
	if us.ExpireTime <= now {
		us.Status = UserSubscriptionStatusExpired
		us.Update()
		return nil, nil, errors.New("订阅已过期")
	}

	// 获取套餐信息
	sub, err := GetSubscriptionById(us.SubscriptionId)
	if err != nil {
		return nil, nil, err
	}

	// 检查分组
	if !sub.IsGroupAllowed(group) {
		return nil, nil, errors.New("该订阅不支持当前分组")
	}

	// 缓存
	gopool.Go(func() {
		CacheSetUserSubscription(userId, &us)
	})

	return &us, sub, nil
}

// GetActiveUserSubscriptionNoGroup 获取用户激活的订阅（不检查分组）
// 订阅额度作为用户余额的替代，不应该区分分组
func GetActiveUserSubscriptionNoGroup(userId int) (*UserSubscription, *Subscription, error) {
	// 先尝试从缓存获取
	cached, err := CacheGetUserSubscription(userId)
	if err == nil && cached != nil && cached.Status == UserSubscriptionStatusActive {
		now := common.GetTimestamp()
		if cached.ExpireTime > now {
			sub, _ := GetSubscriptionById(cached.SubscriptionId)
			if sub != nil {
				return cached, sub, nil
			}
		}
	}

	// 从数据库查询
	var us UserSubscription
	now := common.GetTimestamp()
	err = DB.Where("user_id = ? AND status = ? AND expire_time > ?",
		userId, UserSubscriptionStatusActive, now).First(&us).Error
	if err != nil {
		return nil, nil, err
	}

	// 检查是否过期
	if us.ExpireTime <= now {
		us.Status = UserSubscriptionStatusExpired
		us.Update()
		return nil, nil, errors.New("订阅已过期")
	}

	// 获取套餐信息
	sub, err := GetSubscriptionById(us.SubscriptionId)
	if err != nil {
		return nil, nil, err
	}

	// 缓存
	gopool.Go(func() {
		CacheSetUserSubscription(userId, &us)
	})

	return &us, sub, nil
}

// GetUserSubscriptions 获取用户所有订阅
func GetUserSubscriptions(userId int, startIdx int, num int) ([]*UserSubscription, int64, error) {
	var list []*UserSubscription
	var total int64

	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err := tx.Model(&UserSubscription{}).Where("user_id = ?", userId).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	err = tx.Where("user_id = ?", userId).Order("id DESC").Limit(num).Offset(startIdx).Find(&list).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	// 加载套餐信息，并检查过期状态
	now := common.GetTimestamp()
	for _, us := range list {
		sub, _ := GetSubscriptionById(us.SubscriptionId)
		us.SubscriptionInfo = sub

		// 检查是否过期：如果状态是激活但已过期，更新状态
		if us.Status == UserSubscriptionStatusActive && us.ExpireTime <= now {
			us.Status = UserSubscriptionStatusExpired
			// 异步更新数据库
			gopool.Go(func() {
				DB.Model(&UserSubscription{}).Where("id = ?", us.Id).Update("status", UserSubscriptionStatusExpired)
				CacheDeleteUserSubscription(userId)
			})
		}
	}

	return list, total, nil
}

// ExpireSubscriptions 定时任务：检查并过期订阅
func ExpireSubscriptions() error {
	now := common.GetTimestamp()
	result := DB.Model(&UserSubscription{}).
		Where("status = ? AND expire_time <= ?", UserSubscriptionStatusActive, now).
		Update("status", UserSubscriptionStatusExpired)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		common.SysLog(fmt.Sprintf("过期了 %d 个订阅", result.RowsAffected))
	}

	return nil
}

// ========== SubscriptionLog 方法 ==========

func RecordSubscriptionLog(log *SubscriptionLog) error {
	log.CreatedTime = common.GetTimestamp()
	return DB.Create(log).Error
}

func GetSubscriptionLogs(userId int, startIdx, num int) ([]*SubscriptionLog, int64, error) {
	var logs []*SubscriptionLog
	var total int64

	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err := tx.Model(&SubscriptionLog{}).Where("user_id = ?", userId).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	err = tx.Where("user_id = ?", userId).Order("id DESC").Offset(startIdx).Limit(num).Find(&logs).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return logs, total, err
}

// ========== 辅助函数 ==========

func getTodayStart() int64 {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
}

func getWeekStart() int64 {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	daysToMonday := weekday - 1
	return time.Date(now.Year(), now.Month(), now.Day()-daysToMonday, 0, 0, 0, 0, now.Location()).Unix()
}

func getMonthStart() int64 {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Unix()
}

// ========== Redis 缓存函数 ==========

// CacheSetUserSubscription 缓存用户订阅信息
func CacheSetUserSubscription(userId int, us *UserSubscription) error {
	if !common.RedisEnabled {
		return nil
	}
	key := fmt.Sprintf("user_subscription:%d", userId)
	data, err := json.Marshal(us)
	if err != nil {
		return err
	}
	return common.RDB.Set(context.Background(), key, data, 30*time.Minute).Err()
}

// CacheGetUserSubscription 获取缓存的用户订阅
func CacheGetUserSubscription(userId int) (*UserSubscription, error) {
	if !common.RedisEnabled {
		return nil, errors.New("redis not enabled")
	}
	key := fmt.Sprintf("user_subscription:%d", userId)
	data, err := common.RDB.Get(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}
	var us UserSubscription
	err = json.Unmarshal([]byte(data), &us)
	return &us, err
}

// CacheDeleteUserSubscription 删除用户订阅缓存
func CacheDeleteUserSubscription(userId int) error {
	if !common.RedisEnabled {
		return nil
	}
	key := fmt.Sprintf("user_subscription:%d", userId)
	return common.RDB.Del(context.Background(), key).Err()
}

// CacheIncrSubscriptionQuota 原子性增加订阅额度使用量（Redis HINCRBY）
func CacheIncrSubscriptionQuota(userSubscriptionId int, quotaType string, amount int) error {
	if !common.RedisEnabled {
		return nil
	}
	key := fmt.Sprintf("subscription_quota:%d", userSubscriptionId)
	field := fmt.Sprintf("%s_used", quotaType) // daily_used, weekly_used, monthly_used
	return common.RDB.HIncrBy(context.Background(), key, field, int64(amount)).Err()
}
