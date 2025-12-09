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
	TotalQuotaLimit    int    `json:"total_quota_limit" gorm:"column:monthly_quota_limit;default:0"` // 总限额（订阅期内不重置）
	AllowedGroups      string `json:"allowed_groups" gorm:"type:text;not null"` // JSON array
	DurationDays       int    `json:"duration_days" gorm:"default:30"`
	Status             int    `json:"status" gorm:"default:1"`
	CreatedTime        int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime        int64  `json:"updated_time" gorm:"bigint"`
}

// UserSubscription 用户订阅
// 注意：用量数据存储在 Redis 中，不在数据库
type UserSubscription struct {
	Id             int   `json:"id"`
	UserId         int   `json:"user_id" gorm:"index"`
	SubscriptionId int   `json:"subscription_id" gorm:"index"`
	RedemptionId   *int  `json:"redemption_id"`
	Status         int   `json:"status" gorm:"default:1;index"`
	StartTime      int64 `json:"start_time" gorm:"bigint"`
	ExpireTime     int64 `json:"expire_time" gorm:"bigint;index"`
	CreatedTime    int64 `json:"created_time" gorm:"bigint"`
	UpdatedTime    int64 `json:"updated_time" gorm:"bigint"`

	// 关联数据（不存数据库，用于 API 返回）
	SubscriptionInfo *Subscription `json:"subscription_info,omitempty" gorm:"-"`
	// Redis 用量数据（不存数据库，用于 API 返回）
	DailyQuotaUsed  int `json:"daily_quota_used" gorm:"-"`
	WeeklyQuotaUsed int `json:"weekly_quota_used" gorm:"-"`
	TotalQuotaUsed  int `json:"total_quota_used" gorm:"-"` // 总用量（订阅期内累计）
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

