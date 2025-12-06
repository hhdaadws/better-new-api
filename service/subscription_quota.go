package service

import (
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// TryConsumeSubscriptionQuota 尝试使用订阅额度
// 返回值：(是否使用了订阅额度, 错误信息)
func TryConsumeSubscriptionQuota(relayInfo *relaycommon.RelayInfo, quota int) (bool, error) {
	// 1. 获取用户激活的订阅
	userSub, subscription, err := model.GetActiveUserSubscription(
		relayInfo.UserId,
		relayInfo.UsingGroup,
	)
	if err != nil {
		// 没有订阅或订阅不支持该分组，返回 false
		return false, nil
	}

	// 2. 尝试消费额度
	err = userSub.ConsumeQuota(quota, subscription)
	if err != nil {
		// 订阅额度不足，返回 false，让系统降级到普通额度
		return false, nil
	}

	// 3. 记录日志（异步）
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
