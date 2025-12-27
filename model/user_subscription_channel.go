package model

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// UserSubscriptionChannel 用户订阅渠道绑定
// 存储用户专属分组下的渠道绑定关系
type UserSubscriptionChannel struct {
	Id                 int   `json:"id" gorm:"primaryKey"`
	UserSubscriptionId int   `json:"user_subscription_id" gorm:"index;not null"`
	UserId             int   `json:"user_id" gorm:"index;not null"`
	ChannelId          int   `json:"channel_id" gorm:"index;not null"`
	CreatedTime        int64 `json:"created_time" gorm:"bigint"`
	UpdatedTime        int64 `json:"updated_time" gorm:"bigint"`

	// 非数据库字段，用于 API 返回
	ChannelInfo *Channel `json:"channel_info,omitempty" gorm:"-"`
}

// ExclusiveGroupPrefix 专属分组名前缀
const ExclusiveGroupPrefix = "sub_user_"

// GetExclusiveGroupName 获取用户的专属分组名
func GetExclusiveGroupName(userId int) string {
	return fmt.Sprintf("%s%d", ExclusiveGroupPrefix, userId)
}

// IsExclusiveGroup 判断分组名是否为专属分组
func IsExclusiveGroup(groupName string) bool {
	return strings.HasPrefix(groupName, ExclusiveGroupPrefix)
}

// GetExclusiveGroupUserId 从专属分组名中提取用户ID
func GetExclusiveGroupUserId(groupName string) (int, bool) {
	if !IsExclusiveGroup(groupName) {
		return 0, false
	}
	userIdStr := strings.TrimPrefix(groupName, ExclusiveGroupPrefix)
	var userId int
	_, err := fmt.Sscanf(userIdStr, "%d", &userId)
	if err != nil {
		return 0, false
	}
	return userId, true
}

// Insert 插入用户订阅渠道绑定记录
func (usc *UserSubscriptionChannel) Insert() error {
	usc.CreatedTime = common.GetTimestamp()
	usc.UpdatedTime = usc.CreatedTime

	err := DB.Create(usc).Error
	if err != nil {
		return err
	}

	// 更新渠道的 Group 字段，添加专属分组
	exclusiveGroup := GetExclusiveGroupName(usc.UserId)
	err = AddGroupToChannel(usc.ChannelId, exclusiveGroup)
	if err != nil {
		// 回滚：删除刚创建的记录
		DB.Delete(usc)
		return err
	}

	return nil
}

// Delete 删除用户订阅渠道绑定记录
func (usc *UserSubscriptionChannel) Delete() error {
	err := DB.Delete(usc).Error
	if err != nil {
		return err
	}

	// 检查该用户是否还有其他绑定到此渠道的记录
	var count int64
	DB.Model(&UserSubscriptionChannel{}).
		Where("user_id = ? AND channel_id = ?", usc.UserId, usc.ChannelId).
		Count(&count)

	// 如果没有其他绑定，从渠道的 Group 字段移除专属分组
	if count == 0 {
		exclusiveGroup := GetExclusiveGroupName(usc.UserId)
		RemoveGroupFromChannel(usc.ChannelId, exclusiveGroup)
	}

	return nil
}

// GetUserSubscriptionChannels 获取用户的所有专属渠道
func GetUserSubscriptionChannels(userId int) ([]*UserSubscriptionChannel, error) {
	var channels []*UserSubscriptionChannel
	err := DB.Where("user_id = ?", userId).Find(&channels).Error
	if err != nil {
		return nil, err
	}

	// 加载渠道信息
	for _, usc := range channels {
		channel, err := GetChannelById(usc.ChannelId, false)
		if err == nil {
			usc.ChannelInfo = channel
		}
	}

	return channels, nil
}

// GetUserSubscriptionChannelsBySubscriptionId 根据订阅ID获取渠道绑定
func GetUserSubscriptionChannelsBySubscriptionId(userSubscriptionId int) ([]*UserSubscriptionChannel, error) {
	var channels []*UserSubscriptionChannel
	err := DB.Where("user_subscription_id = ?", userSubscriptionId).Find(&channels).Error
	if err != nil {
		return nil, err
	}

	// 加载渠道信息
	for _, usc := range channels {
		channel, err := GetChannelById(usc.ChannelId, false)
		if err == nil {
			usc.ChannelInfo = channel
		}
	}

	return channels, nil
}

// DeleteUserSubscriptionChannel 删除指定用户的指定渠道绑定
func DeleteUserSubscriptionChannel(userId, channelId int) error {
	var usc UserSubscriptionChannel
	err := DB.Where("user_id = ? AND channel_id = ?", userId, channelId).First(&usc).Error
	if err != nil {
		return err
	}
	return usc.Delete()
}

// DeleteUserSubscriptionChannelsByUserId 删除用户的所有专属渠道绑定
func DeleteUserSubscriptionChannelsByUserId(userId int) error {
	// 先获取所有绑定的渠道ID
	var channelIds []int
	DB.Model(&UserSubscriptionChannel{}).
		Where("user_id = ?", userId).
		Pluck("channel_id", &channelIds)

	// 删除所有绑定记录
	err := DB.Where("user_id = ?", userId).Delete(&UserSubscriptionChannel{}).Error
	if err != nil {
		return err
	}

	// 从所有渠道的 Group 字段移除专属分组
	exclusiveGroup := GetExclusiveGroupName(userId)
	for _, channelId := range channelIds {
		RemoveGroupFromChannel(channelId, exclusiveGroup)
	}

	return nil
}

// HasExclusiveGroupChannels 检查用户的专属分组是否有配置渠道
func HasExclusiveGroupChannels(userId int) (bool, error) {
	var count int64
	err := DB.Model(&UserSubscriptionChannel{}).
		Where("user_id = ?", userId).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// AddGroupToChannel 向渠道添加分组
func AddGroupToChannel(channelId int, groupName string) error {
	channel, err := GetChannelById(channelId, false)
	if err != nil {
		return err
	}

	groups := strings.Split(channel.Group, ",")
	for _, g := range groups {
		if strings.TrimSpace(g) == groupName {
			// 分组已存在
			return nil
		}
	}

	// 添加分组
	if channel.Group == "" {
		channel.Group = groupName
	} else {
		channel.Group = channel.Group + "," + groupName
	}

	err = DB.Model(&Channel{}).Where("id = ?", channelId).Update("group", channel.Group).Error
	if err != nil {
		return err
	}

	// 刷新渠道缓存
	InitChannelCache()
	return nil
}

// RemoveGroupFromChannel 从渠道移除分组
func RemoveGroupFromChannel(channelId int, groupName string) error {
	channel, err := GetChannelById(channelId, false)
	if err != nil {
		return err
	}

	groups := strings.Split(channel.Group, ",")
	newGroups := make([]string, 0, len(groups))
	for _, g := range groups {
		g = strings.TrimSpace(g)
		if g != "" && g != groupName {
			newGroups = append(newGroups, g)
		}
	}

	newGroup := strings.Join(newGroups, ",")
	err = DB.Model(&Channel{}).Where("id = ?", channelId).Update("group", newGroup).Error
	if err != nil {
		return err
	}

	// 刷新渠道缓存
	InitChannelCache()
	return nil
}

// GetExclusiveGroupChannelIds 获取用户专属分组下的所有渠道ID
func GetExclusiveGroupChannelIds(userId int) ([]int, error) {
	var channelIds []int
	err := DB.Model(&UserSubscriptionChannel{}).
		Where("user_id = ?", userId).
		Pluck("channel_id", &channelIds).Error
	if err != nil {
		return nil, err
	}
	return channelIds, nil
}

// ExclusiveGroupUserInfo 专属分组用户信息
type ExclusiveGroupUserInfo struct {
	UserId              int     `json:"user_id"`
	Username            string  `json:"username"`
	DisplayName         string  `json:"display_name"`
	Email               string  `json:"email"`
	ExclusiveGroupRatio float64 `json:"exclusive_group_ratio"`
	ChannelCount        int     `json:"channel_count"`
}

// GetUsersWithExclusiveGroup 获取所有启用了专属分组的用户
func GetUsersWithExclusiveGroup() ([]*ExclusiveGroupUserInfo, error) {
	var results []*ExclusiveGroupUserInfo

	// 查询启用了专属分组的用户
	err := DB.Raw(`
		SELECT
			u.id as user_id,
			u.username,
			u.display_name,
			u.email,
			u.exclusive_group_ratio,
			COALESCE(usc.channel_count, 0) as channel_count
		FROM users u
		LEFT JOIN (
			SELECT user_id, COUNT(*) as channel_count
			FROM user_subscription_channels
			GROUP BY user_id
		) usc ON u.id = usc.user_id
		WHERE u.enable_exclusive_group = true
		AND u.deleted_at IS NULL
		ORDER BY u.id
	`).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}
