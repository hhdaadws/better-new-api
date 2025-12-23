package service

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

func ReturnPreConsumedQuota(c *gin.Context, relayInfo *relaycommon.RelayInfo) {
	// Check if this is a free group request
	isFreeGroup := relayInfo.UsingGroup == CheckinQuotaGroup

	if isFreeGroup {
		// Free group: only return check-in quota, do NOT call PostConsumeQuota
		// because free group never touched user's paid quota
		if relayInfo.CheckinQuotaConsumed > 0 {
			logger.LogInfo(c, fmt.Sprintf("用户 %d 请求失败 (free 分组), 返还签到额度 %s", relayInfo.UserId, logger.FormatQuota(relayInfo.CheckinQuotaConsumed)))
			gopool.Go(func() {
				err := ReturnCheckinQuota(relayInfo.UserId, relayInfo.CheckinQuotaConsumed)
				if err != nil {
					common.SysLog("error return check-in quota: " + err.Error())
				}
			})
		}
		// Also return token quota for free group
		if relayInfo.FinalPreConsumedQuota != 0 && !relayInfo.IsPlayground {
			gopool.Go(func() {
				err := model.IncreaseTokenQuota(relayInfo.TokenId, relayInfo.TokenKey, relayInfo.FinalPreConsumedQuota)
				if err != nil {
					common.SysLog("error return token quota for free group: " + err.Error())
				}
			})
		}
		return
	}

	// Non-free group: return pre-consumed quota
	// 必须返还到正确的来源（订阅额度或用户余额）
	if relayInfo.FinalPreConsumedQuota != 0 {
		logger.LogInfo(c, fmt.Sprintf("用户 %d 请求失败, 返还预扣费额度 %s", relayInfo.UserId, logger.FormatQuota(relayInfo.FinalPreConsumedQuota)))
		gopool.Go(func() {
			// 返还 Token 额度
			if !relayInfo.IsPlayground {
				err := model.IncreaseTokenQuota(relayInfo.TokenId, relayInfo.TokenKey, relayInfo.FinalPreConsumedQuota)
				if err != nil {
					common.SysLog("error return token quota: " + err.Error())
				}
			}

			// 返还到预扣的来源
			if relayInfo.SubscriptionPreConsumed {
				// 预扣的是订阅额度，返还到订阅额度
				err := ReturnSubscriptionQuota(relayInfo.UserId, relayInfo.FinalPreConsumedQuota)
				if err != nil {
					common.SysLog("error return subscription quota: " + err.Error())
				}
			} else {
				// 预扣的是用户余额，返还到用户余额
				err := model.IncreaseUserQuota(relayInfo.UserId, relayInfo.FinalPreConsumedQuota, false)
				if err != nil {
					common.SysLog("error return user quota: " + err.Error())
				}
			}
		})
	}
}

// CheckinQuotaGroup is the group name that check-in quota can be used for
// Admin must create a channel group with exactly this name
const CheckinQuotaGroup = "free"

// PreConsumeQuota checks if the user has enough quota to pre-consume.
// It returns the pre-consumed quota if successful, or an error if not.
// IMPORTANT: "free" group can ONLY use check-in quota, NOT user's paid quota.
// Other groups can ONLY use user's paid quota, NOT check-in quota.
func PreConsumeQuota(c *gin.Context, preConsumedQuota int, relayInfo *relaycommon.RelayInfo) *types.NewAPIError {
	userQuota, err := model.GetUserQuota(relayInfo.UserId, false)
	if err != nil {
		return types.NewError(err, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
	}

	// Check if current request is using the "free" group
	isFreeGroup := relayInfo.UsingGroup == CheckinQuotaGroup

	// For "free" group: ONLY use check-in quota
	// For other groups: ONLY use user's paid quota
	if isFreeGroup {
		// Free group: exclusively use check-in quota
		checkinQuota, _ := GetCheckinQuota(relayInfo.UserId)

		if checkinQuota <= 0 {
			return types.NewErrorWithStatusCode(
				fmt.Errorf("您在 free 分组的免费额度已用完（剩余: %s）。请签到获取免费额度，或使用付费分组的 API Key", logger.FormatQuota(checkinQuota)),
				types.ErrorCodeInsufficientUserQuota, http.StatusForbidden,
				types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}

		if checkinQuota < preConsumedQuota {
			return types.NewErrorWithStatusCode(
				fmt.Errorf("您在 free 分组的免费额度不足（剩余: %s，需要: %s）。请签到获取更多免费额度，或使用付费分组的 API Key",
					logger.FormatQuota(checkinQuota), logger.FormatQuota(preConsumedQuota)),
				types.ErrorCodeInsufficientUserQuota, http.StatusForbidden,
				types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}

		// Pre-consume from check-in quota only
		if preConsumedQuota > 0 {
			err := PreConsumeTokenQuota(relayInfo, preConsumedQuota)
			if err != nil {
				return types.NewErrorWithStatusCode(err, types.ErrorCodePreConsumeTokenQuotaFailed, http.StatusForbidden, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
			}

			checkinConsumed, _ := ConsumeCheckinQuota(relayInfo.UserId, preConsumedQuota)
			relayInfo.CheckinQuotaConsumed = checkinConsumed

			logger.LogInfo(c, fmt.Sprintf("用户 %d 使用签到额度 %s (free 分组), 预扣费后剩余签到额度: %s",
				relayInfo.UserId, logger.FormatQuota(checkinConsumed), logger.FormatQuota(checkinQuota-checkinConsumed)))
		}

		relayInfo.UserQuota = userQuota
		relayInfo.FinalPreConsumedQuota = preConsumedQuota
		return nil
	}

	// Non-free group: 优先使用订阅额度，不足时降级到用户余额
	// 订阅额度是用户余额的替代品，不区分分组

	// ========== 专属分组：仅使用订阅额度，不降级到用户余额 ==========
	isExclusiveGroup := model.IsExclusiveGroup(relayInfo.UsingGroup)
	if isExclusiveGroup {
		// 验证专属分组是否属于当前用户
		exclusiveUserId, ok := model.GetExclusiveGroupUserId(relayInfo.UsingGroup)
		if !ok || exclusiveUserId != relayInfo.UserId {
			return types.NewErrorWithStatusCode(
				fmt.Errorf("无权使用他人的专属分组"),
				types.ErrorCodeForbidden, http.StatusForbidden,
				types.ErrOptionWithSkipRetry())
		}

		// 获取用户的有效订阅
		userSub, sub, err := model.GetActiveUserSubscriptionNoGroup(relayInfo.UserId)
		if err != nil || sub == nil || userSub == nil {
			return types.NewErrorWithStatusCode(
				fmt.Errorf("专属分组需要有效订阅"),
				types.ErrorCodeInsufficientUserQuota, http.StatusForbidden,
				types.ErrOptionWithSkipRetry())
		}

		// 验证订阅是否启用专属分组
		if !sub.EnableExclusiveGroup {
			return types.NewErrorWithStatusCode(
				fmt.Errorf("当前订阅套餐未启用专属分组功能"),
				types.ErrorCodeForbidden, http.StatusForbidden,
				types.ErrOptionWithSkipRetry())
		}

		// 获取订阅剩余额度
		quotaRedis := NewSubscriptionQuotaRedis(userSub.Id, sub)
		dailyUsed, weeklyUsed, totalUsed, _ := quotaRedis.GetQuotaUsed()

		// 计算各维度剩余额度
		dailyRemaining := sub.DailyQuotaLimit - dailyUsed
		if sub.DailyQuotaLimit == 0 {
			dailyRemaining = int(^uint(0) >> 1) // 无限制
		}
		weeklyRemaining := sub.WeeklyQuotaLimit - weeklyUsed
		if sub.WeeklyQuotaLimit == 0 {
			weeklyRemaining = int(^uint(0) >> 1) // 无限制
		}
		totalRemaining := sub.TotalQuotaLimit - totalUsed
		if sub.TotalQuotaLimit == 0 {
			totalRemaining = int(^uint(0) >> 1) // 无限制
		}

		// 取最小值作为可用额度
		subscriptionQuotaAvailable := dailyRemaining
		if weeklyRemaining < subscriptionQuotaAvailable {
			subscriptionQuotaAvailable = weeklyRemaining
		}
		if totalRemaining < subscriptionQuotaAvailable {
			subscriptionQuotaAvailable = totalRemaining
		}
		if subscriptionQuotaAvailable < 0 {
			subscriptionQuotaAvailable = 0
		}

		// 专属分组仅使用订阅额度，不降级到用户余额
		if subscriptionQuotaAvailable <= 0 {
			return types.NewErrorWithStatusCode(
				fmt.Errorf("专属分组订阅额度已用完（日剩余: %s，周剩余: %s，总剩余: %s）",
					logger.FormatQuota(sub.DailyQuotaLimit-dailyUsed),
					logger.FormatQuota(sub.WeeklyQuotaLimit-weeklyUsed),
					logger.FormatQuota(sub.TotalQuotaLimit-totalUsed)),
				types.ErrorCodeInsufficientUserQuota, http.StatusForbidden,
				types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}

		relayInfo.UserQuota = userQuota

		// 信任机制判断
		trustQuota := common.GetTrustQuota()
		if subscriptionQuotaAvailable > trustQuota {
			if !relayInfo.TokenUnlimited {
				tokenQuota := c.GetInt("token_quota")
				if tokenQuota > trustQuota {
					preConsumedQuota = 0
					logger.LogInfo(c, fmt.Sprintf("用户 %d 专属分组订阅额度 %s 且令牌 %d 额度 %d 充足, 信任且不需要预扣费",
						relayInfo.UserId, logger.FormatQuota(subscriptionQuotaAvailable),
						relayInfo.TokenId, tokenQuota))
				}
			} else {
				preConsumedQuota = 0
				logger.LogInfo(c, fmt.Sprintf("用户 %d 专属分组订阅额度 %s 且为无限额度令牌, 信任且不需要预扣费",
					relayInfo.UserId, logger.FormatQuota(subscriptionQuotaAvailable)))
			}
		}

		if preConsumedQuota > 0 {
			// 预扣 Token 额度
			err := PreConsumeTokenQuota(relayInfo, preConsumedQuota)
			if err != nil {
				return types.NewErrorWithStatusCode(err, types.ErrorCodePreConsumeTokenQuotaFailed, http.StatusForbidden, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
			}

			// 仅从订阅额度预扣，不降级
			usedSubscription, _ := TryPreConsumeSubscriptionQuota(relayInfo.UserId, preConsumedQuota)
			if !usedSubscription {
				// 返还 Token 额度
				model.IncreaseTokenQuota(relayInfo.TokenId, relayInfo.TokenKey, preConsumedQuota)
				return types.NewErrorWithStatusCode(
					fmt.Errorf("专属分组订阅额度预扣失败，请检查额度是否充足"),
					types.ErrorCodeInsufficientUserQuota, http.StatusForbidden,
					types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
			}

			relayInfo.SubscriptionPreConsumed = true
			relayInfo.ExclusiveGroupUsed = true
			logger.LogInfo(c, fmt.Sprintf("用户 %d 从专属分组订阅额度预扣费 %s",
				relayInfo.UserId, logger.FormatQuota(preConsumedQuota)))
		}

		relayInfo.ExclusiveGroupUsed = true
		relayInfo.FinalPreConsumedQuota = preConsumedQuota
		return nil
	}

	// ========== 非专属分组：优先使用订阅额度，不足时降级到用户余额 ==========

	// 检查是否有可用的订阅额度（用于判断总额度是否足够）
	subscriptionQuotaAvailable := 0 // 订阅可用额度
	if userSub, sub, err := model.GetActiveUserSubscriptionNoGroup(relayInfo.UserId); err == nil && sub != nil {
		// 获取订阅剩余额度（取日/周/总限额中最小的可用额度）
		quotaRedis := NewSubscriptionQuotaRedis(userSub.Id, sub)
		dailyUsed, weeklyUsed, totalUsed, _ := quotaRedis.GetQuotaUsed()

		// 计算各维度剩余额度
		dailyRemaining := sub.DailyQuotaLimit - dailyUsed
		if sub.DailyQuotaLimit == 0 {
			dailyRemaining = int(^uint(0) >> 1) // 无限制
		}
		weeklyRemaining := sub.WeeklyQuotaLimit - weeklyUsed
		if sub.WeeklyQuotaLimit == 0 {
			weeklyRemaining = int(^uint(0) >> 1) // 无限制
		}
		totalRemaining := sub.TotalQuotaLimit - totalUsed
		if sub.TotalQuotaLimit == 0 {
			totalRemaining = int(^uint(0) >> 1) // 无限制
		}

		// 取最小值作为可用额度
		subscriptionQuotaAvailable = dailyRemaining
		if weeklyRemaining < subscriptionQuotaAvailable {
			subscriptionQuotaAvailable = weeklyRemaining
		}
		if totalRemaining < subscriptionQuotaAvailable {
			subscriptionQuotaAvailable = totalRemaining
		}
		if subscriptionQuotaAvailable < 0 {
			subscriptionQuotaAvailable = 0
		}
	}

	// 判断总额度是否足够（订阅可用额度 + 用户余额）
	totalAvailableQuota := userQuota + subscriptionQuotaAvailable
	if totalAvailableQuota <= 0 {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("用户额度不足，剩余额度: %s（订阅剩余: %s，余额: %s）",
				logger.FormatQuota(totalAvailableQuota),
				logger.FormatQuota(subscriptionQuotaAvailable),
				logger.FormatQuota(userQuota)),
			types.ErrorCodeInsufficientUserQuota, http.StatusForbidden,
			types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
	}

	trustQuota := common.GetTrustQuota()

	relayInfo.UserQuota = userQuota
	// 如果总可用额度充足（订阅剩余+用户余额），考虑信任机制
	if totalAvailableQuota > trustQuota {
		// 总额度充足，判断令牌额度是否充足
		if !relayInfo.TokenUnlimited {
			// 非无限令牌，判断令牌额度是否充足
			tokenQuota := c.GetInt("token_quota")
			if tokenQuota > trustQuota {
				// 令牌额度充足，信任令牌
				preConsumedQuota = 0
				logger.LogInfo(c, fmt.Sprintf("用户 %d 总可用额度 %s (订阅剩余: %s, 余额: %s) 且令牌 %d 额度 %d 充足, 信任且不需要预扣费",
					relayInfo.UserId, logger.FormatQuota(totalAvailableQuota),
					logger.FormatQuota(subscriptionQuotaAvailable), logger.FormatQuota(userQuota),
					relayInfo.TokenId, tokenQuota))
			}
		} else {
			// in this case, we do not pre-consume quota
			// because the user has enough quota
			preConsumedQuota = 0
			logger.LogInfo(c, fmt.Sprintf("用户 %d 总可用额度 %s (订阅剩余: %s, 余额: %s) 且为无限额度令牌, 信任且不需要预扣费",
				relayInfo.UserId, logger.FormatQuota(totalAvailableQuota),
				logger.FormatQuota(subscriptionQuotaAvailable), logger.FormatQuota(userQuota)))
		}
	}

	if preConsumedQuota > 0 {
		// 先预扣 Token 额度
		err := PreConsumeTokenQuota(relayInfo, preConsumedQuota)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodePreConsumeTokenQuotaFailed, http.StatusForbidden, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}

		// ========== 预扣费优先级：订阅额度 > 用户余额 ==========
		usedSubscription, _ := TryPreConsumeSubscriptionQuota(relayInfo.UserId, preConsumedQuota)

		if usedSubscription {
			// 成功从订阅额度预扣
			relayInfo.SubscriptionPreConsumed = true
			logger.LogInfo(c, fmt.Sprintf("用户 %d 从订阅额度预扣费 %s",
				relayInfo.UserId, logger.FormatQuota(preConsumedQuota)))
		} else {
			// 订阅额度不足或不可用，从用户余额预扣
			if userQuota < preConsumedQuota {
				// Token 额度已扣，需要返还
				model.IncreaseTokenQuota(relayInfo.TokenId, relayInfo.TokenKey, preConsumedQuota)
				return types.NewErrorWithStatusCode(
					fmt.Errorf("预扣费额度失败，用户剩余额度: %s，需要预扣费额度: %s", logger.FormatQuota(userQuota), logger.FormatQuota(preConsumedQuota)),
					types.ErrorCodeInsufficientUserQuota, http.StatusForbidden,
					types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
			}

			err = model.DecreaseUserQuota(relayInfo.UserId, preConsumedQuota)
			if err != nil {
				// Token 额度已扣，需要返还
				model.IncreaseTokenQuota(relayInfo.TokenId, relayInfo.TokenKey, preConsumedQuota)
				return types.NewError(err, types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
			}
			relayInfo.SubscriptionPreConsumed = false
			logger.LogInfo(c, fmt.Sprintf("用户 %d 从用户余额预扣费 %s, 预扣费后剩余额度: %s",
				relayInfo.UserId, logger.FormatQuota(preConsumedQuota), logger.FormatQuota(userQuota-preConsumedQuota)))
		}
	}

	relayInfo.FinalPreConsumedQuota = preConsumedQuota
	return nil
}
