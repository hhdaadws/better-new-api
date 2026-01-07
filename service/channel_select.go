package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

func CacheGetRandomSatisfiedChannel(c *gin.Context, group string, modelName string, retry int) (*model.Channel, string, error) {
	var channel *model.Channel
	var err error
	selectGroup := group
	userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)

	// Get sticky session ID from context (set by distributor middleware)
	sessionId := common.GetContextKeyString(c, constant.ContextKeyStickySessionId)
	if common.DebugEnabled {
		common.SysLog("Sticky session ID from context: " + sessionId)
	}

	if group == "auto" {
		if len(setting.GetAutoGroups()) == 0 {
			return nil, selectGroup, errors.New("auto groups is not enabled")
		}
		for _, autoGroup := range GetUserAutoGroup(userGroup) {
			logger.LogDebug(c, "Auto selecting group:", autoGroup)
			channel, _ = model.GetRandomSatisfiedChannelWithStickySession(autoGroup, modelName, retry, sessionId)
			if channel == nil {
				continue
			} else {
				c.Set("auto_group", autoGroup)
				selectGroup = autoGroup
				logger.LogDebug(c, "Auto selected group:", autoGroup)
				break
			}
		}
	} else {
		channel, err = model.GetRandomSatisfiedChannelWithStickySession(group, modelName, retry, sessionId)
		if err != nil {
			return nil, group, err
		}
	}

	// Bind sticky session if channel was selected and sessionId is provided
	if channel != nil && sessionId != "" {
		// Check if this channel switch qualifies for free cache creation
		// (switching from lower priority to higher priority channel, and previous usage within 5 minutes)
		// Only opus and sonnet models are eligible for free cache creation, haiku is excluded
		modelLower := strings.ToLower(modelName)
		isEligibleForFreeCache := strings.Contains(modelLower, "opus") ||
			strings.Contains(modelLower, "sonnet")

		if isEligibleForFreeCache && common.RedisEnabled && operation_setting.IsFreeCacheCreationEnabled() {
			eligible, prevChannelId := common.CheckChannelSwitchForFreeCache(
				selectGroup, modelName, sessionId,
				channel.Id, channel.GetPriority())

			if eligible {
				common.SetContextKey(c, constant.ContextKeyFreeCacheCreation, true)
				common.SetContextKey(c, constant.ContextKeyFreeCachePrevChannel, prevChannelId)
				if common.DebugEnabled {
					common.SysLog(fmt.Sprintf("Free cache creation eligible: sessionId=%s, prevChannel=%d, newChannel=%d",
						sessionId, prevChannelId, channel.Id))
				}
			}

			// Update channel usage record (only for opus/sonnet models)
			_ = common.SetSessionChannelUsage(selectGroup, modelName, sessionId,
				channel.Id, channel.GetPriority())
		}

		username := c.GetString("username")
		tokenName := c.GetString("token_name")
		if common.DebugEnabled {
			common.SysLog("Binding sticky session: sessionId=" + sessionId + ", username=" + username + ", tokenName=" + tokenName)
		}
		if bindErr := model.BindStickySession(selectGroup, modelName, sessionId, channel, username, tokenName); bindErr != nil {
			logger.LogWarn(c, "Failed to bind sticky session: "+bindErr.Error())
		} else {
			// Mark this request as using sticky session for quota tracking
			common.SetContextKey(c, constant.ContextKeyStickySessionChannelId, channel.Id)
		}
	} else if common.DebugEnabled && channel != nil && sessionId == "" {
		common.SysLog("Sticky session not bound: sessionId is empty")
	}

	return channel, selectGroup, nil
}
