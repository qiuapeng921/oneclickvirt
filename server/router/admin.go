package router

import (
	"oneclickvirt/api/v1/admin"
	"oneclickvirt/api/v1/config"
	"oneclickvirt/api/v1/system"
	"oneclickvirt/api/v1/traffic"
	"oneclickvirt/middleware"
	authModel "oneclickvirt/model/auth"

	"github.com/gin-gonic/gin"
)

// InitAdminRouter 管理员路由
func InitAdminRouter(Router *gin.RouterGroup) {
	AdminGroup := Router.Group("/v1/admin")
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

		// 实例管理
		AdminGroup.GET("/instances", admin.GetInstanceList)
		AdminGroup.POST("/instances", admin.CreateInstance)
		AdminGroup.PUT("/instances/:id", admin.UpdateInstance)
		AdminGroup.DELETE("/instances/:id", admin.DeleteInstance)
		AdminGroup.POST("/instances/:id/action", admin.AdminInstanceAction)
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
		AdminGroup.POST("/invite-codes/batch-delete", admin.BatchDeleteInviteCodes)
		AdminGroup.DELETE("/invite-codes/:id", admin.DeleteInviteCode)

		// 系统监控
		AdminGroup.GET("/monitoring/system", admin.GetAdminDashboard)
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
		AdminGroup.POST("/providers/test-ssh-connection", admin.TestSSHConnection)

		// 证书管理
		AdminGroup.POST("/providers/:id/generate-cert", admin.GenerateProviderCert)
		AdminGroup.POST("/providers/:id/auto-configure-stream", admin.AutoConfigureProviderStream)
		AdminGroup.POST("/providers/:id/health-check", admin.CheckProviderHealth)
		AdminGroup.GET("/providers/:id/status", admin.GetProviderStatus)

		// 配置导出
		AdminGroup.POST("/providers/export-configs", admin.ExportProviderConfigs)

		// 配置任务管理
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

		// 端口映射管理
		AdminGroup.GET("/port-mappings", admin.GetPortMappingList)
		AdminGroup.POST("/port-mappings", admin.CreatePortMapping)                   // 仅支持手动添加单个端口（LXD/Incus/PVE）
		AdminGroup.DELETE("/port-mappings/:id", admin.DeletePortMapping)             // 仅支持删除手动添加的端口
		AdminGroup.POST("/port-mappings/batch-delete", admin.BatchDeletePortMapping) // 仅支持删除手动添加的端口
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
}
