package service

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

func CacheGetRandomSatisfiedChannel(c *gin.Context, group string, modelName string, retry int) (*model.Channel, string, error) {
	var channel *model.Channel
	var err error
	selectGroup := group
	userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)

	// Get sticky session ID from context (set by distributor middleware)
	sessionId := common.GetContextKeyString(c, constant.ContextKeyStickySessionId)

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
		clientIP := c.ClientIP()
		if bindErr := model.BindStickySession(selectGroup, modelName, sessionId, channel, clientIP); bindErr != nil {
			logger.LogWarn(c, "Failed to bind sticky session: "+bindErr.Error())
		}
	}

	return channel, selectGroup, nil
}
