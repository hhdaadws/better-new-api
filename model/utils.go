package model

import (
	"errors"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

const (
	BatchUpdateTypeUserQuota = iota
	BatchUpdateTypeTokenQuota
	BatchUpdateTypeUsedQuota
	BatchUpdateTypeChannelUsedQuota
	BatchUpdateTypeRequestCount
	BatchUpdateTypeCount // if you add a new type, you need to add a new map and a new lock
)

// Memory-based batch update stores (fallback when Redis is not available)
var batchUpdateStores []map[int]int
var batchUpdateLocks []sync.Mutex

// Flag to track if batch updater is running
var batchUpdaterRunning bool
var batchUpdaterMutex sync.Mutex

func init() {
	for i := 0; i < BatchUpdateTypeCount; i++ {
		batchUpdateStores = append(batchUpdateStores, make(map[int]int))
		batchUpdateLocks = append(batchUpdateLocks, sync.Mutex{})
	}
}

func InitBatchUpdater() {
	batchUpdaterMutex.Lock()
	if batchUpdaterRunning {
		batchUpdaterMutex.Unlock()
		return
	}
	batchUpdaterRunning = true
	batchUpdaterMutex.Unlock()

	// Log which storage mode is being used
	if common.RedisEnabled {
		common.SysLog("batch updater initialized with Redis storage (data persisted across restarts)")
	} else {
		common.SysLog("batch updater initialized with memory storage (WARNING: data may be lost on restart)")
	}

	gopool.Go(func() {
		for {
			time.Sleep(time.Duration(common.BatchUpdateInterval) * time.Second)
			batchUpdate()
		}
	})
}

// addNewRecord adds a new record to the batch update queue
// When Redis is enabled, data is stored in Redis for persistence
// When Redis is not available, falls back to memory storage
func addNewRecord(type_ int, id int, value int) {
	if common.RedisEnabled {
		// Use Redis for persistent storage
		err := common.RedisBatchUpdateHIncrBy(type_, id, value)
		if err != nil {
			common.SysLog("failed to add batch update record to Redis: " + err.Error() + ", falling back to memory")
			// Fallback to memory storage
			addNewRecordToMemory(type_, id, value)
		}
	} else {
		// Use memory storage
		addNewRecordToMemory(type_, id, value)
	}
}

// addNewRecordToMemory adds a record to memory storage (original implementation)
func addNewRecordToMemory(type_ int, id int, value int) {
	batchUpdateLocks[type_].Lock()
	defer batchUpdateLocks[type_].Unlock()
	if _, ok := batchUpdateStores[type_][id]; !ok {
		batchUpdateStores[type_][id] = value
	} else {
		batchUpdateStores[type_][id] += value
	}
}

// batchUpdate processes all pending batch updates
func batchUpdate() {
	if common.RedisEnabled {
		batchUpdateFromRedis()
	} else {
		batchUpdateFromMemory()
	}
}

// batchUpdateFromRedis processes batch updates from Redis storage
func batchUpdateFromRedis() {
	// Check if there's any data to update
	hasData := false
	for i := 0; i < BatchUpdateTypeCount; i++ {
		has, err := common.RedisBatchUpdateHasData(i)
		if err != nil {
			common.SysLog("failed to check Redis batch update data: " + err.Error())
			continue
		}
		if has {
			hasData = true
			break
		}
	}

	if !hasData {
		return
	}

	common.SysLog("batch update started (Redis mode)")
	for i := 0; i < BatchUpdateTypeCount; i++ {
		// Atomically get and clear data to prevent data loss
		store, err := common.RedisBatchUpdateGetAndClear(i)
		if err != nil {
			common.SysLog("failed to get batch update data from Redis: " + err.Error())
			continue
		}

		for key, value := range store {
			processBatchUpdateRecord(i, key, value)
		}
	}
	common.SysLog("batch update finished (Redis mode)")
}

// batchUpdateFromMemory processes batch updates from memory storage (original implementation)
func batchUpdateFromMemory() {
	// check if there's any data to update
	hasData := false
	for i := 0; i < BatchUpdateTypeCount; i++ {
		batchUpdateLocks[i].Lock()
		if len(batchUpdateStores[i]) > 0 {
			hasData = true
			batchUpdateLocks[i].Unlock()
			break
		}
		batchUpdateLocks[i].Unlock()
	}

	if !hasData {
		return
	}

	common.SysLog("batch update started (memory mode)")
	for i := 0; i < BatchUpdateTypeCount; i++ {
		batchUpdateLocks[i].Lock()
		store := batchUpdateStores[i]
		batchUpdateStores[i] = make(map[int]int)
		batchUpdateLocks[i].Unlock()

		for key, value := range store {
			processBatchUpdateRecord(i, key, value)
		}
	}
	common.SysLog("batch update finished (memory mode)")
}

// processBatchUpdateRecord processes a single batch update record
func processBatchUpdateRecord(updateType int, key int, value int) {
	switch updateType {
	case BatchUpdateTypeUserQuota:
		err := increaseUserQuota(key, value)
		if err != nil {
			common.SysLog("failed to batch update user quota: " + err.Error())
		}
	case BatchUpdateTypeTokenQuota:
		err := increaseTokenQuota(key, value)
		if err != nil {
			common.SysLog("failed to batch update token quota: " + err.Error())
		}
	case BatchUpdateTypeUsedQuota:
		updateUserUsedQuota(key, value)
	case BatchUpdateTypeRequestCount:
		updateUserRequestCount(key, value)
	case BatchUpdateTypeChannelUsedQuota:
		updateChannelUsedQuota(key, value)
	}
}

// FlushBatchUpdates forces an immediate batch update
// This should be called before program shutdown to ensure no data is lost
func FlushBatchUpdates() {
	common.SysLog("flushing batch updates before shutdown...")
	batchUpdate()
	common.SysLog("batch updates flushed successfully")
}

// GetBatchUpdateStats returns statistics about pending batch updates
// Useful for monitoring and debugging
func GetBatchUpdateStats() map[string]int {
	stats := make(map[string]int)
	typeNames := []string{"UserQuota", "TokenQuota", "UsedQuota", "ChannelUsedQuota", "RequestCount"}

	if common.RedisEnabled {
		for i := 0; i < BatchUpdateTypeCount; i++ {
			data, err := common.RedisBatchUpdateGetAll(i)
			if err != nil {
				stats[typeNames[i]] = -1 // Error indicator
				continue
			}
			stats[typeNames[i]] = len(data)
		}
	} else {
		for i := 0; i < BatchUpdateTypeCount; i++ {
			batchUpdateLocks[i].Lock()
			stats[typeNames[i]] = len(batchUpdateStores[i])
			batchUpdateLocks[i].Unlock()
		}
	}

	return stats
}

func RecordExist(err error) (bool, error) {
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}

func shouldUpdateRedis(fromDB bool, err error) bool {
	return common.RedisEnabled && fromDB && err == nil
}
