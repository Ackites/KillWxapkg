package config

import (
	"sync"
)

// SharedConfigManager 用于管理共享配置
type SharedConfigManager struct {
	mu       sync.RWMutex
	settings map[string]interface{}
}

var shareInstance *SharedConfigManager
var shareOnce sync.Once

// NewSharedConfigManager 创建一个新的SharedConfigManager
func NewSharedConfigManager() *SharedConfigManager {
	shareOnce.Do(func() {
		shareInstance = &SharedConfigManager{
			settings: make(map[string]interface{}),
		}
	})
	return shareInstance
}

// Set 设置一个配置项的值
func (scm *SharedConfigManager) Set(key string, value interface{}) {
	scm.mu.Lock()
	defer scm.mu.Unlock()
	scm.settings[key] = value
}

// SetBulk 批量设置配置项的值
func (scm *SharedConfigManager) SetBulk(configs map[string]interface{}) {
	scm.mu.Lock()
	defer scm.mu.Unlock()
	for key, value := range configs {
		scm.settings[key] = value
	}
}

// Get 获取一个配置项的值
func (scm *SharedConfigManager) Get(key string) (interface{}, bool) {
	scm.mu.RLock()
	defer scm.mu.RUnlock()
	value, exists := scm.settings[key]
	return value, exists
}

// GetBulk 批量获取配置项的值
func (scm *SharedConfigManager) GetBulk(keys []string) map[string]interface{} {
	scm.mu.RLock()
	defer scm.mu.RUnlock()
	results := make(map[string]interface{})
	for _, key := range keys {
		if value, exists := scm.settings[key]; exists {
			results[key] = value
		}
	}
	return results
}

// Delete 删除一个配置项
func (scm *SharedConfigManager) Delete(key string) {
	scm.mu.Lock()
	defer scm.mu.Unlock()
	delete(scm.settings, key)
}

// GetAll 返回所有配置项的副本
func (scm *SharedConfigManager) GetAll() map[string]interface{} {
	scm.mu.RLock()
	defer scm.mu.RUnlock()
	c := make(map[string]interface{})
	for key, value := range scm.settings {
		c[key] = value
	}
	return c
}
