package config

import (
	"fmt"
	"net/http"
	"oneclickvirt/service/auth"
	"oneclickvirt/service/resources"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"oneclickvirt/config"
	"oneclickvirt/global"
	"oneclickvirt/middleware"
	authModel "oneclickvirt/model/auth"
	"oneclickvirt/model/common"
	configModel "oneclickvirt/model/config"
)

// GetUnifiedConfig 获取统一配置接口
// @Summary 获取系统配置
// @Description 根据用户权限返回相应的配置信息
// @Tags 配置管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param scope query string false "配置范围" Enums(public,user,admin) default(user)
// @Success 200 {object} common.Response{data=interface{}} "获取成功"
// @Failure 401 {object} common.Response "认证失败"
// @Failure 403 {object} common.Response "权限不足"
// @Failure 500 {object} common.Response "获取失败"
// @Router /config [get]
func GetUnifiedConfig(c *gin.Context) {
	authCtx, exists := middleware.GetAuthContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, common.Response{
			Code: 401,
			Msg:  "用户未认证",
		})
		return
	}

	scope := c.DefaultQuery("scope", "user")

	// 根据用户权限和请求范围决定返回的配置
	configManager := config.GetConfigManager()
	if configManager == nil {
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 500,
			Msg:  "配置管理器未初始化",
		})
		return
	}

	var result map[string]interface{}

	switch scope {
	case "public":
		// 公开配置，所有用户都可以访问
		result = getPublicConfig(configManager)
	case "user":
		// 用户配置，普通用户可以访问的配置
		result = getUserConfig(configManager, authCtx)
	case "admin", "global":
		// 管理员配置和全局配置，只有管理员可以访问
		permissionService := auth.PermissionService{}
		hasAdminPermission := permissionService.HasPermission(authCtx.UserID, "admin")
		if !hasAdminPermission {
			c.JSON(http.StatusForbidden, common.Response{
				Code: 403,
				Msg:  "权限不足",
			})
			return
		}
		result = getAdminConfig(configManager)
	default:
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 400,
			Msg:  "无效的配置范围",
		})
		return
	}

	common.ResponseSuccess(c, result)
}

// UpdateUnifiedConfig 更新统一配置接口
// @Summary 更新系统配置
// @Description 根据用户权限更新相应的配置信息
// @Tags 配置管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body configModel.UnifiedConfigRequest true "配置更新请求"
// @Success 200 {object} common.Response "更新成功"
// @Failure 400 {object} common.Response "参数错误"
// @Failure 401 {object} common.Response "认证失败"
// @Failure 403 {object} common.Response "权限不足"
// @Failure 500 {object} common.Response "更新失败"
// @Router /config [put]
func UpdateUnifiedConfig(c *gin.Context) {
	authCtx, exists := middleware.GetAuthContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, common.Response{
			Code: 401,
			Msg:  "用户未认证",
		})
		return
	}

	// 解析请求体
	var rawData map[string]interface{}
	if err := c.ShouldBindJSON(&rawData); err != nil {
		common.ResponseWithError(c, common.NewError(common.CodeValidationError, "参数错误"))
		return
	}

	var req configModel.UnifiedConfigRequest

	// 检查是否是新的统一格式
	if scope, exists := rawData["scope"]; exists {
		if config, configExists := rawData["config"]; configExists {
			req.Scope = scope.(string)
			req.Config = config.(map[string]interface{})
		} else {
			common.ResponseWithError(c, common.NewError(common.CodeValidationError, "统一格式缺少config字段"))
			return
		}
	} else {
		// 向后兼容：直接配置数据
		req.Scope = "admin" // 默认管理员范围
		req.Config = rawData
	}

	// 验证权限
	if !hasConfigUpdatePermission(authCtx, req.Scope) {
		c.JSON(http.StatusForbidden, common.Response{
			Code: 403,
			Msg:  "权限不足",
		})
		return
	}

	configManager := config.GetConfigManager()
	if configManager == nil {
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 500,
			Msg:  "配置管理器未初始化",
		})
		return
	}

	// 根据范围过滤配置项
	filteredConfig := filterConfigByScope(req.Config, req.Scope, authCtx)

	// 更新配置
	if err := configManager.UpdateConfig(filteredConfig); err != nil {
		common.ResponseWithError(c, common.NewError(common.CodeConfigError, err.Error()))
		return
	}

	// 同步重要配置到全局配置对象（使用ConfigService确保一致性）
	if err := syncConfigToGlobalViaService(filteredConfig); err != nil {
		global.APP_LOG.Error("同步配置到全局对象失败", zap.Error(err))
		// 不阻止成功响应，因为配置已经保存到数据库
	}

	common.ResponseSuccess(c, nil, "配置更新成功")
}

// getPublicConfig 获取公开配置
func getPublicConfig(cm *config.ConfigManager) map[string]interface{} {
	allConfig := cm.GetAllConfig()
	publicConfig := make(map[string]interface{})

	// 只返回公开的配置项
	publicKeys := []string{
		"app.name",
		"app.version",
		"app.description",
		"auth.enablePublicRegistration",
	}

	for _, key := range publicKeys {
		if value, exists := allConfig[key]; exists {
			publicConfig[key] = value
		}
	}

	return publicConfig
}

// syncConfigToGlobalViaService 通过ConfigService同步配置到全局配置对象
func syncConfigToGlobalViaService(config map[string]interface{}) error {
	// 重新触发配置加载，确保全局配置同步
	configService := auth.ConfigService{}

	// 通过ConfigService的ensureDefaultLevelLimits确保配额配置同步
	// 这是一个安全的操作，会确保默认配置存在
	configService.GetConfig()

	// 处理特定的配额配置更新
	if quotaData, exists := config["quota"]; exists {
		if quotaMap, ok := quotaData.(map[string]interface{}); ok {
			// 更新默认等级
			if defaultLevel, exists := quotaMap["defaultLevel"]; exists {
				if level, ok := defaultLevel.(float64); ok {
					global.APP_CONFIG.Quota.DefaultLevel = int(level)
				} else if level, ok := defaultLevel.(int); ok {
					global.APP_CONFIG.Quota.DefaultLevel = level
				}
			}

			// 对于等级限制，我们通过重新初始化来确保同步
			if _, exists := quotaMap["levelLimits"]; exists {
				// 强制重新加载配置文件以确保同步
				// 这会触发配置的重新读取和同步
				global.APP_LOG.Info("等级限制配置已更新，强制重新同步配置")

				// 使用新的自动同步服务检测变更并同步用户限制
				quotaSyncService := resources.QuotaSyncService{}

				// 创建旧配置映射（从全局配置）
				oldConfig := make(map[string]interface{})
				if global.APP_CONFIG.Quota.LevelLimits != nil {
					oldQuota := map[string]interface{}{
						"levelLimits": convertLevelLimitsToMap(global.APP_CONFIG.Quota.LevelLimits),
					}
					oldConfig["quota"] = oldQuota
				}

				// 检测变更并自动同步
				if err := quotaSyncService.DetectAndSyncLevelChanges(oldConfig, config); err != nil {
					global.APP_LOG.Error("自动同步用户资源限制失败", zap.Error(err))
					// 不阻止配置保存，但记录错误
				}
			}
		}
	}
	return nil
}

// convertLevelLimitsToMap 将LevelLimits转换为map格式以便比较
func convertLevelLimitsToMap(levelLimits map[int]config.LevelLimitInfo) map[string]interface{} {
	result := make(map[string]interface{})

	for level, limitInfo := range levelLimits {
		levelStr := fmt.Sprintf("%d", level)
		limitMap := map[string]interface{}{
			"maxInstances": limitInfo.MaxInstances,
			"maxTraffic":   limitInfo.MaxTraffic,
		}

		if limitInfo.MaxResources != nil {
			limitMap["maxResources"] = limitInfo.MaxResources
		}

		result[levelStr] = limitMap
	}

	return result
}

// getUserConfig 获取用户配置（使用服务端权限验证）
func getUserConfig(cm *config.ConfigManager, authCtx *authModel.AuthContext) map[string]interface{} {
	allConfig := cm.GetAllConfig()
	userConfig := make(map[string]interface{})
	permissionService := auth.PermissionService{}

	// 用户可以访问的配置项
	allowedKeys := []string{
		"app.name",
		"app.version",
		"auth.enablePublicRegistration",
		"quota.defaultLevel",
		"quota.levelLimits",
	}

	// 使用权限服务验证，而不是依赖JWT中的userType
	hasAdminPermission := permissionService.HasPermission(authCtx.UserID, "admin")
	if hasAdminPermission {
		allowedKeys = append(allowedKeys, []string{
			"auth.enableEmail",
			"auth.enableTelegram",
			"auth.enableQQ",
		}...)
	}

	for _, key := range allowedKeys {
		if value, exists := allConfig[key]; exists {
			userConfig[key] = value
		}
	}

	return userConfig
}

// getAdminConfig 获取管理员配置
func getAdminConfig(cm *config.ConfigManager) map[string]interface{} {
	// 管理员可以访问所有配置
	return cm.GetAllConfig()
}

// hasConfigUpdatePermission 检查配置更新权限（使用服务端权限验证）
func hasConfigUpdatePermission(authCtx *authModel.AuthContext, scope string) bool {
	// 使用权限服务进行服务端权限验证
	permissionService := auth.PermissionService{}

	switch scope {
	case "public":
		// 公开配置不允许更新
		return false
	case "user":
		// 普通用户配置，管理员可以更新
		// 使用权限服务验证，而不是依赖客户端传入的userType
		hasAdminPermission := permissionService.HasPermission(authCtx.UserID, "admin")
		return hasAdminPermission
	case "admin", "global":
		// 管理员配置和全局配置，只有管理员可以更新
		hasAdminPermission := permissionService.HasPermission(authCtx.UserID, "admin")
		return hasAdminPermission
	default:
		return false
	}
}

// filterConfigByScope 根据范围过滤配置（使用服务端权限验证）
func filterConfigByScope(config map[string]interface{}, scope string, authCtx *authModel.AuthContext) map[string]interface{} {
	filtered := make(map[string]interface{})
	permissionService := auth.PermissionService{}

	switch scope {
	case "user":
		// 只允许更新用户级别的配置
		allowedKeys := map[string]bool{
			"quota.defaultLevel": true,
			"quota.levelLimits":  true,
		}

		// 使用权限服务验证，而不是依赖JWT中的userType
		hasAdminPermission := permissionService.HasPermission(authCtx.UserID, "admin")
		if hasAdminPermission {
			allowedKeys["auth.enablePublicRegistration"] = true
		}

		for key, value := range config {
			if allowedKeys[key] {
				filtered[key] = value
			}
		}
	case "admin":
		// 管理员可以更新所有配置
		hasAdminPermission := permissionService.HasPermission(authCtx.UserID, "admin")
		if hasAdminPermission {
			filtered = config
		}
	case "global":
		// 全局配置，只有管理员可以更新
		hasAdminPermission := permissionService.HasPermission(authCtx.UserID, "admin")
		if hasAdminPermission {
			filtered = config
		}
	}

	return filtered
}
