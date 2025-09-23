package database

import (
	"context"
	"sync"
	"time"

	"oneclickvirt/global"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DatabaseService 数据库服务抽象层 - 专为MySQL优化
type DatabaseService struct {
	mutex sync.RWMutex
}

var (
	dbService     *DatabaseService
	dbServiceOnce sync.Once
)

// GetDatabaseService 获取数据库服务单例
func GetDatabaseService() *DatabaseService {
	dbServiceOnce.Do(func() {
		dbService = &DatabaseService{}
	})
	return dbService
}

// ExecuteInTransaction 在事务中执行操作
func (ds *DatabaseService) ExecuteInTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}

// ExecuteWithTimeout 带超时的数据库操作
func (ds *DatabaseService) ExecuteWithTimeout(db *gorm.DB, timeout time.Duration, fn func(tx *gorm.DB) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return db.WithContext(ctx).Transaction(fn)
}

// ExecuteTransaction 执行事务（避免嵌套事务）
func (ds *DatabaseService) ExecuteTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	db := ds.getDB()
	if db == nil {
		global.APP_LOG.Error("数据库连接不可用")
		return gorm.ErrInvalidDB
	}

	err := ds.ExecuteInTransaction(db, fn)
	if err != nil {
		global.APP_LOG.Debug("数据库事务执行失败", zap.String("error", err.Error()))
	}
	return err
}

// ExecuteQuery 执行查询操作
func (ds *DatabaseService) ExecuteQuery(ctx context.Context, fn func() error) error {
	return fn()
}

// getDB 获取数据库连接（内部使用）
func (ds *DatabaseService) getDB() *gorm.DB {
	// 导入全局包以获取数据库连接
	return global.APP_DB
}
