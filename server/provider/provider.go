package provider

import (
	"context"
	"fmt"
	"sync"

	"oneclickvirt/model/provider"
	"oneclickvirt/provider/health"
)

// 类型别名，使用model包中的结构体
type Instance = provider.ProviderInstance
type Image = provider.ProviderImage
type InstanceConfig = provider.ProviderInstanceConfig
type NodeConfig = provider.ProviderNodeConfig

// ProgressCallback 进度回调函数类型
type ProgressCallback func(percentage int, message string)

// Provider 统一接口
type Provider interface {
	// 基础信息
	GetType() string
	GetName() string
	GetSupportedInstanceTypes() []string // 获取支持的实例类型

	// 实例管理
	ListInstances(ctx context.Context) ([]Instance, error)
	CreateInstance(ctx context.Context, config InstanceConfig) error
	CreateInstanceWithProgress(ctx context.Context, config InstanceConfig, progressCallback ProgressCallback) error
	StartInstance(ctx context.Context, id string) error
	StopInstance(ctx context.Context, id string) error
	RestartInstance(ctx context.Context, id string) error
	DeleteInstance(ctx context.Context, id string) error
	GetInstance(ctx context.Context, id string) (*Instance, error)

	// 镜像管理
	ListImages(ctx context.Context) ([]Image, error)
	PullImage(ctx context.Context, image string) error
	DeleteImage(ctx context.Context, id string) error

	// 连接管理
	Connect(ctx context.Context, config NodeConfig) error
	Disconnect(ctx context.Context) error
	IsConnected() bool

	// 健康检查 - 使用新的health包
	HealthCheck(ctx context.Context) (*health.HealthResult, error)
	GetHealthChecker() health.HealthChecker

	// 密码管理
	SetInstancePassword(ctx context.Context, instanceID, password string) error
	ResetInstancePassword(ctx context.Context, instanceID string) (string, error)

	// SSH命令执行
	ExecuteSSHCommand(ctx context.Context, command string) (string, error)
}

// Registry Provider 注册表
type Registry struct {
	providers map[string]func() Provider
	instances map[string]Provider
	mu        sync.RWMutex
}

var globalRegistry = &Registry{
	providers: make(map[string]func() Provider),
	instances: make(map[string]Provider),
}

// RegisterProvider 注册 Provider
func RegisterProvider(name string, factory func() Provider) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.providers[name] = factory
}

// GetProvider 获取 Provider 实例
func GetProvider(name string) (Provider, error) {
	globalRegistry.mu.RLock()
	if instance, exists := globalRegistry.instances[name]; exists {
		globalRegistry.mu.RUnlock()
		return instance, nil
	}
	globalRegistry.mu.RUnlock()

	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	// 双重检查
	if instance, exists := globalRegistry.instances[name]; exists {
		return instance, nil
	}

	factory, exists := globalRegistry.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not registered", name)
	}

	instance := factory()
	globalRegistry.instances[name] = instance
	return instance, nil
}

// ListProviders 列出所有已注册的 Provider
func ListProviders() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	var names []string
	for name := range globalRegistry.providers {
		names = append(names, name)
	}
	return names
}

// GetAllProviders 获取所有 Provider 实例
func GetAllProviders() map[string]Provider {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	result := make(map[string]Provider)
	for name, instance := range globalRegistry.instances {
		result[name] = instance
	}
	return result
}
