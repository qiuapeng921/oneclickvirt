package health

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ProviderType 提供商类型
type ProviderType string

const (
	ProviderTypeDocker  ProviderType = "docker"
	ProviderTypeLXD     ProviderType = "lxd"
	ProviderTypeIncus   ProviderType = "incus"
	ProviderTypeProxmox ProviderType = "proxmox"
)

// HealthManager 健康检查管理器
type HealthManager struct {
	checkers map[string]HealthChecker
	logger   *zap.Logger
}

// NewHealthManager 创建健康检查管理器
func NewHealthManager(logger *zap.Logger) *HealthManager {
	return &HealthManager{
		checkers: make(map[string]HealthChecker),
		logger:   logger,
	}
}

// RegisterChecker 注册健康检查器
func (hm *HealthManager) RegisterChecker(id string, checker HealthChecker) {
	hm.checkers[id] = checker
}

// CreateChecker 创建指定类型的健康检查器
func (hm *HealthManager) CreateChecker(providerType ProviderType, config HealthConfig) (HealthChecker, error) {
	// 设置默认值
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.APIScheme == "" {
		config.APIScheme = "https"
	}

	switch providerType {
	case ProviderTypeDocker:
		if config.APIScheme == "" {
			config.APIScheme = "http"
		}
		if config.APIPort == 0 {
			config.APIPort = 2375
		}
		return NewDockerHealthChecker(config, hm.logger), nil

	case ProviderTypeLXD:
		if config.APIPort == 0 {
			config.APIPort = 8443
		}
		return NewLXDHealthChecker(config, hm.logger), nil

	case ProviderTypeIncus:
		if config.APIPort == 0 {
			config.APIPort = 8443
		}
		return NewIncusHealthChecker(config, hm.logger), nil

	case ProviderTypeProxmox:
		if config.APIPort == 0 {
			config.APIPort = 8006
		}
		return NewProxmoxHealthChecker(config, hm.logger), nil

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// CheckHealth 执行健康检查
func (hm *HealthManager) CheckHealth(ctx context.Context, id string) (*HealthResult, error) {
	checker, exists := hm.checkers[id]
	if !exists {
		return nil, fmt.Errorf("health checker not found for ID: %s", id)
	}

	return checker.CheckHealth(ctx)
}

// CheckAllHealth 检查所有注册的健康检查器
func (hm *HealthManager) CheckAllHealth(ctx context.Context) (map[string]*HealthResult, error) {
	results := make(map[string]*HealthResult)

	for id, checker := range hm.checkers {
		result, err := checker.CheckHealth(ctx)
		if err != nil {
			hm.logger.Error("Health check failed",
				zap.String("checker_id", id),
				zap.Error(err))
			// 创建错误结果
			result = &HealthResult{
				Status:    HealthStatusUnhealthy,
				Timestamp: time.Now(),
				Errors:    []string{err.Error()},
			}
		} else {
			hm.logger.Debug("Health check succeeded",
				zap.String("checker_id", id),
				zap.String("status", string(result.Status)))
		}
		results[id] = result
	}

	return results, nil
}

// RemoveChecker 移除健康检查器
func (hm *HealthManager) RemoveChecker(id string) {
	if checker, exists := hm.checkers[id]; exists {
		// 如果检查器实现了Close方法，则调用它
		if closer, ok := checker.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				hm.logger.Warn("Failed to close health checker",
					zap.String("checker_id", id),
					zap.Error(err))
			}
		}
		delete(hm.checkers, id)
	}
}

// GetChecker 获取健康检查器
func (hm *HealthManager) GetChecker(id string) (HealthChecker, bool) {
	checker, exists := hm.checkers[id]
	return checker, exists
}

// ListCheckers 列出所有注册的检查器ID
func (hm *HealthManager) ListCheckers() []string {
	ids := make([]string, 0, len(hm.checkers))
	for id := range hm.checkers {
		ids = append(ids, id)
	}
	return ids
}

// Close 关闭所有健康检查器
func (hm *HealthManager) Close() error {
	for id, checker := range hm.checkers {
		if closer, ok := checker.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				hm.logger.Warn("Failed to close health checker",
					zap.String("checker_id", id),
					zap.Error(err))
			}
		}
	}
	hm.checkers = make(map[string]HealthChecker)
	return nil
}
