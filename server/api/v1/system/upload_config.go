package system

import (
	"oneclickvirt/service/files"
	"time"

	"github.com/gin-gonic/gin"
	"oneclickvirt/config"
	"oneclickvirt/middleware"
	"oneclickvirt/model/common"
)

// GetUploadConfig 获取文件上传配置
// @Summary 获取文件上传配置
// @Description 获取当前的文件上传配置信息
// @Tags 上传管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=config.UploadConfig} "获取成功"
// @Failure 401 {object} common.Response "未授权"
// @Failure 403 {object} common.Response "权限不足"
// @Router /admin/upload/config [get]
func GetUploadConfig(c *gin.Context) {
	authCtx, exists := middleware.GetAuthContext(c)
	if !exists || authCtx.UserType != "admin" {
		common.ResponseWithError(c, common.NewError(common.CodeUnauthorized, "需要管理员权限"))
		return
	}

	// 这里应该从数据库或配置文件读取实际配置
	// 目前返回默认配置
	uploadConfig := config.DefaultUploadConfig

	common.ResponseSuccess(c, uploadConfig, "获取配置成功")
}

// UpdateUploadConfig 更新文件上传配置
// @Summary 更新文件上传配置
// @Description 更新文件上传相关配置
// @Tags 上传管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param config body config.UploadConfig true "上传配置"
// @Success 200 {object} common.Response "更新成功"
// @Failure 400 {object} common.Response "参数错误"
// @Failure 401 {object} common.Response "未授权"
// @Failure 403 {object} common.Response "权限不足"
// @Router /admin/upload/config [put]
func UpdateUploadConfig(c *gin.Context) {
	authCtx, exists := middleware.GetAuthContext(c)
	if !exists || authCtx.UserType != "admin" {
		common.ResponseWithError(c, common.NewError(common.CodeUnauthorized, "需要管理员权限"))
		return
	}

	var uploadConfig config.UploadConfig
	if err := c.ShouldBindJSON(&uploadConfig); err != nil {
		common.ResponseWithError(c, common.NewError(common.CodeValidationError, "参数错误"))
		return
	}

	// 验证配置参数
	if uploadConfig.MaxAvatarSize <= 0 || uploadConfig.MaxAvatarSize > 10*1024*1024 {
		common.ResponseWithError(c, common.NewError(common.CodeValidationError, "头像大小限制无效"))
		return
	}

	if uploadConfig.MaxFileSize <= 0 || uploadConfig.MaxFileSize > 100*1024*1024 {
		common.ResponseWithError(c, common.NewError(common.CodeValidationError, "文件大小限制无效"))
		return
	}

	if uploadConfig.UploadDir == "" {
		common.ResponseWithError(c, common.NewError(common.CodeValidationError, "上传目录不能为空"))
		return
	}

	// TODO: 这里应该保存配置到数据库或配置文件
	// 并重新加载配置到内存中

	common.ResponseSuccess(c, nil, "配置更新成功")
}

// GetUploadStats 获取上传统计信息
// @Summary 获取上传统计信息
// @Description 获取文件上传相关统计数据
// @Tags 上传管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=object} "获取成功"
// @Failure 401 {object} common.Response "未授权"
// @Failure 403 {object} common.Response "权限不足"
// @Router /admin/upload/stats [get]
func GetUploadStats(c *gin.Context) {
	authCtx, exists := middleware.GetAuthContext(c)
	if !exists || authCtx.UserType != "admin" {
		common.ResponseWithError(c, common.NewError(common.CodeUnauthorized, "需要管理员权限"))
		return
	}

	uploadConfig := config.DefaultUploadConfig
	statsService := files.FileStatsService{}

	stats, err := statsService.GetUploadStats(uploadConfig.UploadDir, uploadConfig.RetentionDays)
	if err != nil {
		common.ResponseWithError(c, common.NewError(common.CodeInternalError, "获取统计失败: "+err.Error()))
		return
	}

	common.ResponseSuccess(c, stats, "获取统计成功")
}

// CleanupExpiredFiles 清理过期文件
// @Summary 清理过期文件
// @Description 立即执行过期文件清理（排除头像文件）
// @Tags 上传管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=object} "清理成功"
// @Failure 401 {object} common.Response "未授权"
// @Failure 403 {object} common.Response "权限不足"
// @Failure 500 {object} common.Response "清理失败"
// @Router /admin/upload/cleanup [post]
func CleanupExpiredFiles(c *gin.Context) {
	authCtx, exists := middleware.GetAuthContext(c)
	if !exists || authCtx.UserType != "admin" {
		common.ResponseWithError(c, common.NewError(common.CodeUnauthorized, "需要管理员权限"))
		return
	}

	cleanupService := files.FileCleanupService{}
	statsService := files.FileStatsService{}
	uploadConfig := config.DefaultUploadConfig

	// 获取清理前的统计
	beforeCount, err := statsService.GetCleanableFilesCount(uploadConfig.UploadDir, uploadConfig.RetentionDays)
	if err != nil {
		common.ResponseWithError(c, common.NewError(common.CodeInternalError, "获取清理前统计失败: "+err.Error()))
		return
	}

	// 清理过期文件（排除头像）
	if err := cleanupService.CleanupExpiredFiles(uploadConfig.UploadDir, uploadConfig.RetentionDays); err != nil {
		common.ResponseWithError(c, common.NewError(common.CodeInternalError, "清理过期文件失败: "+err.Error()))
		return
	}

	// 清理空目录
	if err := cleanupService.CleanupEmptyDirectories(uploadConfig.UploadDir); err != nil {
		common.ResponseWithError(c, common.NewError(common.CodeInternalError, "清理空目录失败: "+err.Error()))
		return
	}

	// 获取清理后的统计
	afterCount, err := statsService.GetCleanableFilesCount(uploadConfig.UploadDir, uploadConfig.RetentionDays)
	if err != nil {
		common.ResponseWithError(c, common.NewError(common.CodeInternalError, "获取清理后统计失败: "+err.Error()))
		return
	}

	cleanedFiles := beforeCount - afterCount
	result := gin.H{
		"message":        "文件清理完成（已排除头像文件）",
		"cleaned_files":  cleanedFiles,
		"cleanup_time":   time.Now().Format(time.RFC3339),
		"retention_days": uploadConfig.RetentionDays,
	}

	common.ResponseSuccess(c, result, "文件清理完成")
}
