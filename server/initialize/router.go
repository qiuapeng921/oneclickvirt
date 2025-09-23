package initialize

import (
	"net/http"

	"oneclickvirt/api/v1/admin"
	"oneclickvirt/api/v1/config"
	"oneclickvirt/api/v1/provider"
	"oneclickvirt/api/v1/public"
	"oneclickvirt/api/v1/system"
	"oneclickvirt/api/v1/traffic"
	"oneclickvirt/api/v1/user"
	"oneclickvirt/middleware"
	authModel "oneclickvirt/model/auth"
	"oneclickvirt/router"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Routers 新的统一路由架构
func Routers() *gin.Engine {
	Router := gin.Default()
	// CORS配置
	Router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length", "Authorization"},
		AllowCredentials: true,
	}))
	// 全局中间件
	Router.Use(middleware.ErrorHandler())
	Router.Use(middleware.InputValidator())
	// 健康检查 - 使用public包中的标准健康检查
	Router.GET("/health", public.HealthCheck)
	// Swagger文档路由
	Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// API路由组
	ApiGroup := Router.Group("/api")
	{
		// 健康检查也在API路径下，保持与前端一致
		ApiGroup.GET("/health", public.HealthCheck)
		// 公开访问路由
		PublicGroup := ApiGroup.Group("")
		PublicGroup.Use(middleware.RequireAuth(authModel.AuthLevelPublic))
		{
			PublicGroup.GET("/ping", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "pong"})
			})
			// 认证相关路由
			router.InitAuthRouter(PublicGroup)
			router.InitPublicRouter(PublicGroup)
		}
		// 统一配置API（替代原来的两套配置接口）
		ConfigGroup := ApiGroup.Group("/v1/config")
		ConfigGroup.Use(middleware.RequireAuth(authModel.AuthLevelAdmin))
		{
			ConfigGroup.GET("", config.GetUnifiedConfig)
			ConfigGroup.PUT("", config.UpdateUnifiedConfig)
		}
		// 普通用户路由
		UserGroup := ApiGroup.Group("/v1")
		UserGroup.Use(middleware.RequireAuth(authModel.AuthLevelUser))
		{
			// 用户管理
			UserGroup.GET("/user/profile", user.GetUserInfo)
			UserGroup.PUT("/user/profile", user.UpdateProfile)
			UserGroup.PUT("/user/reset-password", user.UserResetPassword)
			UserGroup.GET("/user/info", user.GetUserInfo)
			UserGroup.GET("/user/dashboard", user.GetUserDashboard)
			UserGroup.GET("/user/limits", user.GetUserLimits)
			UserGroup.GET("/user/instances", user.GetUserInstances)
			UserGroup.POST("/user/instances", user.CreateUserInstance)
			UserGroup.GET("/user/instances/:id", user.GetUserInstanceDetail)
			UserGroup.GET("/user/instances/:id/monitoring", user.GetInstanceMonitoring)
			UserGroup.GET("/user/instances/:id/vnstat/summary", user.GetInstanceVnStatSummary)
			UserGroup.GET("/user/instances/:id/vnstat/query", user.QueryInstanceVnStatData)
			UserGroup.GET("/user/instances/:id/vnstat/interfaces", user.GetInstanceVnStatInterfaces)
			UserGroup.GET("/user/instances/:id/vnstat/dashboard", user.GetInstanceVnStatDashboard)
			UserGroup.PUT("/user/instances/:id/reset-password", user.ResetInstancePassword)
			UserGroup.GET("/user/instances/:id/password/:taskId", user.GetInstanceNewPassword)
			UserGroup.GET("/user/instances/:id/ports", user.GetInstancePorts)
			UserGroup.POST("/user/instances/action", user.InstanceAction)
			UserGroup.GET("/user/port-mappings", user.GetUserPortMappings)
			UserGroup.GET("/user/resources/available", user.GetAvailableResources)
			UserGroup.POST("/user/resources/claim", user.ClaimResource)
			UserGroup.GET("/user/providers/available", user.GetAvailableProviders)
			UserGroup.GET("/user/images", user.GetUserSystemImages)
			UserGroup.GET("/user/images/filtered", user.GetFilteredSystemImages)
			UserGroup.GET("/user/providers/:id/capabilities", user.GetProviderCapabilities)
			UserGroup.GET("/user/instance-type-permissions", user.GetInstanceTypePermissions)
			UserGroup.GET("/user/instance-config", user.GetInstanceConfig)
			UserGroup.GET("/user/tasks", user.GetUserTasks)
			UserGroup.POST("/user/tasks/:taskId/cancel", user.CancelUserTask)

			// 流量统计API
			trafficAPI := &traffic.UserTrafficAPI{}
			UserGroup.GET("/user/traffic/overview", trafficAPI.GetTrafficOverview)
			UserGroup.GET("/user/traffic/instance/:instanceId", trafficAPI.GetInstanceTrafficDetail)
			UserGroup.GET("/user/traffic/instances", trafficAPI.GetInstancesTrafficSummary)
			UserGroup.GET("/user/traffic/limit-status", trafficAPI.GetTrafficLimitStatus)
			UserGroup.GET("/user/traffic/vnstat/:instanceId", trafficAPI.GetVnStatData)

			uploadGroup := UserGroup.Group("/upload")
			uploadGroup.Use(middleware.AvatarUploadLimit()) // 上传大小限制
			{
				uploadGroup.POST("/avatar", system.UploadAvatar)
			}
			UserGroup.GET("/dashboard/stats", public.GetDashboardStats)
			// 资源管理（普通用户只能管理自己的资源）
			UserGroup.GET("/instances", user.GetUserInstances)
			UserGroup.POST("/instances", user.CreateUserInstance)
			UserGroup.PUT("/instances/:id", admin.UpdateInstance)
			UserGroup.DELETE("/instances/:id", admin.DeleteInstance)
		}
		// 管理员路由
		AdminGroup := ApiGroup.Group("/v1/admin")
		AdminGroup.Use(middleware.RequireAuth(authModel.AuthLevelAdmin))
		{
			// 仪表盘
			AdminGroup.GET("/dashboard", admin.GetAdminDashboard)
			// 系统配置（管理员专用）
			AdminGroup.GET("/config", config.GetUnifiedConfig)
			AdminGroup.PUT("/config", config.UpdateUnifiedConfig)
			// 用户管理
			AdminGroup.GET("/users", admin.GetUserList)
			AdminGroup.POST("/users", admin.CreateUser)
			AdminGroup.PUT("/users/:id", admin.UpdateUser)
			AdminGroup.DELETE("/users/:id", admin.DeleteUser)
			AdminGroup.PUT("/users/:id/status", admin.UpdateUserStatus)
			AdminGroup.PUT("/users/:id/level", admin.UpdateUserLevel)
			AdminGroup.PUT("/users/:id/reset-password", admin.ResetUserPassword)
			AdminGroup.PUT("/users/batch-level", admin.AdminBatchUpdateUserLevel)
			AdminGroup.PUT("/users/batch-status", admin.AdminBatchUpdateUserStatus)
			AdminGroup.POST("/users/batch-delete", admin.AdminBatchDeleteUsers)
			// 系统管理
			AdminGroup.GET("/instances", admin.GetInstanceList)
			AdminGroup.POST("/instances", admin.CreateInstance)
			AdminGroup.PUT("/instances/:id", admin.UpdateInstance)
			AdminGroup.DELETE("/instances/:id", admin.DeleteInstance)           // 管理员可以强制删除
			AdminGroup.POST("/instances/:id/action", admin.AdminInstanceAction) // 管理员实例操作
			AdminGroup.PUT("/instances/:id/reset-password", admin.ResetInstancePassword)
			AdminGroup.GET("/instances/:id/password/:taskId", admin.GetInstanceNewPassword)
			AdminGroup.GET("/instance-type-permissions", admin.GetAdminInstanceTypePermissions)
			AdminGroup.PUT("/instance-type-permissions", admin.UpdateAdminInstanceTypePermissions)
			// 公告管理
			AdminGroup.GET("/announcements", admin.GetAnnouncements)
			AdminGroup.POST("/announcements", admin.CreateAnnouncement)
			AdminGroup.PUT("/announcements/:id", admin.UpdateAnnouncementItem)
			AdminGroup.DELETE("/announcements/:id", admin.DeleteAnnouncement)
			AdminGroup.PUT("/announcements/batch-status", admin.BatchUpdateAnnouncementStatus)
			AdminGroup.POST("/announcements/batch-delete", admin.BatchDeleteAnnouncements)
			// 邀请码管理
			AdminGroup.GET("/invite-codes", admin.GetInviteCodeList)
			AdminGroup.POST("/invite-codes", admin.CreateInviteCode)
			AdminGroup.POST("/invite-codes/generate", admin.GenerateInviteCode)
			AdminGroup.GET("/invite-codes/export", admin.ExportInviteCodes)
			AdminGroup.DELETE("/invite-codes/:id", admin.DeleteInviteCode)
			// 系统监控
			AdminGroup.GET("/monitoring/system", public.GetDashboardStats)
			AdminGroup.GET("/monitoring/audit-logs", system.GetOperationLogs)
			// 流量同步管理
			AdminGroup.POST("/traffic/sync/instance/:instance_id", admin.SyncInstanceTraffic)
			AdminGroup.POST("/traffic/sync/user/:user_id", admin.SyncUserTraffic)
			AdminGroup.POST("/traffic/sync/provider/:provider_id", admin.SyncProviderTraffic)
			AdminGroup.POST("/traffic/sync/all", admin.SyncAllTraffic)
			// 配额管理
			AdminGroup.GET("/quota/users/:userId", system.GetUserQuotaInfo)
			// Provider管理
			AdminGroup.GET("/providers", admin.GetProviderList)
			AdminGroup.POST("/providers", admin.CreateProvider)
			AdminGroup.PUT("/providers/:id", admin.UpdateProvider)
			AdminGroup.DELETE("/providers/:id", admin.DeleteProvider)
			AdminGroup.POST("/providers/freeze", admin.FreezeProvider)
			AdminGroup.POST("/providers/unfreeze", admin.UnfreezeProvider)
			// 证书管理
			AdminGroup.POST("/providers/:id/generate-cert", admin.GenerateProviderCert)
			AdminGroup.POST("/providers/:id/auto-configure-stream", admin.AutoConfigureProviderStream)
			AdminGroup.POST("/providers/:id/health-check", admin.CheckProviderHealth)
			AdminGroup.GET("/providers/:id/status", admin.GetProviderStatus)
			// 配置导出
			AdminGroup.POST("/providers/export-configs", admin.ExportProviderConfigs)
			// 新的配置任务管理
			AdminGroup.POST("/providers/auto-configure", config.AutoConfigureProvider)
			AdminGroup.GET("/configuration-tasks", config.GetConfigurationTasks)
			AdminGroup.GET("/configuration-tasks/:id", config.GetConfigurationTaskDetail)
			AdminGroup.POST("/configuration-tasks/:id/cancel", config.CancelConfigurationTask)
			// 用户任务管理
			AdminGroup.GET("/tasks", admin.GetAdminTasks)
			AdminGroup.POST("/tasks/force-stop", admin.ForceStopTask)
			AdminGroup.GET("/tasks/stats", admin.GetTaskStats)
			AdminGroup.GET("/tasks/overall-stats", admin.GetTaskOverallStats)
			AdminGroup.POST("/tasks/:taskId/cancel", admin.CancelUserTaskByAdmin)
			// 系统镜像管理
			AdminGroup.GET("/system-images", system.GetSystemImageList)
			AdminGroup.POST("/system-images", system.CreateSystemImage)
			AdminGroup.PUT("/system-images/:id", system.UpdateSystemImage)
			AdminGroup.DELETE("/system-images/:id", system.DeleteSystemImage)
			AdminGroup.POST("/system-images/batch-delete", system.BatchDeleteSystemImages)
			AdminGroup.PUT("/system-images/batch-status", system.BatchUpdateSystemImageStatus)
			// 文件上传管理
			AdminGroup.GET("/upload/config", system.GetUploadConfig)
			AdminGroup.PUT("/upload/config", system.UpdateUploadConfig)
			AdminGroup.GET("/upload/stats", system.GetUploadStats)
			AdminGroup.POST("/upload/cleanup", system.CleanupExpiredFiles)

			// 端口映射管理
			AdminGroup.GET("/port-mappings", admin.GetPortMappingList)
			AdminGroup.POST("/port-mappings", admin.CreatePortMapping)
			AdminGroup.PUT("/port-mappings/:id", admin.UpdatePortMapping)
			AdminGroup.DELETE("/port-mappings/:id", admin.DeletePortMapping)
			AdminGroup.POST("/port-mappings/batch-delete", admin.BatchDeletePortMapping)
			AdminGroup.PUT("/providers/:id/port-config", admin.UpdateProviderPortConfig)
			AdminGroup.GET("/providers/:id/port-usage", admin.GetProviderPortUsage)
			AdminGroup.GET("/instances/:id/port-mappings", admin.GetInstancePortMappings)

			// 流量管理API
			adminTrafficAPI := &traffic.AdminTrafficAPI{}
			AdminGroup.GET("/traffic/overview", adminTrafficAPI.GetSystemTrafficOverview)
			AdminGroup.GET("/traffic/provider/:providerId", adminTrafficAPI.GetProviderTrafficStats)
			AdminGroup.GET("/traffic/user/:userId", adminTrafficAPI.GetUserTrafficStats)
			AdminGroup.GET("/traffic/users/rank", adminTrafficAPI.GetAllUsersTrafficRank)
			AdminGroup.POST("/traffic/manage", adminTrafficAPI.ManageTrafficLimits)

		}
		// 基于资源的细粒度权限路由（虚拟化资源）
		ResourceGroup := ApiGroup.Group("/v1/resources")
		ResourceGroup.Use(middleware.RequireAuth(authModel.AuthLevelUser))
		{
			// 虚拟化资源管理，使用基于资源的权限验证
			VirtualizationGroup := ResourceGroup.Group("/virtualization")
			VirtualizationGroup.Use(middleware.RequireResourcePermission("virtualization"))
			{
				VirtualizationGroup.GET("/providers", system.GetProviders)
				VirtualizationGroup.POST("/providers", admin.CreateProvider)
			}
		}
		// Provider API路由
		ProviderGroup := ApiGroup.Group("/v1/providers")
		ProviderGroup.Use(middleware.RequireAuth(authModel.AuthLevelUser))
		{
			providerApi := &provider.ProviderApi{}
			ProviderGroup.GET("/", providerApi.GetProviders)
			ProviderGroup.POST("/connect", providerApi.ConnectProvider)

			// 动态Provider路由 - 支持任意provider名称
			DynamicProviderGroup := ProviderGroup.Group("/:providerName")
			{
				DynamicProviderGroup.GET("/status", providerApi.GetProviderStatus)
				DynamicProviderGroup.GET("/capabilities", providerApi.GetProviderCapabilities)
				DynamicProviderGroup.GET("/instances", providerApi.ListInstances)
				DynamicProviderGroup.POST("/instances", providerApi.CreateInstance)
				DynamicProviderGroup.GET("/instances/:name", providerApi.GetInstance)
				DynamicProviderGroup.POST("/instances/:name/start", providerApi.StartInstance)
				DynamicProviderGroup.POST("/instances/:name/stop", providerApi.StopInstance)
				DynamicProviderGroup.DELETE("/instances/:name", providerApi.DeleteInstance)
				DynamicProviderGroup.GET("/images", providerApi.ListImages)
				DynamicProviderGroup.POST("/images/pull", providerApi.PullImage)
				DynamicProviderGroup.DELETE("/images/:image", providerApi.DeleteImage)
			}
		}
	}
	return Router
}
