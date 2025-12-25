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
	if sub.TotalQuotaLimit <= 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "总限额必须大于0",
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

	// 从 Redis 读取用量数据
	for _, sub := range subs {
		if sub.SubscriptionInfo != nil {
			quotaRedis := service.NewSubscriptionQuotaRedis(sub.Id, sub.SubscriptionInfo, sub.StartTime)
			dailyUsed, weeklyUsed, totalUsed, _ := quotaRedis.GetQuotaUsed()
			sub.DailyQuotaUsed = dailyUsed
			sub.WeeklyQuotaUsed = weeklyUsed
			sub.TotalQuotaUsed = totalUsed
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

	// 从 Redis 读取用量
	quotaRedis := service.NewSubscriptionQuotaRedis(us.Id, sub, us.StartTime)
	dailyUsed, weeklyUsed, totalUsed, _ := quotaRedis.GetQuotaUsed()

	data := map[string]interface{}{
		"daily": map[string]interface{}{
			"used":  dailyUsed,
			"limit": sub.DailyQuotaLimit,
		},
		"weekly": map[string]interface{}{
			"used":  weeklyUsed,
			"limit": sub.WeeklyQuotaLimit,
		},
		"total": map[string]interface{}{
			"used":  totalUsed,
			"limit": sub.TotalQuotaLimit,
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

// ========== 管理员管理用户订阅接口 ==========

// GetUserSubscriptions 管理员获取用户的所有订阅
func GetUserSubscriptions(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo := common.GetPageQuery(c)
	subs, total, err := model.GetUserSubscriptions(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 从 Redis 读取用量数据
	for _, sub := range subs {
		if sub.SubscriptionInfo != nil {
			quotaRedis := service.NewSubscriptionQuotaRedis(sub.Id, sub.SubscriptionInfo, sub.StartTime)
			dailyUsed, weeklyUsed, totalUsed, _ := quotaRedis.GetQuotaUsed()
			sub.DailyQuotaUsed = dailyUsed
			sub.WeeklyQuotaUsed = weeklyUsed
			sub.TotalQuotaUsed = totalUsed
		}
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(subs)
	common.ApiSuccess(c, pageInfo)
}

// AddUserSubscription 管理员为用户添加订阅
func AddUserSubscription(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req struct {
		SubscriptionId int `json:"subscription_id" binding:"required"`
		DurationDays   int `json:"duration_days"` // 可选，默认使用套餐的 duration_days
	}

	err = c.ShouldBindJSON(&req)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 调用 model 层添加订阅
	userSub, err := model.AdminAddUserSubscription(userId, req.SubscriptionId, req.DurationDays)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 从 Redis 读取用量数据
	if userSub.SubscriptionInfo != nil {
		quotaRedis := service.NewSubscriptionQuotaRedis(userSub.Id, userSub.SubscriptionInfo, userSub.StartTime)
		dailyUsed, weeklyUsed, totalUsed, _ := quotaRedis.GetQuotaUsed()
		userSub.DailyQuotaUsed = dailyUsed
		userSub.WeeklyQuotaUsed = weeklyUsed
		userSub.TotalQuotaUsed = totalUsed
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "订阅添加成功",
		"data":    userSub,
	})
}

// UpdateUserSubscription 管理员修改用户订阅
func UpdateUserSubscription(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	userSubId, err := strconv.Atoi(c.Param("subId"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req struct {
		SubscriptionId *int   `json:"subscription_id"` // 可选，更换套餐
		ExpireTime     *int64 `json:"expire_time"`      // 可选，修改过期时间
	}

	err = c.ShouldBindJSON(&req)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 至少要有一个字段
	if req.SubscriptionId == nil && req.ExpireTime == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "至少需要提供 subscription_id 或 expire_time 之一",
		})
		return
	}

	// 调用 model 层修改订阅
	userSub, err := model.AdminUpdateUserSubscription(userId, userSubId, req.SubscriptionId, req.ExpireTime)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 从 Redis 读取用量数据
	if userSub.SubscriptionInfo != nil {
		quotaRedis := service.NewSubscriptionQuotaRedis(userSub.Id, userSub.SubscriptionInfo, userSub.StartTime)
		dailyUsed, weeklyUsed, totalUsed, _ := quotaRedis.GetQuotaUsed()
		userSub.DailyQuotaUsed = dailyUsed
		userSub.WeeklyQuotaUsed = weeklyUsed
		userSub.TotalQuotaUsed = totalUsed
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "订阅修改成功",
		"data":    userSub,
	})
}

// CancelUserSubscription 管理员取消用户订阅
func CancelUserSubscription(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	userSubId, err := strconv.Atoi(c.Param("subId"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 调用 model 层取消订阅
	err = model.AdminCancelUserSubscription(userId, userSubId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "订阅已取消",
	})
}
