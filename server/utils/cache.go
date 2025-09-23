package utils

import (
	"sync"
	"time"
)

// CaptchaCache 验证码缓存接口
type CaptchaCache interface {
	// SetCaptcha 设置验证码
	SetCaptcha(id string, code string, expiration time.Duration) error
	// GetCaptcha 获取验证码
	GetCaptcha(id string) (string, bool)
	// DeleteCaptcha 删除验证码
	DeleteCaptcha(id string) error
}

// MemoryCaptchaCache 内存验证码缓存实现
type MemoryCaptchaCache struct {
	data  map[string]cacheItem
	mutex sync.RWMutex
}

type cacheItem struct {
	value      string
	expiration time.Time
}

// NewMemoryCaptchaCache 创建新的内存验证码缓存
func NewMemoryCaptchaCache() *MemoryCaptchaCache {
	cache := &MemoryCaptchaCache{
		data: make(map[string]cacheItem),
	}

	// 启动定期清理过期缓存的协程
	go cache.cleanupLoop()

	return cache
}

// SetCaptcha 设置验证码
func (c *MemoryCaptchaCache) SetCaptcha(id string, code string, expiration time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[id] = cacheItem{
		value:      code,
		expiration: time.Now().Add(expiration),
	}

	return nil
}

// GetCaptcha 获取验证码
func (c *MemoryCaptchaCache) GetCaptcha(id string) (string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, exists := c.data[id]
	if !exists {
		return "", false
	}

	// 检查是否过期
	if time.Now().After(item.expiration) {
		return "", false
	}

	return item.value, true
}

// DeleteCaptcha 删除验证码
func (c *MemoryCaptchaCache) DeleteCaptcha(id string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, id)
	return nil
}

// cleanupLoop 定期清理过期缓存
func (c *MemoryCaptchaCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup 清理过期缓存
func (c *MemoryCaptchaCache) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, item := range c.data {
		if now.After(item.expiration) {
			delete(c.data, key)
		}
	}
}
