package incus

import (
	"context"
	"fmt"
	"oneclickvirt/global"
	"oneclickvirt/model/provider"
	"oneclickvirt/provider/portmapping"
	"strconv"

	"go.uber.org/zap"
)

// IncusPortMapping Incus端口映射实现
type IncusPortMapping struct {
	*portmapping.BaseProvider
}

// NewIncusPortMapping 创建Incus端口映射Provider
func NewIncusPortMapping(config *portmapping.ManagerConfig) portmapping.PortMappingProvider {
	return &IncusPortMapping{
		BaseProvider: portmapping.NewBaseProvider("incus", config),
	}
}

// SupportsDynamicMapping Incus支持动态端口映射
func (i *IncusPortMapping) SupportsDynamicMapping() bool {
	return true
}

// CreatePortMapping 创建Incus端口映射
func (i *IncusPortMapping) CreatePortMapping(ctx context.Context, req *portmapping.PortMappingRequest) (*portmapping.PortMappingResult, error) {
	global.APP_LOG.Info("Creating Incus port mapping",
		zap.String("instanceId", req.InstanceID),
		zap.Int("hostPort", req.HostPort),
		zap.Int("guestPort", req.GuestPort),
		zap.String("protocol", req.Protocol))

	// 验证请求参数
	if err := i.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %v", err)
	}

	// 获取实例信息
	instance, err := i.getInstance(req.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %v", err)
	}

	// 获取Provider信息
	providerInfo, err := i.getProvider(req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %v", err)
	}

	// 分配端口
	hostPort := req.HostPort
	if hostPort == 0 {
		hostPort, err = i.BaseProvider.AllocatePort(ctx, req.ProviderID, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate port: %v", err)
		}
	}

	// 使用Incus原生端口映射方法 (proxy device)
	if err := i.createIncusProxyDevice(ctx, instance, hostPort, req.GuestPort, req.Protocol, providerInfo); err != nil {
		return nil, fmt.Errorf("failed to create Incus proxy device: %v", err)
	}

	// 保存到数据库
	result := &portmapping.PortMappingResult{
		InstanceID:    req.InstanceID,
		ProviderID:    req.ProviderID,
		Protocol:      req.Protocol,
		HostPort:      hostPort,
		GuestPort:     req.GuestPort,
		HostIP:        providerInfo.Endpoint,
		PublicIP:      i.getPublicIP(providerInfo),
		IPv6Address:   req.IPv6Address,
		Status:        "active",
		Description:   req.Description,
		MappingMethod: i.determineMappingMethod(req, providerInfo),
		IsSSH:         req.GuestPort == 22,
		IsAutomatic:   req.HostPort == 0,
	}

	// 转换为数据库模型并保存
	portModel := i.BaseProvider.ToDBModel(result)
	if err := global.APP_DB.Create(portModel).Error; err != nil {
		global.APP_LOG.Error("Failed to save port mapping to database", zap.Error(err))
		// 尝试清理已创建的Incus proxy device
		i.cleanupIncusProxyDevice(ctx, instance, hostPort, req.GuestPort, req.Protocol)
		return nil, fmt.Errorf("failed to save port mapping: %v", err)
	}

	result.ID = portModel.ID
	result.CreatedAt = portModel.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	result.UpdatedAt = portModel.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")

	global.APP_LOG.Info("Incus port mapping created successfully",
		zap.Uint("id", result.ID),
		zap.Int("hostPort", hostPort),
		zap.Int("guestPort", req.GuestPort))

	return result, nil
}

// DeletePortMapping 删除Incus端口映射
func (i *IncusPortMapping) DeletePortMapping(ctx context.Context, req *portmapping.DeletePortMappingRequest) error {
	global.APP_LOG.Info("Deleting Incus port mapping",
		zap.Uint("id", req.ID),
		zap.String("instanceId", req.InstanceID))

	// 获取端口映射信息
	var portModel provider.Port
	if err := global.APP_DB.First(&portModel, req.ID).Error; err != nil {
		return fmt.Errorf("port mapping not found: %v", err)
	}

	// 获取实例信息
	instance, err := i.getInstance(req.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %v", err)
	}

	// 删除Incus proxy device
	if err := i.removeIncusProxyDevice(ctx, instance, portModel.HostPort, portModel.GuestPort, portModel.Protocol); err != nil {
		if !req.ForceDelete {
			return fmt.Errorf("failed to remove Incus proxy device: %v", err)
		}
		global.APP_LOG.Warn("Failed to remove Incus proxy device, but force delete is enabled", zap.Error(err))
	}

	// 从数据库删除
	if err := global.APP_DB.Delete(&portModel).Error; err != nil {
		return fmt.Errorf("failed to delete port mapping from database: %v", err)
	}

	global.APP_LOG.Info("Incus port mapping deleted successfully", zap.Uint("id", req.ID))
	return nil
}

// UpdatePortMapping 更新Incus端口映射
func (i *IncusPortMapping) UpdatePortMapping(ctx context.Context, req *portmapping.UpdatePortMappingRequest) (*portmapping.PortMappingResult, error) {
	global.APP_LOG.Info("Updating Incus port mapping", zap.Uint("id", req.ID))

	// 获取现有端口映射
	var portModel provider.Port
	if err := global.APP_DB.First(&portModel, req.ID).Error; err != nil {
		return nil, fmt.Errorf("port mapping not found: %v", err)
	}

	// 获取实例信息
	instance, err := i.getInstance(req.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %v", err)
	}

	// 获取Provider信息
	providerInfo, err := i.getProvider(portModel.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %v", err)
	}

	// 如果端口发生变化，需要重新创建proxy device
	if req.HostPort != portModel.HostPort || req.GuestPort != portModel.GuestPort || req.Protocol != portModel.Protocol {
		// 删除旧的proxy device
		if err := i.removeIncusProxyDevice(ctx, instance, portModel.HostPort, portModel.GuestPort, portModel.Protocol); err != nil {
			global.APP_LOG.Warn("Failed to remove old Incus proxy device", zap.Error(err))
		}

		// 创建新的proxy device
		if err := i.createIncusProxyDevice(ctx, instance, req.HostPort, req.GuestPort, req.Protocol, providerInfo); err != nil {
			return nil, fmt.Errorf("failed to create new Incus proxy device: %v", err)
		}
	}

	// 更新数据库记录
	updates := map[string]interface{}{
		"host_port":   req.HostPort,
		"guest_port":  req.GuestPort,
		"protocol":    req.Protocol,
		"description": req.Description,
		"status":      req.Status,
	}

	if err := global.APP_DB.Model(&portModel).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update port mapping: %v", err)
	}

	// 重新获取更新后的记录
	if err := global.APP_DB.First(&portModel, req.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to get updated port mapping: %v", err)
	}

	result := i.BaseProvider.FromDBModel(&portModel)
	result.HostIP = providerInfo.Endpoint
	result.PublicIP = i.getPublicIP(providerInfo)
	result.MappingMethod = "incus-proxy"

	global.APP_LOG.Info("Incus port mapping updated successfully", zap.Uint("id", req.ID))
	return result, nil
}

// ListPortMappings 列出Incus端口映射
func (i *IncusPortMapping) ListPortMappings(ctx context.Context, instanceID string) ([]*portmapping.PortMappingResult, error) {
	var ports []provider.Port
	if err := global.APP_DB.Where("instance_id = ?", instanceID).Find(&ports).Error; err != nil {
		return nil, fmt.Errorf("failed to list port mappings: %v", err)
	}

	var results []*portmapping.PortMappingResult
	for _, port := range ports {
		result := i.BaseProvider.FromDBModel(&port)
		result.MappingMethod = "incus-proxy"

		// 获取Provider信息以填充IP地址
		if providerInfo, err := i.getProvider(port.ProviderID); err == nil {
			result.HostIP = providerInfo.Endpoint
			result.PublicIP = i.getPublicIP(providerInfo)
		}

		results = append(results, result)
	}

	return results, nil
}

// validateRequest 验证请求参数
func (i *IncusPortMapping) validateRequest(req *portmapping.PortMappingRequest) error {
	if req.InstanceID == "" {
		return fmt.Errorf("instance ID is required")
	}
	if req.GuestPort <= 0 || req.GuestPort > 65535 {
		return fmt.Errorf("invalid guest port: %d", req.GuestPort)
	}
	if req.HostPort < 0 || req.HostPort > 65535 {
		return fmt.Errorf("invalid host port: %d", req.HostPort)
	}
	if req.Protocol == "" {
		req.Protocol = "tcp"
	}
	return portmapping.ValidateProtocol(req.Protocol)
}

// getInstance 获取实例信息
func (i *IncusPortMapping) getInstance(instanceID string) (*provider.Instance, error) {
	var instance provider.Instance
	id, err := strconv.ParseUint(instanceID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid instance ID: %s", instanceID)
	}

	if err := global.APP_DB.First(&instance, uint(id)).Error; err != nil {
		return nil, fmt.Errorf("instance not found: %v", err)
	}

	return &instance, nil
}

// getProvider 获取Provider信息
func (i *IncusPortMapping) getProvider(providerID uint) (*provider.Provider, error) {
	var providerInfo provider.Provider
	if err := global.APP_DB.First(&providerInfo, providerID).Error; err != nil {
		return nil, fmt.Errorf("provider not found: %v", err)
	}
	return &providerInfo, nil
}

// getPublicIP 获取公网IP
func (i *IncusPortMapping) getPublicIP(providerInfo *provider.Provider) string {
	// 对于Incus，使用Provider的endpoint作为公网IP
	return providerInfo.Endpoint
}

// createIncusProxyDevice 创建Incus proxy device
func (i *IncusPortMapping) createIncusProxyDevice(ctx context.Context, instance *provider.Instance, hostPort, guestPort int, protocol string, providerInfo *provider.Provider) error {
	// 这里应该调用Incus API来创建proxy device
	global.APP_LOG.Info("Creating Incus proxy device",
		zap.String("instance", instance.Name),
		zap.Int("hostPort", hostPort),
		zap.Int("guestPort", guestPort),
		zap.String("protocol", protocol))

	// TODO: 实现实际的Incus API调用
	// 例如：incus config device add <instance> proxy<hostPort> proxy listen=tcp:0.0.0.0:<hostPort> connect=tcp:127.0.0.1:<guestPort>

	return nil
}

// removeIncusProxyDevice 删除Incus proxy device
func (i *IncusPortMapping) removeIncusProxyDevice(ctx context.Context, instance *provider.Instance, hostPort, guestPort int, protocol string) error {
	// 这里应该调用Incus API来删除proxy device
	global.APP_LOG.Info("Removing Incus proxy device",
		zap.String("instance", instance.Name),
		zap.Int("hostPort", hostPort),
		zap.Int("guestPort", guestPort),
		zap.String("protocol", protocol))

	// TODO: 实现实际的Incus API调用
	// 例如：incus config device remove <instance> proxy<hostPort>

	return nil
}

// cleanupIncusProxyDevice 清理Incus proxy device（在出错时调用）
func (i *IncusPortMapping) cleanupIncusProxyDevice(ctx context.Context, instance *provider.Instance, hostPort, guestPort int, protocol string) {
	if err := i.removeIncusProxyDevice(ctx, instance, hostPort, guestPort, protocol); err != nil {
		global.APP_LOG.Error("Failed to cleanup Incus proxy device", zap.Error(err))
	}
}

// determineMappingMethod 确定端口映射方法
func (i *IncusPortMapping) determineMappingMethod(req *portmapping.PortMappingRequest, providerInfo *provider.Provider) string {
	// 如果请求中指定了映射方法，使用指定的方法
	if req.MappingMethod != "" {
		return req.MappingMethod
	}

	// 如果启用了IPv6，根据Provider配置确定方法
	if req.IPv6Enabled {
		switch providerInfo.IPv6PortMappingMethod {
		case "iptables":
			return "incus-iptables-ipv6"
		case "device_proxy":
			return "incus-device-proxy-ipv6"
		default:
			return "incus-device-proxy-ipv6"
		}
	}

	// IPv4映射
	switch providerInfo.IPv4PortMappingMethod {
	case "iptables":
		return "incus-iptables"
	case "device_proxy":
		return "incus-device-proxy"
	default:
		return "incus-device-proxy"
	}
}

// init 注册Incus端口映射Provider
func init() {
	portmapping.RegisterProvider("incus", func(config *portmapping.ManagerConfig) portmapping.PortMappingProvider {
		return NewIncusPortMapping(config)
	})
}
