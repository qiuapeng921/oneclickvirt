package utils

import (
	"sync"
	"time"
)

// LogRateLimiter 日志速率限制器
type LogRateLimiter struct {
	limits map[string]*rateLimitEntry
	mu     sync.RWMutex
}

type rateLimitEntry struct {
	lastLog   time.Time
	count     int64
	interval  time.Duration
	threshold int64
}

var globalLogRateLimiter = &LogRateLimiter{
	limits: make(map[string]*rateLimitEntry),
}

// GetLogRateLimiter 获取全局日志速率限制器
func GetLogRateLimiter() *LogRateLimiter {
	return globalLogRateLimiter
}

// ShouldLog 检查是否应该记录日志
func (l *LogRateLimiter) ShouldLog(key string, interval time.Duration, threshold int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	entry, exists := l.limits[key]

	if !exists {
		l.limits[key] = &rateLimitEntry{
			lastLog:   now,
			count:     1,
			interval:  interval,
			threshold: threshold,
		}
		return true
	}

	// 检查是否在同一时间窗口内
	if now.Sub(entry.lastLog) < entry.interval {
		entry.count++
		return entry.count <= entry.threshold
	}

	// 新的时间窗口，重置计数
	entry.lastLog = now
	entry.count = 1
	return true
}

// ShouldLogWithMessage 检查包含消息内容的日志是否应该记录
func (l *LogRateLimiter) ShouldLogWithMessage(message string, interval time.Duration) bool {
	return l.ShouldLog(message, interval, 1) // 相同消息在间隔内只记录一次
}

// CleanupOldEntries 清理旧的限制条目
func (l *LogRateLimiter) CleanupOldEntries() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for key, entry := range l.limits {
		// 如果条目超过1小时没有使用，删除它
		if now.Sub(entry.lastLog) > time.Hour {
			delete(l.limits, key)
		}
	}
}

// StartCleanupTask 启动清理任务
func (l *LogRateLimiter) StartCleanupTask() {
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			l.CleanupOldEntries()
		}
	}()
}
