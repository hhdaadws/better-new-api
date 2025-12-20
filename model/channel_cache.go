package model

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

var group2model2channels map[string]map[string][]int // enabled channel
var channelsIDM map[int]*Channel                     // all channels include disabled
var channelSyncLock sync.RWMutex

func InitChannelCache() {
	if !common.MemoryCacheEnabled {
		return
	}
	newChannelId2channel := make(map[int]*Channel)
	var channels []*Channel
	DB.Find(&channels)
	for _, channel := range channels {
		newChannelId2channel[channel.Id] = channel
	}
	var abilities []*Ability
	DB.Find(&abilities)
	groups := make(map[string]bool)
	for _, ability := range abilities {
		groups[ability.Group] = true
	}
	newGroup2model2channels := make(map[string]map[string][]int)
	for group := range groups {
		newGroup2model2channels[group] = make(map[string][]int)
	}
	for _, channel := range channels {
		if channel.Status != common.ChannelStatusEnabled {
			continue // skip disabled channels
		}
		groups := strings.Split(channel.Group, ",")
		for _, group := range groups {
			models := strings.Split(channel.Models, ",")
			for _, model := range models {
				if _, ok := newGroup2model2channels[group][model]; !ok {
					newGroup2model2channels[group][model] = make([]int, 0)
				}
				newGroup2model2channels[group][model] = append(newGroup2model2channels[group][model], channel.Id)
			}
		}
	}

	// sort by priority
	for group, model2channels := range newGroup2model2channels {
		for model, channels := range model2channels {
			sort.Slice(channels, func(i, j int) bool {
				return newChannelId2channel[channels[i]].GetPriority() > newChannelId2channel[channels[j]].GetPriority()
			})
			newGroup2model2channels[group][model] = channels
		}
	}

	channelSyncLock.Lock()
	group2model2channels = newGroup2model2channels
	//channelsIDM = newChannelId2channel
	for i, channel := range newChannelId2channel {
		if channel.ChannelInfo.IsMultiKey {
			channel.Keys = channel.GetKeys()
			if channel.ChannelInfo.MultiKeyMode == constant.MultiKeyModePolling {
				if oldChannel, ok := channelsIDM[i]; ok {
					// 存在旧的渠道，如果是多key且轮询，保留轮询索引信息
					if oldChannel.ChannelInfo.IsMultiKey && oldChannel.ChannelInfo.MultiKeyMode == constant.MultiKeyModePolling {
						channel.ChannelInfo.MultiKeyPollingIndex = oldChannel.ChannelInfo.MultiKeyPollingIndex
					}
				}
			}
		}
	}
	channelsIDM = newChannelId2channel
	channelSyncLock.Unlock()
	common.SysLog("channels synced from database")
}

func SyncChannelCache(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		common.SysLog("syncing channels from database")
		InitChannelCache()
	}
}

func GetRandomSatisfiedChannel(group string, model string, retry int) (*Channel, error) {
	// if memory cache is disabled, get channel directly from database
	if !common.MemoryCacheEnabled {
		return GetChannel(group, model, retry)
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	// First, try to find channels with the exact model name.
	channels := group2model2channels[group][model]

	// If no channels found, try to find channels with the normalized model name.
	if len(channels) == 0 {
		normalizedModel := ratio_setting.FormatMatchingModelName(model)
		channels = group2model2channels[group][normalizedModel]
	}

	if len(channels) == 0 {
		return nil, nil
	}

	if len(channels) == 1 {
		if channel, ok := channelsIDM[channels[0]]; ok {
			return channel, nil
		}
		return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channels[0])
	}

	uniquePriorities := make(map[int]bool)
	for _, channelId := range channels {
		if channel, ok := channelsIDM[channelId]; ok {
			uniquePriorities[int(channel.GetPriority())] = true
		} else {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channelId)
		}
	}
	var sortedUniquePriorities []int
	for priority := range uniquePriorities {
		sortedUniquePriorities = append(sortedUniquePriorities, priority)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sortedUniquePriorities)))

	if retry >= len(uniquePriorities) {
		retry = len(uniquePriorities) - 1
	}
	targetPriority := int64(sortedUniquePriorities[retry])

	// get the priority for the given retry number
	var sumWeight = 0
	var targetChannels []*Channel
	for _, channelId := range channels {
		if channel, ok := channelsIDM[channelId]; ok {
			if channel.GetPriority() == targetPriority {
				sumWeight += channel.GetWeight()
				targetChannels = append(targetChannels, channel)
			}
		} else {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channelId)
		}
	}

	// Filter out temporarily excluded channels
	targetChannels = filterExcludedChannels(targetChannels)

	if len(targetChannels) == 0 {
		return nil, errors.New(fmt.Sprintf("no channel found, group: %s, model: %s, priority: %d", group, model, targetPriority))
	}

	// smoothing factor and adjustment
	smoothingFactor := 1
	smoothingAdjustment := 0

	if sumWeight == 0 {
		// when all channels have weight 0, set sumWeight to the number of channels and set smoothing adjustment to 100
		// each channel's effective weight = 100
		sumWeight = len(targetChannels) * 100
		smoothingAdjustment = 100
	} else if sumWeight/len(targetChannels) < 10 {
		// when the average weight is less than 10, set smoothing factor to 100
		smoothingFactor = 100
	}

	// Calculate the total weight of all channels up to endIdx
	totalWeight := sumWeight * smoothingFactor

	// Generate a random value in the range [0, totalWeight)
	randomWeight := rand.Intn(totalWeight)

	// Find a channel based on its weight
	for _, channel := range targetChannels {
		randomWeight -= channel.GetWeight()*smoothingFactor + smoothingAdjustment
		if randomWeight < 0 {
			return channel, nil
		}
	}
	// return null if no channel is not found
	return nil, errors.New("channel not found")
}

func CacheGetChannel(id int) (*Channel, error) {
	if !common.MemoryCacheEnabled {
		return GetChannelById(id, true)
	}
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	c, ok := channelsIDM[id]
	if !ok {
		return nil, fmt.Errorf("渠道# %d，已不存在", id)
	}
	return c, nil
}

func CacheGetChannelInfo(id int) (*ChannelInfo, error) {
	if !common.MemoryCacheEnabled {
		channel, err := GetChannelById(id, true)
		if err != nil {
			return nil, err
		}
		return &channel.ChannelInfo, nil
	}
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	c, ok := channelsIDM[id]
	if !ok {
		return nil, fmt.Errorf("渠道# %d，已不存在", id)
	}
	return &c.ChannelInfo, nil
}

func CacheUpdateChannelStatus(id int, status int) {
	if !common.MemoryCacheEnabled {
		return
	}
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()
	if channel, ok := channelsIDM[id]; ok {
		channel.Status = status
	}
	if status != common.ChannelStatusEnabled {
		// delete the channel from group2model2channels
		for group, model2channels := range group2model2channels {
			for model, channels := range model2channels {
				for i, channelId := range channels {
					if channelId == id {
						// remove the channel from the slice
						group2model2channels[group][model] = append(channels[:i], channels[i+1:]...)
						break
					}
				}
			}
		}
	}
}

func CacheUpdateChannel(channel *Channel) {
	if !common.MemoryCacheEnabled {
		return
	}
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()
	if channel == nil {
		return
	}

	println("CacheUpdateChannel:", channel.Id, channel.Name, channel.Status, channel.ChannelInfo.MultiKeyPollingIndex)

	println("before:", channelsIDM[channel.Id].ChannelInfo.MultiKeyPollingIndex)
	channelsIDM[channel.Id] = channel
	println("after :", channelsIDM[channel.Id].ChannelInfo.MultiKeyPollingIndex)
}

// GetRandomSatisfiedChannelWithStickySession selects a channel considering sticky session bindings
// If sessionId is provided and a valid binding exists, returns the bound channel
// Otherwise, selects a new channel considering sticky session capacity
func GetRandomSatisfiedChannelWithStickySession(group string, model string, retry int, sessionId string) (*Channel, error) {
	// If sessionId is provided and Redis is enabled, check for existing binding
	if sessionId != "" && common.RedisEnabled {
		channelId, err := common.GetStickySessionChannel(group, model, sessionId)
		if err == nil && channelId > 0 {
			// Found existing binding, verify channel is still valid
			channel, err := validateAndGetStickyChannel(channelId, group, model)
			if err == nil && channel != nil {
				// Renew TTL if needed
				channelSetting := channel.GetSetting()
				ttl := channelSetting.StickySessionTTLMinutes
				if ttl <= 0 {
					ttl = 60
				}
				_ = common.RenewStickySessionTTL(group, model, sessionId, ttl)
				if common.DebugEnabled {
					common.SysLog(fmt.Sprintf("Sticky session hit: sessionId=%s, channelId=%d", sessionId, channelId))
				}
				return channel, nil
			}
			// Channel not valid, delete the binding and fall through to normal selection
			if common.DebugEnabled {
				common.SysLog(fmt.Sprintf("Sticky session invalid, deleting: sessionId=%s, channelId=%d", sessionId, channelId))
			}
			_ = common.DeleteStickySession(group, model, sessionId, channelId)
		}
	}

	// Normal channel selection with sticky session capacity consideration
	return getRandomSatisfiedChannelWithCapacity(group, model, retry, sessionId)
}

// validateAndGetStickyChannel checks if a sticky session's channel is still valid
func validateAndGetStickyChannel(channelId int, group, model string) (*Channel, error) {
	if !common.MemoryCacheEnabled {
		channel, err := GetChannelById(channelId, true)
		if err != nil {
			return nil, err
		}
		if channel.Status != common.ChannelStatusEnabled {
			return nil, errors.New("channel disabled")
		}
		// Check if channel supports the model and group
		if !channelSupportsModelAndGroup(channel, group, model) {
			return nil, errors.New("channel does not support model or group")
		}
		// Check if sticky session is enabled for this channel
		setting := channel.GetSetting()
		if !setting.StickySessionEnabled {
			return nil, errors.New("sticky session disabled for this channel")
		}
		return channel, nil
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	channel, ok := channelsIDM[channelId]
	if !ok || channel.Status != common.ChannelStatusEnabled {
		return nil, errors.New("channel not found or disabled")
	}

	// Check model and group support
	if !channelSupportsModelAndGroup(channel, group, model) {
		return nil, errors.New("channel does not support model or group")
	}

	// Check if sticky session is enabled for this channel
	setting := channel.GetSetting()
	if !setting.StickySessionEnabled {
		return nil, errors.New("sticky session disabled for this channel")
	}

	return channel, nil
}

// channelSupportsModelAndGroup checks if a channel supports the given model and group
func channelSupportsModelAndGroup(channel *Channel, group, model string) bool {
	// Check group
	groups := strings.Split(channel.Group, ",")
	groupFound := false
	for _, g := range groups {
		if strings.TrimSpace(g) == group {
			groupFound = true
			break
		}
	}
	if !groupFound {
		return false
	}

	// Check model
	models := strings.Split(channel.Models, ",")
	normalizedModel := ratio_setting.FormatMatchingModelName(model)
	for _, m := range models {
		if strings.TrimSpace(m) == model || strings.TrimSpace(m) == normalizedModel {
			return true
		}
	}
	return false
}

// getRandomSatisfiedChannelWithCapacity selects channel considering sticky session capacity
func getRandomSatisfiedChannelWithCapacity(group string, model string, retry int, sessionId string) (*Channel, error) {
	if !common.MemoryCacheEnabled {
		return GetChannel(group, model, retry)
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	channels := group2model2channels[group][model]
	if len(channels) == 0 {
		normalizedModel := ratio_setting.FormatMatchingModelName(model)
		channels = group2model2channels[group][normalizedModel]
	}

	if len(channels) == 0 {
		return nil, nil
	}

	// Build priority groups
	uniquePriorities := make(map[int]bool)
	for _, channelId := range channels {
		if channel, ok := channelsIDM[channelId]; ok {
			uniquePriorities[int(channel.GetPriority())] = true
		}
	}

	var sortedUniquePriorities []int
	for priority := range uniquePriorities {
		sortedUniquePriorities = append(sortedUniquePriorities, priority)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sortedUniquePriorities)))

	// Try each priority level starting from retry
	for priorityIdx := retry; priorityIdx < len(sortedUniquePriorities); priorityIdx++ {
		targetPriority := int64(sortedUniquePriorities[priorityIdx])

		// Collect channels at this priority level
		var targetChannels []*Channel
		for _, channelId := range channels {
			if channel, ok := channelsIDM[channelId]; ok {
				if channel.GetPriority() == targetPriority {
					targetChannels = append(targetChannels, channel)
				}
			}
		}

		if len(targetChannels) == 0 {
			continue
		}

		// Filter out temporarily excluded channels first
		targetChannels = filterExcludedChannels(targetChannels)
		if len(targetChannels) == 0 {
			continue
		}

		// If sessionId is provided, filter by sticky session capacity
		if sessionId != "" && common.RedisEnabled {
			availableChannels := filterBySessionCapacity(targetChannels)
			if len(availableChannels) > 0 {
				// Use binding-count-based selection for even distribution when all channels
				// have weight 0 and sticky session enabled
				if shouldUseBindingCountSelection(availableChannels) {
					return selectByBindingCount(availableChannels)
				}
				return selectByWeight(availableChannels)
			}
			// No available channels at this priority, try next priority level
			continue
		}

		// No sessionId or Redis not enabled, use normal selection
		return selectByWeight(targetChannels)
	}

	// If we've exhausted all priority levels with capacity filtering,
	// fall back to normal selection without capacity filtering
	if sessionId != "" && common.RedisEnabled && common.DebugEnabled {
		common.SysLog("All channels at capacity, falling back to normal selection")
	}
	return GetRandomSatisfiedChannel(group, model, retry)
}

// filterExcludedChannels filters out channels that are temporarily excluded due to session concurrency errors
func filterExcludedChannels(channels []*Channel) []*Channel {
	result := make([]*Channel, 0, len(channels))
	for _, channel := range channels {
		setting := channel.GetSetting()
		// Only check exclusion status for channels with auto-exclude enabled
		if setting.SessionConcurrencyAutoExclude && common.IsChannelSessionExcluded(channel.Id) {
			if common.DebugEnabled {
				common.SysLog(fmt.Sprintf("Channel %d temporarily excluded from dispatch", channel.Id))
			}
			continue // Skip excluded channel
		}
		result = append(result, channel)
	}
	return result
}

// filterBySessionCapacity filters channels that have available session slots
func filterBySessionCapacity(channels []*Channel) []*Channel {
	result := make([]*Channel, 0)
	for _, channel := range channels {
		setting := channel.GetSetting()

		// Channel doesn't use sticky sessions, always available
		if !setting.StickySessionEnabled {
			result = append(result, channel)
			continue
		}

		// Unlimited sessions
		if setting.StickySessionMaxCount <= 0 {
			result = append(result, channel)
			continue
		}

		// Check current session count
		currentCount, err := common.GetChannelStickySessionCount(channel.Id)
		if err != nil {
			// Error getting count, skip this channel
			continue
		}

		if currentCount < setting.StickySessionMaxCount {
			result = append(result, channel)
		}
	}
	return result
}

// shouldUseBindingCountSelection checks if binding-count-based selection should be used
// Returns true when all channels have weight 0 and sticky session enabled
func shouldUseBindingCountSelection(channels []*Channel) bool {
	if len(channels) <= 1 {
		return false
	}
	for _, channel := range channels {
		if channel.GetWeight() != 0 {
			return false
		}
		setting := channel.GetSetting()
		if !setting.StickySessionEnabled {
			return false
		}
	}
	return true
}

// selectByBindingCount selects channel with fewest sticky session bindings
// This ensures even distribution of sticky sessions across channels
func selectByBindingCount(channels []*Channel) (*Channel, error) {
	if len(channels) == 0 {
		return nil, errors.New("no channels available")
	}
	if len(channels) == 1 {
		return channels[0], nil
	}

	type channelWithCount struct {
		channel *Channel
		count   int
	}

	channelCounts := make([]channelWithCount, 0, len(channels))
	for _, channel := range channels {
		count := 0
		if common.RedisEnabled {
			c, err := common.GetChannelStickySessionCount(channel.Id)
			if err == nil {
				count = c
			}
		}
		channelCounts = append(channelCounts, channelWithCount{channel: channel, count: count})
	}

	// Find minimum binding count
	minCount := channelCounts[0].count
	for _, cc := range channelCounts[1:] {
		if cc.count < minCount {
			minCount = cc.count
		}
	}

	// Filter channels with minimum binding count
	candidates := make([]*Channel, 0)
	for _, cc := range channelCounts {
		if cc.count == minCount {
			candidates = append(candidates, cc.channel)
		}
	}

	// Random selection among candidates with equal min count
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	return candidates[rand.Intn(len(candidates))], nil
}

// selectByWeight selects a channel from the list based on weight
func selectByWeight(channels []*Channel) (*Channel, error) {
	if len(channels) == 0 {
		return nil, errors.New("no channels available")
	}

	if len(channels) == 1 {
		return channels[0], nil
	}

	// Calculate total weight
	var sumWeight = 0
	for _, channel := range channels {
		sumWeight += channel.GetWeight()
	}

	// smoothing factor and adjustment
	smoothingFactor := 1
	smoothingAdjustment := 0

	if sumWeight == 0 {
		sumWeight = len(channels) * 100
		smoothingAdjustment = 100
	} else if sumWeight/len(channels) < 10 {
		smoothingFactor = 100
	}

	totalWeight := sumWeight * smoothingFactor
	randomWeight := rand.Intn(totalWeight)

	for _, channel := range channels {
		randomWeight -= channel.GetWeight()*smoothingFactor + smoothingAdjustment
		if randomWeight < 0 {
			return channel, nil
		}
	}

	return channels[0], nil
}

// BindStickySession binds a session to a channel if sticky session is enabled
func BindStickySession(group, model, sessionId string, channel *Channel, username, tokenName string) error {
	if sessionId == "" || !common.RedisEnabled {
		return nil
	}

	setting := channel.GetSetting()
	if !setting.StickySessionEnabled {
		return nil
	}

	ttl := setting.StickySessionTTLMinutes
	if ttl <= 0 {
		ttl = 60 // Default 60 minutes
	}

	return common.SetStickySession(group, model, sessionId, channel.Id, ttl, username, tokenName)
}
