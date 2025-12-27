package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// ========== 管理员接口：用户专属渠道管理 ==========

// GetUserExclusiveChannels 获取用户的专属渠道列表
func GetUserExclusiveChannels(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}

	channels, err := model.GetUserSubscriptionChannels(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channels,
	})
}

// AddUserExclusiveChannel 为用户添加专属渠道
func AddUserExclusiveChannel(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}

	var req struct {
		ChannelId int `json:"channel_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 验证用户是否启用了专属分组
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	if !userCache.EnableExclusiveGroup {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该用户未启用专属分组功能",
		})
		return
	}

	// 获取用户订阅（用于关联记录，如果有的话）
	userSub, _, _ := model.GetActiveUserSubscriptionNoGroup(userId)
	userSubId := 0
	if userSub != nil {
		userSubId = userSub.Id
	}

	// 验证渠道是否存在
	channel, err := model.GetChannelById(req.ChannelId, false)
	if err != nil || channel == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "渠道不存在",
		})
		return
	}

	// 检查是否已绑定
	existingChannels, _ := model.GetUserSubscriptionChannels(userId)
	for _, ec := range existingChannels {
		if ec.ChannelId == req.ChannelId {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "该渠道已绑定",
			})
			return
		}
	}

	// 创建绑定
	usc := &model.UserSubscriptionChannel{
		UserSubscriptionId: userSubId,
		UserId:             userId,
		ChannelId:          req.ChannelId,
	}

	if err := usc.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "绑定失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "渠道绑定成功",
	})
}

// RemoveUserExclusiveChannel 移除用户的专属渠道
func RemoveUserExclusiveChannel(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}

	channelId, err := strconv.Atoi(c.Param("channelId"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的渠道ID",
		})
		return
	}

	if err := model.DeleteUserSubscriptionChannel(userId, channelId); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "移除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "渠道绑定已移除",
	})
}

// GetAvailableChannelsForExclusive 获取可供绑定的渠道列表
func GetAvailableChannelsForExclusive(c *gin.Context) {
	// 获取所有启用的渠道
	channels, err := model.GetAllChannelsForExclusive()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channels,
	})
}

// GetUsersWithExclusiveGroup 获取有专属分组权限的用户列表
func GetUsersWithExclusiveGroup(c *gin.Context) {
	users, err := model.GetUsersWithExclusiveGroup()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    users,
	})
}

// ========== 用户接口 ==========

// GetUserExclusiveGroupStatus 获取当前用户的专属分组状态
func GetUserExclusiveGroupStatus(c *gin.Context) {
	userId := c.GetInt("id")

	hasPermission, hasChannels, groupName, groupRatio := service.GetUserExclusiveGroupStatus(userId)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"has_permission": hasPermission,
			"has_channels":   hasChannels,
			"group_name":     groupName,
			"group_ratio":    groupRatio,
		},
	})
}
