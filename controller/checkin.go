package controller

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// CheckinConfig holds the configuration for daily check-in
type CheckinConfig struct {
	Enabled     bool   `json:"enabled"`      // 是否启用签到功能
	QuotaAmount int    `json:"quota_amount"` // 签到获得的额度数量
	Group       string `json:"group"`        // 签到额度只能用于的渠道分组 (固定为 "free")
}

// CheckinQuotaGroup is the fixed group name for check-in quota
// This must match the constant in service/pre_consume_quota.go
const CheckinQuotaGroup = "free"

// Default check-in configuration
var checkinConfig = CheckinConfig{
	Enabled:     true,
	QuotaAmount: 500000,          // 默认 $1 额度
	Group:       CheckinQuotaGroup, // 签到额度只能用于 "free" 分组
}

// GetCheckinConfig returns the current check-in configuration
func GetCheckinConfig() CheckinConfig {
	return checkinConfig
}

// SetCheckinConfig updates the check-in configuration
func SetCheckinConfig(config CheckinConfig) {
	checkinConfig = config
}

// GetCheckinInfo returns check-in configuration and user's check-in status
func GetCheckinInfo(c *gin.Context) {
	if !common.RedisEnabled {
		common.ApiErrorMsg(c, "签到功能需要启用 Redis")
		return
	}

	userId := c.GetInt("id")
	status, err := service.GetCheckinStatus(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 签到强制使用 Turnstile（只要配置了 site key）
	turnstileRequired := common.TurnstileSiteKey != "" && common.TurnstileSecretKey != ""

	common.ApiSuccess(c, gin.H{
		"config": gin.H{
			"enabled":            checkinConfig.Enabled,
			"quota_amount":       checkinConfig.QuotaAmount,
			"quota_amount_label": logger.LogQuota(checkinConfig.QuotaAmount),
			"group":              checkinConfig.Group,
			"turnstile_required": turnstileRequired,
			"turnstile_site_key": common.TurnstileSiteKey,
		},
		"status": status,
	})
}

// PerformCheckin handles the daily check-in request
func PerformCheckin(c *gin.Context) {
	if !common.RedisEnabled {
		common.ApiErrorMsg(c, "签到功能需要启用 Redis")
		return
	}

	if !checkinConfig.Enabled {
		common.ApiErrorMsg(c, "签到功能未启用")
		return
	}

	userId := c.GetInt("id")

	// Perform check-in
	status, err := service.PerformCheckin(userId, checkinConfig.QuotaAmount)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	// Record log
	model.RecordLog(userId, model.LogTypeSystem, fmt.Sprintf("每日签到成功，获得临时额度 %s（当日有效）", logger.LogQuota(checkinConfig.QuotaAmount)))

	common.ApiSuccess(c, gin.H{
		"message": fmt.Sprintf("签到成功！获得 %s 临时额度（当日有效）", logger.LogQuota(checkinConfig.QuotaAmount)),
		"status":  status,
	})
}

// AdminUpdateCheckinConfig allows admin to update check-in configuration
func AdminUpdateCheckinConfig(c *gin.Context) {
	var config CheckinConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	if config.QuotaAmount < 0 {
		common.ApiErrorMsg(c, "额度数量不能为负数")
		return
	}

	SetCheckinConfig(config)
	common.ApiSuccess(c, gin.H{
		"message": "签到配置更新成功",
		"config":  config,
	})
}

// AdminGetCheckinConfig returns current check-in configuration for admin
func AdminGetCheckinConfig(c *gin.Context) {
	common.ApiSuccess(c, checkinConfig)
}
