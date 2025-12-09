package service

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// TryConsumeSubscriptionQuota 尝试使用订阅额度
// 订阅额度是用户余额的替代品，不区分分组
// 返回值：(是否使用了订阅额度, 错误信息)
func TryConsumeSubscriptionQuota(relayInfo *relaycommon.RelayInfo, quota int) (bool, error) {
	if !common.RedisEnabled {
		return false, fmt.Errorf("Redis is required for subscription quota")
	}

	// 获取用户激活的订阅（不区分分组）
	userSub, subscription, err := model.GetActiveUserSubscriptionNoGroup(relayInfo.UserId)
	if err != nil {
		// 没有激活的订阅，返回 false
		return false, nil
	}

	// 使用 Redis 消费额度
	quotaRedis := NewSubscriptionQuotaRedis(userSub.Id, subscription)
	err = quotaRedis.ConsumeQuota(quota)
	if err != nil {
		// 订阅额度不足，返回 false，让系统降级到普通用户余额
		return false, nil
	}

	// 记录日志（异步）
	go func() {
		log := &model.SubscriptionLog{
			UserSubscriptionId: userSub.Id,
			UserId:             relayInfo.UserId,
			QuotaUsed:          quota,
			ChannelId:          relayInfo.ChannelId,
			ModelName:          relayInfo.OriginModelName,
			TokenName:          relayInfo.TokenKey,
		}
		model.RecordSubscriptionLog(log)
	}()

	return true, nil
}

// TryPreConsumeSubscriptionQuota 尝试预扣订阅额度
// 返回值：(是否预扣成功, 错误信息)
func TryPreConsumeSubscriptionQuota(userId int, quota int) (bool, error) {
	if !common.RedisEnabled {
		return false, fmt.Errorf("Redis is required for subscription quota")
	}

	// 获取用户激活的订阅（不区分分组）
	userSub, subscription, err := model.GetActiveUserSubscriptionNoGroup(userId)
	if err != nil {
		// 没有激活的订阅
		return false, nil
	}

	// 使用 Redis 预扣额度
	quotaRedis := NewSubscriptionQuotaRedis(userSub.Id, subscription)
	err = quotaRedis.ConsumeQuota(quota)
	if err != nil {
		// 订阅额度不足
		return false, nil
	}

	return true, nil
}

// ReturnSubscriptionQuota 返还订阅额度
// 当请求失败需要退还预扣的订阅额度时调用
func ReturnSubscriptionQuota(userId int, quota int) error {
	if quota <= 0 {
		return nil
	}

	if !common.RedisEnabled {
		return fmt.Errorf("Redis is required for subscription quota")
	}

	// 获取用户激活的订阅
	userSub, subscription, err := model.GetActiveUserSubscriptionNoGroup(userId)
	if err != nil {
		// 没有激活的订阅，无法返还（理论上不应该发生）
		return err
	}

	// 使用 Redis 返还额度
	quotaRedis := NewSubscriptionQuotaRedis(userSub.Id, subscription)
	return quotaRedis.ReturnQuota(quota)
}
