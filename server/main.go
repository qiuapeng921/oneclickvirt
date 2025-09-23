package main

import (
	"context"
	"fmt"
	"oneclickvirt/service/auth"
	"oneclickvirt/service/log"
	"oneclickvirt/service/scheduler"
	"oneclickvirt/service/storage"
	"oneclickvirt/service/task"
	userProviderService "oneclickvirt/service/user/provider"
	"oneclickvirt/service/vnstat"
	"time"

	"oneclickvirt/core"
	"oneclickvirt/global"
	"oneclickvirt/initialize"

	_ "oneclickvirt/docs"
	_ "oneclickvirt/provider/docker"
	_ "oneclickvirt/provider/incus"
	_ "oneclickvirt/provider/lxd"
	_ "oneclickvirt/provider/proxmox"

	"go.uber.org/zap"
)

// @title OneClickVirt API
// @version 1.0
// @description 一键虚拟化管理平台API接口文档
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8888
// @BasePath /api/v1
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	initializeSystem()
	runServer()
}

func initializeSystem() {
	global.APP_VP = core.Viper()
	global.APP_LOG = core.Zap()
	zap.ReplaceGlobals(global.APP_LOG)

	global.APP_LOG.Info("系统初始化开始")

	// 初始化存储目录结构
	storageService := storage.GetStorageService()
	if err := storageService.InitializeStorage(); err != nil {
		global.APP_LOG.Error("存储目录初始化失败", zap.Error(err))
		// 不要panic，让应用继续运行，但记录错误
	} else {
		global.APP_LOG.Debug("存储目录初始化完成")
	}

	// 启动日志轮转定时任务
	if global.APP_CONFIG.Zap.RetentionDay > 0 {
		logRotationService := log.GetLogRotationService()

		// 启动定时清理任务（每天凌晨3点执行）
		go func() {
			for {
				now := time.Now()
				// 计算到下一个凌晨3点的时间
				next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
				if now.After(next) {
					next = next.Add(24 * time.Hour)
				}

				duration := next.Sub(now)
				global.APP_LOG.Info("日志清理任务已安排",
					zap.Time("nextRun", next),
					zap.Duration("duration", duration))

				time.Sleep(duration)

				// 执行日志清理
				global.APP_LOG.Info("开始执行日志清理任务")
				if err := logRotationService.CleanupOldLogs(); err != nil {
					global.APP_LOG.Error("日志清理失败", zap.Error(err))
				} else {
					global.APP_LOG.Info("日志清理完成")
				}

				// 压缩旧日志
				if err := logRotationService.CompressOldLogs(); err != nil {
					global.APP_LOG.Error("日志压缩失败", zap.Error(err))
				} else {
					global.APP_LOG.Info("日志压缩完成")
				}
			}
		}()
	}

	global.APP_DB = initialize.Gorm()
	if global.APP_DB != nil {
		initialize.RegisterTables(global.APP_DB)
		initialize.InitializeConfigManager()
		global.APP_LOG.Debug("数据库连接和表注册完成")
		global.APP_LOG.Info("数据库初始化完成")

		// 初始化JWT密钥管理服务
		jwtService := &auth.JWTKeyService{}
		if err := jwtService.InitializeJWTKeys(); err != nil {
			global.APP_LOG.Error("JWT密钥服务初始化失败", zap.Error(err))
		} else {
			global.APP_LOG.Debug("JWT密钥管理服务初始化完成")
		}

		// Provider服务现在采用按需连接，不再预加载
		global.APP_LOG.Debug("Provider服务配置为按需连接模式")
	} else {
		global.APP_LOG.Warn("数据库初始化失败，系统将在待初始化状态运行")
		global.APP_LOG.Info("请访问 /init 页面进行系统初始化")
		// 不要panic，让应用继续运行，但数据库相关功能会受限
	}
	taskService := task.GetTaskService()

	// 设置全局任务服务实例，避免循环依赖
	userProviderService.SetGlobalTaskService(taskService)

	schedulerService := scheduler.NewSchedulerService(taskService)
	global.APP_SCHEDULER = schedulerService
	schedulerService.StartScheduler()

	// 启动监控调度器
	vnstatService := vnstat.NewService()
	monitoringSchedulerService := scheduler.NewMonitoringSchedulerService(vnstatService)
	global.APP_MONITORING_SCHEDULER = monitoringSchedulerService
	monitoringSchedulerService.Start(context.Background())

	global.APP_LOG.Info("系统初始化完成")
}

func runServer() {
	// 使用统一权限架构路由
	Router := initialize.Routers()
	global.APP_LOG.Debug("路由初始化完成，使用统一权限架构")
	address := fmt.Sprintf(":%d", global.APP_CONFIG.System.Addr)
	s := initialize.InitServer(address, Router)
	global.APP_LOG.Info("服务器启动成功", zap.String("address", address))
	global.APP_LOG.Info("API文档地址", zap.String("url", fmt.Sprintf("http://127.0.0.1%s/swagger/index.html", address)))

	if err := s.ListenAndServe(); err != nil {
		global.APP_LOG.Fatal("服务器启动失败", zap.Error(err))
	}
}
