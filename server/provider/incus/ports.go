package incus

import (
	"context"
	"fmt"
	"oneclickvirt/global"
	providerModel "oneclickvirt/model/provider"
	"sort"
	"strings"

	"go.uber.org/zap"
)

// configurePortMappings 配置端口映射
func (i *IncusProvider) configurePortMappings(ctx context.Context, instanceName string, networkConfig NetworkConfig, instanceIP string) error {
	return i.configurePortMappingsWithIP(ctx, instanceName, networkConfig, instanceIP)
}

// configurePortMappingsWithIP 使用指定的实例IP配置端口映射
func (i *IncusProvider) configurePortMappingsWithIP(ctx context.Context, instanceName string, networkConfig NetworkConfig, instanceIP string) error {
	// 检查是否为独立IP模式或纯IPv6模式，如果是则跳过IPv4端口映射
	// dedicated_ipv4: 独立IPv4，不需要端口映射
	// dedicated_ipv4_ipv6: 独立IPv4 + 独立IPv6，不需要端口映射
	// ipv6_only: 纯IPv6，不允许任何IPv4操作
	if networkConfig.NetworkType == "dedicated_ipv4" || networkConfig.NetworkType == "dedicated_ipv4_ipv6" || networkConfig.NetworkType == "ipv6_only" {
		global.APP_LOG.Info("独立IP模式或纯IPv6模式，跳过IPv4端口映射配置",
			zap.String("instance", instanceName),
			zap.String("networkType", networkConfig.NetworkType))
		return nil
	}

	// 从数据库获取实例的端口映射配置
	var instance providerModel.Instance
	if err := global.APP_DB.Where("name = ?", instanceName).First(&instance).Error; err != nil {
		return fmt.Errorf("获取实例信息失败: %w", err)
	}

	// 获取实例的所有端口映射
	var portMappings []providerModel.Port
	if err := global.APP_DB.Where("instance_id = ? AND status = 'active'", instance.ID).Find(&portMappings).Error; err != nil {
		return fmt.Errorf("获取端口映射失败: %w", err)
	}

	if len(portMappings) == 0 {
		global.APP_LOG.Warn("未找到端口映射配置", zap.String("instance", instanceName))
		return nil
	}

	// 分离SSH端口和其他端口
	var sshPort *providerModel.Port
	var otherPorts []providerModel.Port

	for i := range portMappings {
		if portMappings[i].IsSSH {
			sshPort = &portMappings[i]
		} else {
			otherPorts = append(otherPorts, portMappings[i])
		}
	}

	// 1. 单独配置SSH端口映射
	if sshPort != nil {
		var mappingMethod string
		var targetIP string

		// 根据网络类型选择映射方法和目标IP
		if networkConfig.NetworkType == "ipv6_only" {
			// 纯IPv6模式，获取IPv6地址并使用IPv6映射方法
			ipv6, err := i.GetInstanceIPv6(ctx, instanceName)
			if err != nil {
				global.APP_LOG.Warn("获取IPv6地址失败，跳过SSH端口映射",
					zap.String("instance", instanceName),
					zap.Error(err))
			} else {
				mappingMethod = networkConfig.IPv6PortMappingMethod
				targetIP = ipv6
			}
		} else {
			// 其他模式使用IPv4
			mappingMethod = networkConfig.IPv4PortMappingMethod
			targetIP = instanceIP
		}

		if targetIP != "" && mappingMethod != "" {
			if err := i.setupPortMappingWithIP(instanceName, sshPort.HostPort, sshPort.GuestPort, sshPort.Protocol, mappingMethod, targetIP); err != nil {
				global.APP_LOG.Warn("配置SSH端口映射失败",
					zap.String("instance", instanceName),
					zap.Int("hostPort", sshPort.HostPort),
					zap.Int("guestPort", sshPort.GuestPort),
					zap.String("targetIP", targetIP),
					zap.Error(err))
			}
		}
	}

	// 2. 使用区间映射配置其他端口
	if len(otherPorts) > 0 {
		var mappingMethod string
		var targetIP string

		// 根据网络类型选择映射方法和目标IP
		if networkConfig.NetworkType == "ipv6_only" {
			// 纯IPv6模式，获取IPv6地址并使用IPv6映射方法
			ipv6, err := i.GetInstanceIPv6(ctx, instanceName)
			if err != nil {
				global.APP_LOG.Warn("获取IPv6地址失败，跳过其他端口映射",
					zap.String("instance", instanceName),
					zap.Error(err))
			} else {
				mappingMethod = networkConfig.IPv6PortMappingMethod
				targetIP = ipv6
			}
		} else {
			// 其他模式使用IPv4
			mappingMethod = networkConfig.IPv4PortMappingMethod
			targetIP = instanceIP
		}

		if targetIP != "" && mappingMethod != "" {
			if err := i.setupPortRangeMappingWithIP(instanceName, otherPorts, mappingMethod, targetIP); err != nil {
				global.APP_LOG.Warn("配置端口区间映射失败",
					zap.String("instance", instanceName),
					zap.String("targetIP", targetIP),
					zap.Error(err))
			}
		}
	}

	return nil
}

// configureFirewallPorts 配置防火墙端口
func (i *IncusProvider) configureFirewallPorts(instanceName string) error {
	// 获取实例的端口映射信息
	var instance providerModel.Instance
	if err := global.APP_DB.Where("name = ?", instanceName).First(&instance).Error; err != nil {
		return fmt.Errorf("获取实例信息失败: %w", err)
	}

	var portMappings []providerModel.Port
	if err := global.APP_DB.Where("instance_id = ? AND status = 'active'", instance.ID).Find(&portMappings).Error; err != nil {
		return fmt.Errorf("获取端口映射失败: %w", err)
	}

	if len(portMappings) == 0 {
		return nil
	}

	// 检查防火墙类型并配置
	if i.hasFirewalld() {
		return i.configureFirewalldPorts(portMappings)
	} else if i.hasUfw() {
		return i.configureUfwPorts(portMappings)
	}

	return nil
}

// hasFirewalld 检查是否有firewalld
func (i *IncusProvider) hasFirewalld() bool {
	_, err := i.sshClient.Execute("command -v firewall-cmd")
	return err == nil
}

// hasUfw 检查是否有ufw
func (i *IncusProvider) hasUfw() bool {
	_, err := i.sshClient.Execute("command -v ufw")
	return err == nil
}

// configureFirewalldPorts 配置firewalld端口
func (i *IncusProvider) configureFirewalldPorts(portMappings []providerModel.Port) error {
	for _, port := range portMappings {
		cmd := fmt.Sprintf("firewall-cmd --permanent --add-port=%d/%s", port.HostPort, strings.ToLower(port.Protocol))
		_, err := i.sshClient.Execute(cmd)
		if err != nil {
			global.APP_LOG.Warn("配置firewalld端口失败",
				zap.Int("port", port.HostPort),
				zap.String("protocol", port.Protocol),
				zap.Error(err))
		}
	}

	// 重新加载firewall配置
	_, err := i.sshClient.Execute("firewall-cmd --reload")
	return err
}

// configureUfwPorts 配置ufw端口
func (i *IncusProvider) configureUfwPorts(portMappings []providerModel.Port) error {
	for _, port := range portMappings {
		cmd := fmt.Sprintf("ufw allow %d/%s", port.HostPort, strings.ToLower(port.Protocol))
		_, err := i.sshClient.Execute(cmd)
		if err != nil {
			global.APP_LOG.Warn("配置ufw端口失败",
				zap.Int("port", port.HostPort),
				zap.String("protocol", port.Protocol),
				zap.Error(err))
		}
	}

	// 重新加载ufw配置
	_, err := i.sshClient.Execute("ufw reload")
	return err
}

// setupPortMappingWithIP 使用指定的实例IP设置端口映射
func (i *IncusProvider) setupPortMappingWithIP(instanceName string, hostPort, guestPort int, protocol, method, instanceIP string) error {
	global.APP_LOG.Info("设置端口映射(使用已知IP)",
		zap.String("instance", instanceName),
		zap.Int("hostPort", hostPort),
		zap.Int("guestPort", guestPort),
		zap.String("protocol", protocol),
		zap.String("method", method),
		zap.String("instanceIP", instanceIP))

	switch method {
	case "device_proxy":
		return i.setupDeviceProxyMappingWithIP(instanceName, hostPort, guestPort, protocol)
	case "iptables":
		return i.setupIptablesMappingWithIP(instanceName, hostPort, guestPort, protocol, instanceIP)
	case "native":
		// 独立IPv4模式下使用native方法，跳过端口映射
		global.APP_LOG.Info("独立IPv4模式，跳过端口映射",
			zap.String("instance", instanceName),
			zap.Int("hostPort", hostPort),
			zap.Int("guestPort", guestPort),
			zap.String("protocol", protocol))
		return nil
	default:
		// 默认使用device proxy方式
		return i.setupDeviceProxyMappingWithIP(instanceName, hostPort, guestPort, protocol)
	}
}

// setupDeviceProxyMappingWithIP 使用Incus device proxy设置端口映射
func (i *IncusProvider) setupDeviceProxyMappingWithIP(instanceName string, hostPort, guestPort int, protocol string) error {
	deviceName := fmt.Sprintf("proxy-%s-%d", protocol, hostPort)
	hostIP, err := i.getHostIP()
	if err != nil {
		hostIP = "0.0.0.0" // 回退方案
	}

	cmd := fmt.Sprintf("incus config device add %s %s proxy listen=%s:%s:%d connect=%s:0.0.0.0:%d nat=true",
		instanceName, deviceName, strings.ToLower(protocol), hostIP, hostPort, strings.ToLower(protocol), guestPort)

	_, err = i.sshClient.Execute(cmd)
	if err != nil {
		return fmt.Errorf("设置device proxy映射失败: %w", err)
	}

	global.APP_LOG.Info("device proxy端口映射配置成功",
		zap.String("instanceName", instanceName),
		zap.String("deviceName", deviceName),
		zap.Int("hostPort", hostPort),
		zap.Int("guestPort", guestPort))

	return nil
}

// setupIptablesMappingWithIP 使用iptables设置端口映射
func (i *IncusProvider) setupIptablesMappingWithIP(instanceName string, hostPort, guestPort int, protocol, instanceIP string) error {
	// 这里实现iptables端口映射逻辑
	global.APP_LOG.Info("使用iptables设置端口映射",
		zap.String("instanceName", instanceName),
		zap.Int("hostPort", hostPort),
		zap.Int("guestPort", guestPort),
		zap.String("protocol", protocol),
		zap.String("instanceIP", instanceIP))

	// TODO: 实现iptables端口映射
	return fmt.Errorf("iptables端口映射尚未实现")
}

// setupPortRangeMappingWithIP 设置端口范围映射
func (i *IncusProvider) setupPortRangeMappingWithIP(instanceName string, ports []providerModel.Port, method string, instanceIP string) error {
	if len(ports) == 0 {
		return nil
	}

	// 按端口号排序
	sort.Slice(ports, func(i, j int) bool {
		return ports[i].HostPort < ports[j].HostPort
	})

	// 寻找连续的端口范围
	ranges := i.findPortRanges(ports)

	for _, portRange := range ranges {
		if len(portRange) == 1 {
			// 单个端口
			port := portRange[0]
			err := i.setupPortMappingWithIP(instanceName, port.HostPort, port.GuestPort, port.Protocol, method, instanceIP)
			if err != nil {
				global.APP_LOG.Warn("单个端口映射失败",
					zap.Int("port", port.HostPort),
					zap.Error(err))
			}
		} else {
			// 端口范围
			startPort := portRange[0]
			endPort := portRange[len(portRange)-1]
			err := i.setupPortRangeMapping(instanceName, startPort.HostPort, endPort.HostPort, startPort.Protocol)
			if err != nil {
				global.APP_LOG.Warn("端口范围映射失败",
					zap.Int("startPort", startPort.HostPort),
					zap.Int("endPort", endPort.HostPort),
					zap.Error(err))
			}
		}
	}

	return nil
}

// findPortRanges 查找连续的端口范围
func (i *IncusProvider) findPortRanges(ports []providerModel.Port) [][]providerModel.Port {
	if len(ports) == 0 {
		return nil
	}

	var ranges [][]providerModel.Port
	currentRange := []providerModel.Port{ports[0]}

	for i := 1; i < len(ports); i++ {
		// 检查是否是连续端口且协议相同
		if ports[i].HostPort == ports[i-1].HostPort+1 &&
			ports[i].Protocol == ports[i-1].Protocol {
			currentRange = append(currentRange, ports[i])
		} else {
			ranges = append(ranges, currentRange)
			currentRange = []providerModel.Port{ports[i]}
		}
	}
	ranges = append(ranges, currentRange)

	return ranges
}

// setupPortRangeMapping 设置端口范围映射
func (i *IncusProvider) setupPortRangeMapping(instanceName string, startPort, endPort int, protocol string) error {
	deviceName := fmt.Sprintf("proxy-%s-%d-%d", protocol, startPort, endPort)
	hostIP, err := i.getHostIP()
	if err != nil {
		hostIP = "0.0.0.0" // 回退方案
	}

	cmd := fmt.Sprintf("incus config device add %s %s proxy listen=%s:%s:%d-%d connect=%s:0.0.0.0:%d-%d nat=true",
		instanceName, deviceName, strings.ToLower(protocol), hostIP, startPort, endPort, strings.ToLower(protocol), startPort, endPort)

	_, err = i.sshClient.Execute(cmd)
	if err != nil {
		return fmt.Errorf("设置端口范围映射失败: %w", err)
	}

	global.APP_LOG.Info("端口范围映射配置成功",
		zap.String("instanceName", instanceName),
		zap.String("deviceName", deviceName),
		zap.Int("startPort", startPort),
		zap.Int("endPort", endPort))

	return nil
}
