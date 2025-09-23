package traffic

import (
	"time"

	"oneclickvirt/global"

	"go.uber.org/zap"
)

// SyncTriggerService 流量同步触发服务
type SyncTriggerService struct {
	service      *Service
	limitService *LimitService
}

// NewSyncTriggerService 创建流量同步触发服务
func NewSyncTriggerService() *SyncTriggerService {
	return &SyncTriggerService{
		service:      NewService(),
		limitService: NewLimitService(),
	}
}

// TriggerInstanceTrafficSync 触发单个实例的流量同步
func (s *SyncTriggerService) TriggerInstanceTrafficSync(instanceID uint, reason string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				global.APP_LOG.Error("流量同步过程中发生panic",
					zap.Uint("instanceID", instanceID),
					zap.String("reason", reason),
					zap.Any("panic", r))
			}
		}()

		global.APP_LOG.Info("触发实例流量同步",
			zap.Uint("instanceID", instanceID),
			zap.String("reason", reason))

		// 同步实例流量数据
		if err := s.service.SyncInstanceTraffic(instanceID); err != nil {
			global.APP_LOG.Error("同步实例流量失败",
				zap.Uint("instanceID", instanceID),
				zap.String("reason", reason),
				zap.Error(err))
			return
		}

		global.APP_LOG.Debug("实例流量同步完成",
			zap.Uint("instanceID", instanceID),
			zap.String("reason", reason))
	}()
}

// TriggerUserTrafficSync 触发用户所有实例的流量同步
func (s *SyncTriggerService) TriggerUserTrafficSync(userID uint, reason string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				global.APP_LOG.Error("用户流量同步过程中发生panic",
					zap.Uint("userID", userID),
					zap.String("reason", reason),
					zap.Any("panic", r))
			}
		}()

		global.APP_LOG.Info("触发用户流量同步",
			zap.Uint("userID", userID),
			zap.String("reason", reason))

		// 使用vnStat数据检查流量限制（这会同时触发同步）
		if _, _, err := s.limitService.CheckUserTrafficLimitWithVnStat(userID); err != nil {
			global.APP_LOG.Error("同步用户流量失败",
				zap.Uint("userID", userID),
				zap.String("reason", reason),
				zap.Error(err))
			return
		}

		global.APP_LOG.Debug("用户流量同步完成",
			zap.Uint("userID", userID),
			zap.String("reason", reason))
	}()
}

// TriggerProviderTrafficSync 触发Provider所有实例的流量同步
func (s *SyncTriggerService) TriggerProviderTrafficSync(providerID uint, reason string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				global.APP_LOG.Error("Provider流量同步过程中发生panic",
					zap.Uint("providerID", providerID),
					zap.String("reason", reason),
					zap.Any("panic", r))
			}
		}()

		global.APP_LOG.Info("触发Provider流量同步",
			zap.Uint("providerID", providerID),
			zap.String("reason", reason))

		// 使用vnStat数据检查Provider流量限制（这会同时触发同步）
		if _, _, err := s.limitService.CheckProviderTrafficLimitWithVnStat(providerID); err != nil {
			global.APP_LOG.Error("同步Provider流量失败",
				zap.Uint("providerID", providerID),
				zap.String("reason", reason),
				zap.Error(err))
			return
		}

		global.APP_LOG.Debug("Provider流量同步完成",
			zap.Uint("providerID", providerID),
			zap.String("reason", reason))
	}()
}

// TriggerDelayedInstanceTrafficSync 延迟触发实例流量同步（用于实例启动后等待稳定）
func (s *SyncTriggerService) TriggerDelayedInstanceTrafficSync(instanceID uint, delay time.Duration, reason string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				global.APP_LOG.Error("延迟流量同步过程中发生panic",
					zap.Uint("instanceID", instanceID),
					zap.Duration("delay", delay),
					zap.String("reason", reason),
					zap.Any("panic", r))
			}
		}()

		global.APP_LOG.Info("计划延迟触发实例流量同步",
			zap.Uint("instanceID", instanceID),
			zap.Duration("delay", delay),
			zap.String("reason", reason))

		// 等待指定时间
		time.Sleep(delay)

		// 再次检查实例是否还存在
		var count int64
		if err := global.APP_DB.Model(&struct{}{}).Table("instances").Where("id = ? AND soft_deleted = ?", instanceID, false).Count(&count).Error; err != nil || count == 0 {
			global.APP_LOG.Warn("延迟同步时实例已不存在，跳过",
				zap.Uint("instanceID", instanceID),
				zap.String("reason", reason))
			return
		}

		// 触发流量同步
		s.TriggerInstanceTrafficSync(instanceID, reason)
	}()
}
