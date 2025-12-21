package model

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// RiskControlBanInfo 风控封禁详情
type RiskControlBanInfo struct {
	BannedAt      int64    `json:"banned_at"`       // 封禁时间戳
	IPCount       int      `json:"ip_count"`        // 触发时的IP数量
	TimeWindowMin int      `json:"time_window_min"` // 时间窗口（分钟）
	IPList        []string `json:"ip_list"`         // IP列表
	Reason        string   `json:"reason"`          // 封禁原因
}

// CountDistinctIPsInTimeWindow 统计指定用户在时间窗口内的不同IP数量
func CountDistinctIPsInTimeWindow(userId int, timeWindowMin int) (int, []string, error) {
	startTime := time.Now().Add(-time.Duration(timeWindowMin) * time.Minute).Unix()

	var ips []string
	err := LOG_DB.Model(&Log{}).
		Where("user_id = ? AND type = ? AND created_at >= ? AND ip != ''", userId, LogTypeConsume, startTime).
		Distinct("ip").
		Pluck("ip", &ips).Error

	if err != nil {
		return 0, nil, err
	}

	return len(ips), ips, nil
}

// CheckAndBanUser 检查用户是否触发风控并执行封禁
// 返回：是否被封禁，错误
func CheckAndBanUser(userId int, currentIP string) (banned bool, err error) {
	// 检查风控是否启用
	if !common.RiskControlEnabled {
		return false, nil
	}

	// 获取用户信息
	user, err := GetUserById(userId, false)
	if err != nil {
		return false, err
	}

	// 检查是否已被封禁
	if user.Status != common.UserStatusEnabled {
		return false, nil
	}

	// 检查是否豁免
	if user.RiskControlExempt {
		return false, nil
	}

	// 管理员默认豁免
	if user.Role >= common.RoleAdminUser {
		return false, nil
	}

	// 先尝试使用 Redis 快速判断
	if common.RedisEnabled {
		// 添加当前IP到Redis集合
		err = common.RiskControlAddIP(userId, currentIP, common.RiskControlTimeWindowMin)
		if err != nil {
			common.SysLog(fmt.Sprintf("风控: 添加IP到Redis失败，用户ID: %d, 错误: %s", userId, err.Error()))
		}

		// 快速获取IP数量
		count, err := common.RiskControlGetIPCount(userId)
		if err == nil && count < int64(common.RiskControlIPThreshold) {
			// 未超过阈值，无需进一步检查
			return false, nil
		}
	}

	// 从数据库统计不同IP数量
	ipCount, ipList, err := CountDistinctIPsInTimeWindow(userId, common.RiskControlTimeWindowMin)
	if err != nil {
		common.SysLog(fmt.Sprintf("风控: 检测失败，用户ID: %d, 错误: %s", userId, err.Error()))
		return false, err
	}

	// 判断是否超过阈值
	if ipCount >= common.RiskControlIPThreshold {
		// 构建封禁信息
		banInfo := RiskControlBanInfo{
			BannedAt:      time.Now().Unix(),
			IPCount:       ipCount,
			TimeWindowMin: common.RiskControlTimeWindowMin,
			IPList:        ipList,
			Reason: fmt.Sprintf("在 %d 分钟内使用了 %d 个不同IP地址，超过阈值 %d",
				common.RiskControlTimeWindowMin, ipCount, common.RiskControlIPThreshold),
		}

		banInfoJSON, _ := common.Marshal(banInfo)

		// 更新用户状态
		err = DB.Model(&User{}).Where("id = ?", userId).Updates(map[string]interface{}{
			"status":                   common.UserStatusRiskBanned,
			"risk_control_banned_at":   banInfo.BannedAt,
			"risk_control_banned_info": string(banInfoJSON),
		}).Error

		if err != nil {
			return false, err
		}

		// 清除用户缓存
		_ = invalidateUserCache(userId)

		// 清除Redis中的IP集合
		if common.RedisEnabled {
			_ = common.RiskControlClearIPs(userId)
		}

		// 记录日志
		RecordLog(userId, LogTypeSystem, fmt.Sprintf("用户触发风控封禁：%s", banInfo.Reason))

		common.SysLog(fmt.Sprintf("风控: 封禁用户 ID: %d, 用户名: %s, IP数量: %d", userId, user.Username, ipCount))

		return true, nil
	}

	return false, nil
}

// UnbanRiskControlUser 解除风控封禁
func UnbanRiskControlUser(userId int, adminId int) error {
	// 检查用户是否处于风控封禁状态
	user, err := GetUserById(userId, false)
	if err != nil {
		return err
	}

	if user.Status != common.UserStatusRiskBanned {
		return fmt.Errorf("该用户未被风控封禁")
	}

	// 更新用户状态
	err = DB.Model(&User{}).Where("id = ?", userId).Updates(map[string]interface{}{
		"status":                   common.UserStatusEnabled,
		"risk_control_banned_at":   0,
		"risk_control_banned_info": "",
	}).Error

	if err != nil {
		return err
	}

	// 清除用户缓存
	_ = invalidateUserCache(userId)

	// 清除Redis中的IP集合
	if common.RedisEnabled {
		_ = common.RiskControlClearIPs(userId)
	}

	// 记录日志
	adminUsername, _ := GetUsernameById(adminId, false)
	RecordLog(userId, LogTypeManage, fmt.Sprintf("管理员 %s (ID: %d) 解除了风控封禁", adminUsername, adminId))

	common.SysLog(fmt.Sprintf("风控: 管理员 %s 解除了用户 ID: %d 的风控封禁", adminUsername, userId))

	return nil
}

// SetRiskControlExempt 设置用户风控豁免状态
func SetRiskControlExempt(userId int, exempt bool, adminId int) error {
	err := DB.Model(&User{}).Where("id = ?", userId).Update("risk_control_exempt", exempt).Error

	if err != nil {
		return err
	}

	// 清除用户缓存
	_ = invalidateUserCache(userId)

	// 如果开启豁免，清除Redis中的IP集合
	if exempt && common.RedisEnabled {
		_ = common.RiskControlClearIPs(userId)
	}

	// 记录日志
	adminUsername, _ := GetUsernameById(adminId, false)
	action := "开启"
	if !exempt {
		action = "关闭"
	}
	RecordLog(userId, LogTypeManage, fmt.Sprintf("管理员 %s (ID: %d) %s了用户的风控豁免", adminUsername, adminId, action))

	common.SysLog(fmt.Sprintf("风控: 管理员 %s %s了用户 ID: %d 的风控豁免", adminUsername, action, userId))

	return nil
}
