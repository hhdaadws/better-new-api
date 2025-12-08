package controller

import (
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// ========== 管理员接口 ==========

func GetAllSubscriptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	subs, total, err := model.GetAllSubscriptions(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(subs)
	common.ApiSuccess(c, pageInfo)
}

func GetSubscriptionById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	sub, err := model.GetSubscriptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    sub,
	})
}

func AddSubscription(c *gin.Context) {
	sub := &model.Subscription{}
	err := c.ShouldBindJSON(sub)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 验证
	if utf8.RuneCountInString(sub.Name) == 0 || utf8.RuneCountInString(sub.Name) > 64 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "套餐名称长度必须在1-64之间",
		})
		return
	}
	if sub.MonthlyQuotaLimit <= 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "月度限额必须大于0",
		})
		return
	}
	if sub.AllowedGroups == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "必须指定允许的分组",
		})
		return
	}

	err = sub.Insert()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    sub,
	})
}

func UpdateSubscription(c *gin.Context) {
	sub := &model.Subscription{}
	err := c.ShouldBindJSON(sub)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 验证
	if utf8.RuneCountInString(sub.Name) == 0 || utf8.RuneCountInString(sub.Name) > 64 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "套餐名称长度必须在1-64之间",
		})
		return
	}

	err = sub.Update()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    sub,
	})
}

func DeleteSubscription(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	sub, err := model.GetSubscriptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	err = sub.Delete()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// AddSubscriptionRedemption 创建订阅套餐兑换码
func AddSubscriptionRedemption(c *gin.Context) {
	var req struct {
		Name           string `json:"name"`
		SubscriptionId int    `json:"subscription_id"`
		Count          int    `json:"count"`
		ExpiredTime    int64  `json:"expired_time"`
	}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 验证套餐存在
	sub, err := model.GetSubscriptionById(req.SubscriptionId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "套餐不存在",
		})
		return
	}

	if sub.Status != model.SubscriptionStatusEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "套餐已禁用",
		})
		return
	}

	if req.Count <= 0 || req.Count > 100 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "兑换码数量必须在1-100之间",
		})
		return
	}

	if utf8.RuneCountInString(req.Name) == 0 || utf8.RuneCountInString(req.Name) > 20 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "兑换码名称长度必须在1-20之间",
		})
		return
	}

	if err := validateExpiredTime(req.ExpiredTime); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	var keys []string
	for i := 0; i < req.Count; i++ {
		key := common.GetUUID()
		redemption := &model.Redemption{
			UserId:         c.GetInt("id"),
			Name:           req.Name,
			Key:            key,
			Type:           2, // 订阅套餐码
			SubscriptionId: &req.SubscriptionId,
			CreatedTime:    common.GetTimestamp(),
			ExpiredTime:    req.ExpiredTime,
		}

		err = redemption.Insert()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    keys,
			})
			return
		}
		keys = append(keys, key)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    keys,
	})
}

// ========== 用户接口 ==========

func GetMySubscriptions(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	subs, total, err := model.GetUserSubscriptions(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 如果 Redis 启用，用 Redis 数据覆盖用量字段
	if common.RedisEnabled {
		for _, sub := range subs {
			if sub.SubscriptionInfo != nil {
				quotaRedis := service.NewSubscriptionQuotaRedis(sub.Id, sub.SubscriptionInfo)
				dailyUsed, weeklyUsed, monthlyUsed, _ := quotaRedis.GetQuotaUsed()
				sub.DailyQuotaUsed = dailyUsed
				sub.WeeklyQuotaUsed = weeklyUsed
				sub.MonthlyQuotaUsed = monthlyUsed
			}
		}
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(subs)
	common.ApiSuccess(c, pageInfo)
}

func GetMySubscriptionQuota(c *gin.Context) {
	userId := c.GetInt("id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var us model.UserSubscription
	err = model.DB.Where("id = ? AND user_id = ?", id, userId).First(&us).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 获取套餐信息
	sub, _ := model.GetSubscriptionById(us.SubscriptionId)

	var dailyUsed, weeklyUsed, monthlyUsed int

	// 优先从 Redis 读取用量
	if common.RedisEnabled {
		quotaRedis := service.NewSubscriptionQuotaRedis(us.Id, sub)
		dailyUsed, weeklyUsed, monthlyUsed, _ = quotaRedis.GetQuotaUsed()
	} else {
		// Redis 未启用，从数据库读取
		if us.CheckAndResetQuota() {
			us.Update()
		}
		dailyUsed = us.DailyQuotaUsed
		weeklyUsed = us.WeeklyQuotaUsed
		monthlyUsed = us.MonthlyQuotaUsed
	}

	data := map[string]interface{}{
		"daily": map[string]interface{}{
			"used":  dailyUsed,
			"limit": sub.DailyQuotaLimit,
		},
		"weekly": map[string]interface{}{
			"used":  weeklyUsed,
			"limit": sub.WeeklyQuotaLimit,
		},
		"monthly": map[string]interface{}{
			"used":  monthlyUsed,
			"limit": sub.MonthlyQuotaLimit,
		},
		"expire_time": us.ExpireTime,
		"status":      us.Status,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
}

func GetMySubscriptionLogs(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)

	logs, total, err := model.GetSubscriptionLogs(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
}
