package auth

import (
	"fmt"

	"oneclickvirt/config"
	"oneclickvirt/global"
	configModel "oneclickvirt/model/config"

	"go.uber.org/zap"
)

type ConfigService struct{}

func (s *ConfigService) UpdateConfig(req configModel.UpdateConfigRequest) error {
	global.APP_LOG.Info("开始更新配置")

	// 获取配置管理器
	configManager := config.GetConfigManager()
	if configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}

	// 构建配置更新映射
	configUpdates := make(map[string]interface{})

	// 认证配置
	authConfig := map[string]interface{}{
		"enableEmail":              req.Auth.EnableEmail,
		"enableTelegram":           req.Auth.EnableTelegram,
		"enableQQ":                 req.Auth.EnableQQ,
		"enablePublicRegistration": req.Auth.EnablePublicRegistration,
		"emailSMTPHost":            req.Auth.EmailSMTPHost,
		"emailSMTPPort":            req.Auth.EmailSMTPPort,
		"emailUsername":            req.Auth.EmailUsername,
		"emailPassword":            req.Auth.EmailPassword,
		"telegramBotToken":         req.Auth.TelegramBotToken,
		"qqAppID":                  req.Auth.QQAppID,
		"qqAppKey":                 req.Auth.QQAppKey,
	}
	configUpdates["auth"] = authConfig

	// 配额配置
	quotaConfig := map[string]interface{}{
		"defaultLevel": req.Quota.DefaultLevel,
	}

	// 转换等级限制配置
	if req.Quota.LevelLimits != nil {
		levelLimits := make(map[string]interface{})
		for level, modelLimit := range req.Quota.LevelLimits {
			levelKey := fmt.Sprintf("%d", level)
			levelLimits[levelKey] = map[string]interface{}{
				"maxInstances": modelLimit.MaxInstances,
				"maxResources": modelLimit.MaxResources,
			}
		}
		quotaConfig["levelLimits"] = levelLimits
	}
	configUpdates["quota"] = quotaConfig

	// 邀请码配置
	inviteCodeConfig := map[string]interface{}{
		"enabled":        req.InviteCode.Enabled,
		"required":       req.InviteCode.Required,
		"defaultMaxUses": req.InviteCode.DefaultMaxUses,
	}
	configUpdates["inviteCode"] = inviteCodeConfig

	// 通过配置管理器批量更新配置
	if err := configManager.UpdateConfig(configUpdates); err != nil {
		global.APP_LOG.Error("配置更新失败", zap.Error(err))
		return fmt.Errorf("配置更新失败: %v", err)
	}

	global.APP_LOG.Info("配置更新完成")
	return nil
}

func (s *ConfigService) GetConfig() map[string]interface{} {
	// 获取配置管理器
	configManager := config.GetConfigManager()
	if configManager == nil {
		global.APP_LOG.Error("配置管理器未初始化")
		return map[string]interface{}{}
	}

	// 从配置管理器获取配置
	allConfig := configManager.GetAllConfig()

	// 返回公开的配置部分
	result := make(map[string]interface{})

	if auth, exists := allConfig["auth"]; exists {
		result["auth"] = auth
	}
	if quota, exists := allConfig["quota"]; exists {
		result["quota"] = quota
	}
	if inviteCode, exists := allConfig["inviteCode"]; exists {
		result["inviteCode"] = inviteCode
	}

	return result
}

// SaveInstanceTypePermissions 保存实例类型权限配置
func (s *ConfigService) SaveInstanceTypePermissions(minLevelForContainer, minLevelForVM, minLevelForDelete int) error {
	global.APP_LOG.Info("更新实例类型权限配置",
		zap.Int("minLevelForContainer", minLevelForContainer),
		zap.Int("minLevelForVM", minLevelForVM),
		zap.Int("minLevelForDelete", minLevelForDelete))

	// 获取配置管理器
	configManager := config.GetConfigManager()
	if configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}

	// 构建实例类型权限配置
	instanceTypePermissions := map[string]interface{}{
		"minLevelForContainer": minLevelForContainer,
		"minLevelForVM":        minLevelForVM,
		"minLevelForDelete":    minLevelForDelete,
	}

	// 更新配置
	configUpdates := map[string]interface{}{
		"quota.instanceTypePermissions": instanceTypePermissions,
	}

	if err := configManager.UpdateConfig(configUpdates); err != nil {
		global.APP_LOG.Error("保存实例类型权限配置失败", zap.Error(err))
		return fmt.Errorf("保存实例类型权限配置失败: %v", err)
	}

	global.APP_LOG.Info("实例类型权限配置保存成功")
	return nil
}
