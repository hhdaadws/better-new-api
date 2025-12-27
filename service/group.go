package service

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func GetUserUsableGroups(userGroup string) map[string]string {
	groupsCopy := setting.GetUserUsableGroupsCopy()
	if userGroup != "" {
		specialSettings, b := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.Get(userGroup)
		if b {
			// 处理特殊可用分组
			for specialGroup, desc := range specialSettings {
				if strings.HasPrefix(specialGroup, "-:") {
					// 移除分组
					groupToRemove := strings.TrimPrefix(specialGroup, "-:")
					delete(groupsCopy, groupToRemove)
				} else if strings.HasPrefix(specialGroup, "+:") {
					// 添加分组
					groupToAdd := strings.TrimPrefix(specialGroup, "+:")
					groupsCopy[groupToAdd] = desc
				} else {
					// 直接添加分组
					groupsCopy[specialGroup] = desc
				}
			}
		}
		// 如果userGroup不在UserUsableGroups中，返回UserUsableGroups + userGroup
		if _, ok := groupsCopy[userGroup]; !ok {
			groupsCopy[userGroup] = "用户分组"
		}
	}
	return groupsCopy
}

// GetUserUsableGroupsWithExclusive 获取用户可用分组，包含专属分组
// userGroup: 用户的分组
// userId: 用户ID，用于检查用户是否启用了专属分组
func GetUserUsableGroupsWithExclusive(userGroup string, userId int) map[string]string {
	groupsCopy := GetUserUsableGroups(userGroup)

	// 检查用户是否启用了专属分组
	if userId > 0 {
		exclusiveGroup, hasExclusive := GetUserExclusiveGroup(userId)
		if hasExclusive {
			groupsCopy[exclusiveGroup] = "专属分组"
		}
	}

	return groupsCopy
}

// GetUserExclusiveGroup 获取用户的专属分组名
// 如果用户启用了专属分组，返回专属分组名和 true
// 否则返回空字符串和 false
func GetUserExclusiveGroup(userId int) (string, bool) {
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		return "", false
	}
	if !userCache.EnableExclusiveGroup {
		return "", false
	}
	return model.GetExclusiveGroupName(userId), true
}

// GetUserExclusiveGroupRatio 获取用户的专属分组倍率
func GetUserExclusiveGroupRatio(userId int) float64 {
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		return 1.0
	}
	if userCache.ExclusiveGroupRatio <= 0 {
		return 1.0
	}
	return userCache.ExclusiveGroupRatio
}

// GetUserExclusiveGroupStatus 获取用户专属分组的状态信息
// 返回：是否有专属分组权限、是否配置了渠道、专属分组名、专属分组倍率
func GetUserExclusiveGroupStatus(userId int) (hasPermission bool, hasChannels bool, groupName string, ratio float64) {
	exclusiveGroup, hasExclusive := GetUserExclusiveGroup(userId)
	if !hasExclusive {
		return false, false, "", 1.0
	}

	hasChannels, _ = model.HasExclusiveGroupChannels(userId)
	ratio = GetUserExclusiveGroupRatio(userId)
	return true, hasChannels, exclusiveGroup, ratio
}

func GroupInUserUsableGroups(userGroup, groupName string) bool {
	_, ok := GetUserUsableGroups(userGroup)[groupName]
	return ok
}

// GroupInUserUsableGroupsWithExclusive 检查分组是否在用户可用分组中（包含专属分组）
func GroupInUserUsableGroupsWithExclusive(userGroup, groupName string, userId int) bool {
	// 先检查普通分组
	if _, ok := GetUserUsableGroups(userGroup)[groupName]; ok {
		return true
	}
	// 检查是否为用户的专属分组
	if model.IsExclusiveGroup(groupName) {
		exclusiveUserId, ok := model.GetExclusiveGroupUserId(groupName)
		if ok && exclusiveUserId == userId {
			// 确认用户有专属分组权限
			_, hasExclusive := GetUserExclusiveGroup(userId)
			return hasExclusive
		}
	}
	return false
}

// GetUserAutoGroup 根据用户分组获取自动分组设置
func GetUserAutoGroup(userGroup string) []string {
	groups := GetUserUsableGroups(userGroup)
	autoGroups := make([]string, 0)
	for _, group := range setting.GetAutoGroups() {
		if _, ok := groups[group]; ok {
			autoGroups = append(autoGroups, group)
		}
	}
	return autoGroups
}

// GetUserGroupRatio 获取用户使用某个分组的倍率
// userGroup 用户分组
// group 需要获取倍率的分组
func GetUserGroupRatio(userGroup, group string) float64 {
	ratio, ok := ratio_setting.GetGroupGroupRatio(userGroup, group)
	if ok {
		return ratio
	}
	return ratio_setting.GetGroupRatio(group)
}
