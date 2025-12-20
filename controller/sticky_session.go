package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetChannelStickySessions returns all sticky sessions for a channel
func GetChannelStickySessions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的渠道 ID",
		})
		return
	}

	channel, err := model.GetChannelById(id, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "渠道不存在",
		})
		return
	}

	setting := channel.GetSetting()
	if !setting.StickySessionEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "该渠道未启用粘性会话",
			"data": model.StickySessionInfo{
				ChannelId:      id,
				ChannelName:    channel.Name,
				SessionCount:   0,
				MaxCount:       0,
				TTLMinutes:     0,
				DailyBindLimit: 0,
				DailyBindCount: 0,
				Sessions:       []model.StickySession{},
			},
		})
		return
	}

	sessions, err := common.GetChannelStickySessions(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取会话列表失败: " + err.Error(),
		})
		return
	}

	sessionList := make([]model.StickySession, 0)
	for _, s := range sessions {
		username, _ := s["username"].(string)
		tokenName, _ := s["token_name"].(string)
		sessionList = append(sessionList, model.StickySession{
			SessionHash: s["session_hash"].(string),
			ChannelId:   id,
			Group:       s["group"].(string),
			Model:       s["model"].(string),
			Username:    username,
			TokenName:   tokenName,
			CreatedAt:   s["created_at"].(int64),
			TTL:         s["ttl"].(int64),
		})
	}

	// Get daily bind count
	dailyBindCount, _ := common.GetChannelDailyBindCount(id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": model.StickySessionInfo{
			ChannelId:      id,
			ChannelName:    channel.Name,
			SessionCount:   len(sessionList),
			MaxCount:       setting.StickySessionMaxCount,
			TTLMinutes:     setting.StickySessionTTLMinutes,
			DailyBindLimit: setting.StickySessionDailyBindLimit,
			DailyBindCount: dailyBindCount,
			Sessions:       sessionList,
		},
	})
}

// ReleaseAllChannelStickySessions releases all sticky sessions for a channel
func ReleaseAllChannelStickySessions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的渠道 ID",
		})
		return
	}

	// Verify channel exists
	_, err = model.GetChannelById(id, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "渠道不存在",
		})
		return
	}

	err = common.ReleaseAllChannelStickySessions(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "释放会话失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "所有会话已释放",
	})
}

// ReleaseStickySession releases a specific sticky session
func ReleaseStickySession(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的渠道 ID",
		})
		return
	}

	sessionHash := c.Param("session_hash")
	if sessionHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "会话哈希不能为空",
		})
		return
	}

	// Verify channel exists
	_, err = model.GetChannelById(id, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "渠道不存在",
		})
		return
	}

	err = common.ReleaseStickySessionByChannelAndHash(id, sessionHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "释放会话失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "会话已释放",
	})
}

// GetAllStickySessionStats returns sticky session statistics for all channels
func GetAllStickySessionStats(c *gin.Context) {
	channels, err := model.GetAllChannels(0, 0, true, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取渠道列表失败",
		})
		return
	}

	stats := make([]model.StickySessionInfo, 0)
	for _, channel := range channels {
		setting := channel.GetSetting()
		if !setting.StickySessionEnabled {
			continue
		}

		count, _ := common.GetChannelStickySessionCount(channel.Id)
		dailyBindCount, _ := common.GetChannelDailyBindCount(channel.Id)
		stats = append(stats, model.StickySessionInfo{
			ChannelId:      channel.Id,
			ChannelName:    channel.Name,
			SessionCount:   count,
			MaxCount:       setting.StickySessionMaxCount,
			TTLMinutes:     setting.StickySessionTTLMinutes,
			DailyBindLimit: setting.StickySessionDailyBindLimit,
			DailyBindCount: dailyBindCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
