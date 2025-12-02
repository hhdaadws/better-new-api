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

	// Non-free group: return pre-consumed quota via PostConsumeQuota
	if relayInfo.FinalPreConsumedQuota != 0 {
		logger.LogInfo(c, fmt.Sprintf("用户 %d 请求失败, 返还预扣费额度 %s", relayInfo.UserId, logger.FormatQuota(relayInfo.FinalPreConsumedQuota)))
		gopool.Go(func() {
			relayInfoCopy := *relayInfo

			err := PostConsumeQuota(&relayInfoCopy, -relayInfoCopy.FinalPreConsumedQuota, 0, false)
			if err != nil {
				common.SysLog("error return pre-consumed quota: " + err.Error())
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

	// Non-free group: exclusively use user's paid quota
	if userQuota <= 0 {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("用户额度不足，剩余额度: %s", logger.FormatQuota(userQuota)),
			types.ErrorCodeInsufficientUserQuota, http.StatusForbidden,
			types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
	}

	if userQuota < preConsumedQuota {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("预扣费额度失败，用户剩余额度: %s，需要预扣费额度: %s", logger.FormatQuota(userQuota), logger.FormatQuota(preConsumedQuota)),
			types.ErrorCodeInsufficientUserQuota, http.StatusForbidden,
			types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
	}

	trustQuota := common.GetTrustQuota()

	relayInfo.UserQuota = userQuota
	if userQuota > trustQuota {
		// 用户额度充足，判断令牌额度是否充足
		if !relayInfo.TokenUnlimited {
			// 非无限令牌，判断令牌额度是否充足
			tokenQuota := c.GetInt("token_quota")
			if tokenQuota > trustQuota {
				// 令牌额度充足，信任令牌
				preConsumedQuota = 0
				logger.LogInfo(c, fmt.Sprintf("用户 %d 剩余额度 %s 且令牌 %d 额度 %d 充足, 信任且不需要预扣费", relayInfo.UserId, logger.FormatQuota(userQuota), relayInfo.TokenId, tokenQuota))
			}
		} else {
			// in this case, we do not pre-consume quota
			// because the user has enough quota
			preConsumedQuota = 0
			logger.LogInfo(c, fmt.Sprintf("用户 %d 额度充足且为无限额度令牌, 信任且不需要预扣费", relayInfo.UserId))
		}
	}

	if preConsumedQuota > 0 {
		err := PreConsumeTokenQuota(relayInfo, preConsumedQuota)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodePreConsumeTokenQuotaFailed, http.StatusForbidden, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}

		err = model.DecreaseUserQuota(relayInfo.UserId, preConsumedQuota)
		if err != nil {
			return types.NewError(err, types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
		}

		logger.LogInfo(c, fmt.Sprintf("用户 %d 预扣费 %s, 预扣费后剩余额度: %s",
			relayInfo.UserId, logger.FormatQuota(preConsumedQuota), logger.FormatQuota(userQuota-preConsumedQuota)))
	}

	relayInfo.FinalPreConsumedQuota = preConsumedQuota
	return nil
}
