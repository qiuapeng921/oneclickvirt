package vnstat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"oneclickvirt/global"
	monitoringModel "oneclickvirt/model/monitoring"
	providerModel "oneclickvirt/model/provider"
	"oneclickvirt/provider"
	providerService "oneclickvirt/service/provider"
	"oneclickvirt/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// getInstanceNetworkInterfaces 获取实例的网络接口列表
func (s *Service) getInstanceNetworkInterfaces(providerInstance provider.Provider, instanceName string) ([]string, error) {
	// 根据Provider类型执行不同的命令获取网络接口
	switch providerInstance.GetType() {
	case "docker":
		return s.getDockerNetworkInterfaces(providerInstance, instanceName)
	case "lxd":
		return s.getLXDNetworkInterfaces(providerInstance, instanceName)
	case "incus":
		return s.getIncusNetworkInterfaces(providerInstance, instanceName)
	case "proxmox":
		return s.getProxmoxNetworkInterfaces(providerInstance, instanceName)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerInstance.GetType())
	}
}

// getDockerNetworkInterfaces 获取Docker容器的网络接口
func (s *Service) getDockerNetworkInterfaces(providerInstance provider.Provider, containerName string) ([]string, error) {
	// 检查Provider类型
	if providerInstance.GetType() != "docker" {
		global.APP_LOG.Error("Provider类型不匹配",
			zap.String("container", containerName),
			zap.String("provider_type", providerInstance.GetType()))
		return nil, fmt.Errorf("provider type mismatch")
	}

	// 获取容器所有的网络接口（包括IPv4和IPv6相关的接口）
	vethInterfaces, err := s.getAllDockerVethInterfaces(providerInstance, containerName)
	if err != nil {
		global.APP_LOG.Error("获取Docker容器所有veth接口失败",
			zap.String("container", containerName),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get all veth interfaces: %w", err)
	}

	if len(vethInterfaces) == 0 {
		global.APP_LOG.Error("未找到Docker容器veth接口",
			zap.String("container", containerName))
		return nil, fmt.Errorf("no veth interfaces found for container: %s", containerName)
	}

	global.APP_LOG.Info("成功获取Docker容器所有veth接口",
		zap.String("container", containerName),
		zap.Strings("veth_interfaces", vethInterfaces))

	// 返回所有veth接口，这些都是宿主机上的接口
	return vethInterfaces, nil
}

// getDockerVethInterface 获取Docker容器对应的veth接口
func (s *Service) getDockerVethInterface(providerInstance provider.Provider, containerName string) (string, error) {
	// 通过容器PID和网络命名空间查找veth接口
	return s.getDockerVethByPID(providerInstance, containerName)
}

// getAllDockerVethInterfaces 获取Docker容器所有相关的veth接口
func (s *Service) getAllDockerVethInterfaces(providerInstance provider.Provider, containerName string) ([]string, error) {
	// 获取容器的所有网络接口
	interfaces := []string{}

	// 1. 获取主要的veth接口（IPv4）
	mainVeth, err := s.getDockerVethByPID(providerInstance, containerName)
	if err == nil && mainVeth != "" {
		interfaces = append(interfaces, mainVeth)
	}

	// 2. 查找与容器相关的所有veth接口（可能包括IPv6接口等）
	additionalVeths, err := s.findAdditionalDockerVeths(providerInstance, containerName)
	if err == nil {
		for _, veth := range additionalVeths {
			// 避免重复添加
			found := false
			for _, existing := range interfaces {
				if existing == veth {
					found = true
					break
				}
			}
			if !found {
				interfaces = append(interfaces, veth)
			}
		}
	}

	global.APP_LOG.Info("获取Docker容器所有veth接口",
		zap.String("container", containerName),
		zap.Strings("interfaces", interfaces))

	return interfaces, nil
}

// findAdditionalDockerVeths 查找容器额外的veth接口
func (s *Service) findAdditionalDockerVeths(providerInstance provider.Provider, containerName string) ([]string, error) {
	// 获取容器的网络信息
	inspectCmd := fmt.Sprintf("docker inspect %s", containerName)
	inspectOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), inspectCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// 解析容器的网络配置，查找所有网络接口
	// 这里可以根据需要扩展，目前先返回空，优先使用主接口
	global.APP_LOG.Debug("容器inspect输出",
		zap.String("container", containerName),
		zap.String("output", inspectOutput))

	return []string{}, nil
}

// getDockerVethByPID 通过容器PID获取veth接口（推荐方法）
func (s *Service) getDockerVethByPID(providerInstance provider.Provider, containerName string) (string, error) {
	// 获取容器的PID
	getPidCmd := fmt.Sprintf("docker inspect -f '{{.State.Pid}}' %s", containerName)

	global.APP_LOG.Debug("获取Docker容器PID",
		zap.String("container", containerName),
		zap.String("command", getPidCmd))

	pidOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), getPidCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get container PID: %w", err)
	}

	pid := strings.TrimSpace(pidOutput)
	if pid == "" || pid == "0" {
		return "", fmt.Errorf("container PID is empty or zero")
	}

	// 进入容器网络命名空间获取eth0的对等接口索引
	getPeerCmd := fmt.Sprintf("nsenter -t %s -n ip link show eth0 | grep -o '@if[0-9]\\+' | cut -c4-", pid)

	global.APP_LOG.Debug("获取容器eth0对等接口索引",
		zap.String("container", containerName),
		zap.String("pid", pid),
		zap.String("command", getPeerCmd))

	peerOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), getPeerCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get peer interface index: %w", err)
	}

	peerIndex := strings.TrimSpace(peerOutput)
	if peerIndex == "" {
		return "", fmt.Errorf("peer interface index is empty")
	}

	// 在宿主机上根据接口索引找到对应的veth接口
	findVethCmd := fmt.Sprintf("grep -l '^%s$' /sys/class/net/*/ifindex | xargs -n1 dirname | xargs -n1 basename", peerIndex)

	global.APP_LOG.Debug("根据接口索引查找veth接口",
		zap.String("container", containerName),
		zap.String("peer_index", peerIndex),
		zap.String("command", findVethCmd))

	vethOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), findVethCmd)
	if err != nil {
		return "", fmt.Errorf("failed to find veth interface by index: %w", err)
	}

	vethInterface := strings.TrimSpace(vethOutput)
	if vethInterface == "" {
		return "", fmt.Errorf("veth interface not found for index: %s", peerIndex)
	}

	return vethInterface, nil
}

// getLXDNetworkInterfaces 获取LXD容器/虚拟机的网络接口
func (s *Service) getLXDNetworkInterfaces(providerInstance provider.Provider, instanceName string) ([]string, error) {
	// 获取LXD实例的所有网络接口信息
	lxdInterfaces, err := s.getAllLXDVethInterfaces(providerInstance, instanceName)
	if err != nil {
		global.APP_LOG.Error("获取LXD所有veth接口失败",
			zap.String("instance", instanceName),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get LXD veth interfaces: %w", err)
	}
	return lxdInterfaces, nil
}

// getLXDVethInterface 获取LXD容器/虚拟机对应的veth接口
func (s *Service) getLXDVethInterface(providerInstance provider.Provider, instanceName string) (string, error) {
	// 首先尝试从 lxc info 获取 Host interface 信息
	infoCmd := fmt.Sprintf("lxc info %s", instanceName)
	infoOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), infoCmd)
	if err != nil {
		global.APP_LOG.Warn("获取LXD实例info失败，使用备用方法",
			zap.String("instance", instanceName),
			zap.Error(err))
	} else {
		// 解析 lxc info 输出，查找 Host interface
		lines := strings.Split(infoOutput, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "Host interface:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					hostInterface := strings.TrimSpace(parts[1])
					if hostInterface != "" {
						global.APP_LOG.Info("从lxc info获取到Host interface",
							zap.String("instance", instanceName),
							zap.String("hostInterface", hostInterface))
						return hostInterface, nil
					}
				}
			}
		}

		global.APP_LOG.Warn("lxc info输出中未找到Host interface信息",
			zap.String("instance", instanceName))
	}

	// 无法从lxc info获取准确的Host interface信息
	return "", fmt.Errorf("无法获取LXD实例 %s 的Host interface信息", instanceName)
}

// getAllLXDVethInterfaces 获取LXD实例所有相关的veth接口
func (s *Service) getAllLXDVethInterfaces(providerInstance provider.Provider, instanceName string) ([]string, error) {
	interfaces := []string{}

	// 1. 获取主要的Host interface
	mainInterface, err := s.getLXDVethInterface(providerInstance, instanceName)
	if err == nil && mainInterface != "" {
		interfaces = append(interfaces, mainInterface)
	}

	// 2. 查找可能的其他网络接口（如IPv6专用接口等）
	additionalInterfaces, err := s.findAdditionalLXDInterfaces(providerInstance, instanceName)
	if err == nil {
		for _, iface := range additionalInterfaces {
			// 避免重复添加
			found := false
			for _, existing := range interfaces {
				if existing == iface {
					found = true
					break
				}
			}
			if !found {
				interfaces = append(interfaces, iface)
			}
		}
	}

	global.APP_LOG.Info("获取LXD实例所有网络接口",
		zap.String("instance", instanceName),
		zap.Strings("interfaces", interfaces))

	if len(interfaces) == 0 {
		return nil, fmt.Errorf("未找到LXD实例 %s 的任何网络接口", instanceName)
	}

	return interfaces, nil
}

// findAdditionalLXDInterfaces 查找LXD实例额外的网络接口
func (s *Service) findAdditionalLXDInterfaces(providerInstance provider.Provider, instanceName string) ([]string, error) {
	// 获取实例的网络状态信息，查找所有Host interfaces
	listCmd := fmt.Sprintf("lxc list %s --format json", instanceName)
	listOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), listCmd)
	if err != nil {
		global.APP_LOG.Warn("获取LXD实例网络状态失败",
			zap.String("instance", instanceName),
			zap.Error(err))
		return []string{}, nil
	}

	global.APP_LOG.Debug("LXD实例网络状态",
		zap.String("instance", instanceName),
		zap.String("output", listOutput))

	// TODO: 解析JSON输出，提取所有网络接口
	// 目前先返回空，后续可以根据需要扩展
	return []string{}, nil
}

// getIncusNetworkInterfaces 获取Incus容器/虚拟机的网络接口
func (s *Service) getIncusNetworkInterfaces(providerInstance provider.Provider, instanceName string) ([]string, error) {
	// 获取Incus实例的所有网络接口信息
	incusInterfaces, err := s.getAllIncusVethInterfaces(providerInstance, instanceName)
	if err != nil {
		global.APP_LOG.Error("获取Incus所有veth接口失败",
			zap.String("instance", instanceName),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get Incus veth interfaces: %w", err)
	}
	return incusInterfaces, nil
}

// getIncusVethInterface 获取Incus容器/虚拟机对应的veth接口
func (s *Service) getIncusVethInterface(providerInstance provider.Provider, instanceName string) (string, error) {
	// 方法1: 从 incus info 获取 Host interface 信息
	infoCmd := fmt.Sprintf("incus info %s", instanceName)
	infoOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), infoCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get Incus instance info: %w", err)
	}

	// 解析 incus info 输出，查找 Host interface
	lines := strings.Split(infoOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Host interface:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				hostInterface := strings.TrimSpace(parts[1])
				if hostInterface != "" {
					global.APP_LOG.Info("从incus info获取到Host interface",
						zap.String("instance", instanceName),
						zap.String("hostInterface", hostInterface))
					return hostInterface, nil
				}
			}
		}
	}

	// 所有方法都失败，返回通用接口名
	global.APP_LOG.Warn("所有方法都无法获取准确的veth接口，使用默认接口",
		zap.String("instance", instanceName))
	return "eth0", fmt.Errorf("无法获取Incus实例 %s 的准确Host interface，使用默认接口 eth0", instanceName)
}

// getAllIncusVethInterfaces 获取Incus实例所有相关的veth接口
func (s *Service) getAllIncusVethInterfaces(providerInstance provider.Provider, instanceName string) ([]string, error) {
	interfaces := []string{}

	// 1. 获取主要的Host interface
	mainInterface, err := s.getIncusVethInterface(providerInstance, instanceName)
	if err == nil && mainInterface != "" {
		interfaces = append(interfaces, mainInterface)
	}

	// 2. 查找可能的其他网络接口（如IPv6专用接口等）
	additionalInterfaces, err := s.findAdditionalIncusInterfaces(providerInstance, instanceName)
	if err == nil {
		for _, iface := range additionalInterfaces {
			// 避免重复添加
			found := false
			for _, existing := range interfaces {
				if existing == iface {
					found = true
					break
				}
			}
			if !found {
				interfaces = append(interfaces, iface)
			}
		}
	}

	global.APP_LOG.Info("获取Incus实例所有网络接口",
		zap.String("instance", instanceName),
		zap.Strings("interfaces", interfaces))

	if len(interfaces) == 0 {
		return nil, fmt.Errorf("未找到Incus实例 %s 的任何网络接口", instanceName)
	}

	return interfaces, nil
}

// findAdditionalIncusInterfaces 查找Incus实例额外的网络接口
func (s *Service) findAdditionalIncusInterfaces(providerInstance provider.Provider, instanceName string) ([]string, error) {
	// 获取实例的网络状态信息，查找所有Host interfaces
	listCmd := fmt.Sprintf("incus list %s --format json", instanceName)
	listOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), listCmd)
	if err != nil {
		global.APP_LOG.Warn("获取Incus实例网络状态失败",
			zap.String("instance", instanceName),
			zap.Error(err))
		return []string{}, nil
	}

	global.APP_LOG.Debug("Incus实例网络状态",
		zap.String("instance", instanceName),
		zap.String("output", listOutput))

	// TODO: 解析JSON输出，提取所有网络接口
	// 目前先返回空，后续可以根据需要扩展
	return []string{}, nil
}

// getProxmoxNetworkInterfaces 获取Proxmox虚拟机/容器的网络接口
func (s *Service) getProxmoxNetworkInterfaces(providerInstance provider.Provider, instanceName string) ([]string, error) {
	// 获取Proxmox实例的所有网络接口
	interfaces, err := s.getAllProxmoxNetworkInterfaces(providerInstance, instanceName)
	if err != nil {
		global.APP_LOG.Error("获取Proxmox所有网络接口失败",
			zap.String("instance", instanceName),
			zap.Error(err))
		// 如果都失败，返回默认接口
		global.APP_LOG.Warn("无法获取Proxmox实例接口，使用默认接口",
			zap.String("instance", instanceName))
		return []string{"vmbr0"}, nil
	}
	return interfaces, nil
}

// getAllProxmoxNetworkInterfaces 获取Proxmox实例所有相关的网络接口
func (s *Service) getAllProxmoxNetworkInterfaces(providerInstance provider.Provider, instanceName string) ([]string, error) {
	interfaces := []string{}

	// 首先通过实例名称查找VMID和类型
	vmid, instanceType, err := s.findVMIDByInstanceName(providerInstance, instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to find VMID for instance %s: %w", instanceName, err)
	}

	// 根据实例类型获取所有网络接口
	switch instanceType {
	case "container":
		containerInterfaces, err := s.getAllProxmoxContainerInterfaces(providerInstance, instanceName, vmid)
		if err == nil {
			interfaces = append(interfaces, containerInterfaces...)
		}
	case "vm":
		vmInterfaces, err := s.getAllProxmoxVMInterfaces(providerInstance, instanceName, vmid)
		if err == nil {
			interfaces = append(interfaces, vmInterfaces...)
		}
	default:
		global.APP_LOG.Warn("未知的Proxmox实例类型",
			zap.String("instance", instanceName),
			zap.String("type", instanceType))
	}

	global.APP_LOG.Info("获取Proxmox实例所有网络接口",
		zap.String("instance", instanceName),
		zap.String("vmid", vmid),
		zap.String("type", instanceType),
		zap.Strings("interfaces", interfaces))

	if len(interfaces) == 0 {
		return nil, fmt.Errorf("未找到Proxmox实例 %s 的任何网络接口", instanceName)
	}

	return interfaces, nil
}

// getAllProxmoxContainerInterfaces 获取Proxmox容器的所有veth接口
func (s *Service) getAllProxmoxContainerInterfaces(providerInstance provider.Provider, instanceName, vmid string) ([]string, error) {
	interfaces := []string{}

	// 获取容器配置，查找所有网络接口
	configCmd := fmt.Sprintf("pct config %s | grep -E '^net[0-9]+:'", vmid)
	configOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), configCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get container config: %w", err)
	}

	if strings.TrimSpace(configOutput) == "" {
		return nil, fmt.Errorf("no network config found for container %s (VMID: %s)", instanceName, vmid)
	}

	// 解析每个网络接口配置
	lines := strings.Split(strings.TrimSpace(configOutput), "\n")
	for _, line := range lines {
		if strings.Contains(line, "net") {
			// 为每个网络接口查找对应的veth接口
			vethInterface, err := s.getProxmoxContainerVethByConfig(providerInstance, vmid, line)
			if err == nil && vethInterface != "" {
				interfaces = append(interfaces, vethInterface)
			}
		}
	}

	// 如果没有找到任何接口，尝试使用传统方法
	if len(interfaces) == 0 {
		vethInterface, err := s.getProxmoxContainerInterface(providerInstance, instanceName)
		if err == nil && vethInterface != "" {
			interfaces = append(interfaces, vethInterface)
		}
	}

	return interfaces, nil
}

// getAllProxmoxVMInterfaces 获取Proxmox虚拟机的所有tap接口
func (s *Service) getAllProxmoxVMInterfaces(providerInstance provider.Provider, instanceName, vmid string) ([]string, error) {
	interfaces := []string{}

	// 获取虚拟机配置，查找所有网络接口
	configCmd := fmt.Sprintf("qm config %s | grep -E '^net[0-9]+:'", vmid)
	configOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), configCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM config: %w", err)
	}

	if strings.TrimSpace(configOutput) == "" {
		return nil, fmt.Errorf("no network config found for VM %s (VMID: %s)", instanceName, vmid)
	}

	// 解析每个网络接口配置
	lines := strings.Split(strings.TrimSpace(configOutput), "\n")
	for _, line := range lines {
		if strings.Contains(line, "net") {
			// 为每个网络接口查找对应的tap接口
			tapInterface, err := s.getProxmoxVMTapByConfig(providerInstance, vmid, line)
			if err == nil && tapInterface != "" {
				interfaces = append(interfaces, tapInterface)
			}
		}
	}

	// 如果没有找到任何接口，尝试使用传统方法
	if len(interfaces) == 0 {
		tapInterface, err := s.getProxmoxVMInterface(providerInstance, instanceName)
		if err == nil && tapInterface != "" {
			interfaces = append(interfaces, tapInterface)
		}
	}

	return interfaces, nil
}

// getProxmoxContainerVethByConfig 根据配置获取容器veth接口
func (s *Service) getProxmoxContainerVethByConfig(providerInstance provider.Provider, vmid, netConfig string) (string, error) {
	// 使用现有的方法获取veth接口
	// 这里可以根据具体的网络配置进行优化
	vethCmd := fmt.Sprintf(`VMID=%s; bridge link | grep veth$VMID | awk '{print $2}' | cut -d'@' -f1 | head -1`, vmid)
	vethOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), vethCmd)
	if err != nil {
		return "", fmt.Errorf("failed to find veth interface for container VMID %s: %w", vmid, err)
	}

	vethInterface := strings.TrimSpace(vethOutput)
	if vethInterface == "" {
		return "", fmt.Errorf("no veth interface found for container VMID %s", vmid)
	}

	return vethInterface, nil
}

// getProxmoxVMTapByConfig 根据配置获取虚拟机tap接口
func (s *Service) getProxmoxVMTapByConfig(providerInstance provider.Provider, vmid, netConfig string) (string, error) {
	// 解析网络配置获取bridge
	bridge := "vmbr1" // 默认bridge
	if strings.Contains(netConfig, "bridge=") {
		parts := strings.Split(netConfig, "bridge=")
		if len(parts) > 1 {
			bridgePart := strings.Fields(parts[1])[0]
			bridgePart = strings.Trim(bridgePart, ",")
			if bridgePart != "" {
				bridge = bridgePart
			}
		}
	}

	// 查找与虚拟机关联的tap接口
	tapCmd := fmt.Sprintf("bridge link | grep %s | grep tap%si | awk '{print $2}' | cut -d'@' -f1", bridge, vmid)
	tapOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), tapCmd)
	if err != nil {
		return "", fmt.Errorf("failed to find tap interface for VM VMID %s: %w", vmid, err)
	}

	tapInterface := strings.TrimSpace(tapOutput)
	if tapInterface == "" {
		return "", fmt.Errorf("no tap interface found for VM VMID %s", vmid)
	}

	return tapInterface, nil
}

// getProxmoxContainerInterface 获取Proxmox容器的veth接口
func (s *Service) getProxmoxContainerInterface(providerInstance provider.Provider, instanceName string) (string, error) {
	// 首先通过实例名称查找VMID
	vmid, instanceType, err := s.findVMIDByInstanceName(providerInstance, instanceName)
	if err != nil {
		return "", fmt.Errorf("failed to find VMID for instance %s: %w", instanceName, err)
	}

	// 确认这是一个容器
	if instanceType != "container" {
		return "", fmt.Errorf("instance %s is not a container (type: %s)", instanceName, instanceType)
	}

	// 使用VMID获取容器配置
	configCmd := fmt.Sprintf("pct config %s | grep net0", vmid)
	configOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), configCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get container config: %w", err)
	}

	if strings.TrimSpace(configOutput) == "" {
		return "", fmt.Errorf("no network config found for container %s (VMID: %s)", instanceName, vmid)
	}

	// 获取容器的veth接口
	// 使用更准确的方法查找容器对应的宿主机veth接口
	ctid := vmid

	// 使用你提供的方法获取准确的veth接口
	vethCmd := fmt.Sprintf(`VMID=%s; CIDX=$(pct exec $VMID -- bash -c "ip -o link | awk -F': ' '/@/ {print \$1\":\"\$2}'" | awk -F'@' '{print $2}' | head -1); bridge link | grep veth$VMID | awk -v cidx=$CIDX '{print $2}' | cut -d'@' -f1`, ctid)
	vethOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), vethCmd)
	if err != nil {
		// 如果精确查找失败，尝试简单方法
		fallbackCmd := fmt.Sprintf("bridge link | grep veth%s | awk '{print $2}' | cut -d'@' -f1 | head -1", ctid)
		vethOutput, err = providerInstance.ExecuteSSHCommand(context.Background(), fallbackCmd)
		if err != nil {
			return "", fmt.Errorf("failed to find veth interface for container %s (VMID: %s): %w", instanceName, ctid, err)
		}
	}

	vethInterface := strings.TrimSpace(vethOutput)
	if vethInterface == "" {
		return "", fmt.Errorf("no veth interface found for container %s (VMID: %s)", instanceName, ctid)
	}

	global.APP_LOG.Info("成功获取容器veth接口",
		zap.String("container", instanceName),
		zap.String("vmid", ctid),
		zap.String("interface", vethInterface))

	return vethInterface, nil
}

// getProxmoxVMInterface 获取Proxmox虚拟机的tap接口
func (s *Service) getProxmoxVMInterface(providerInstance provider.Provider, instanceName string) (string, error) {
	// 首先通过实例名称查找VMID
	vmid, instanceType, err := s.findVMIDByInstanceName(providerInstance, instanceName)
	if err != nil {
		return "", fmt.Errorf("failed to find VMID for instance %s: %w", instanceName, err)
	}

	// 确认这是一个虚拟机
	if instanceType != "vm" {
		return "", fmt.Errorf("instance %s is not a VM (type: %s)", instanceName, instanceType)
	}

	// 使用VMID获取虚拟机配置
	configCmd := fmt.Sprintf("qm config %s | grep net0", vmid)
	configOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), configCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get VM config: %w", err)
	}

	if strings.TrimSpace(configOutput) == "" {
		return "", fmt.Errorf("no network config found for VM %s (VMID: %s)", instanceName, vmid)
	}

	// 解析网络配置获取bridge
	bridge := "vmbr1" // 默认bridge
	if strings.Contains(configOutput, "bridge=") {
		parts := strings.Split(configOutput, "bridge=")
		if len(parts) > 1 {
			bridgePart := strings.Fields(parts[1])[0]
			bridgePart = strings.Trim(bridgePart, ",")
			if bridgePart != "" {
				bridge = bridgePart
			}
		}
	}

	// 查找与虚拟机关联的tap接口（使用已获取的VMID）
	tapCmd := fmt.Sprintf("ip link show | grep 'tap%s' | head -1 | awk '{print $2}' | cut -d':' -f1", vmid)
	tapOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), tapCmd)
	if err != nil {
		// 如果直接查找失败，返回桥接接口
		global.APP_LOG.Warn("无法找到具体的tap接口，使用bridge接口",
			zap.String("vm", instanceName),
			zap.String("vmid", vmid),
			zap.String("bridge", bridge))
		return bridge, nil
	}

	tapInterface := strings.TrimSpace(tapOutput)
	if tapInterface == "" {
		// 返回桥接接口作为备选
		return bridge, nil
	}

	return tapInterface, nil
}

// findVethByBridge 通过bridge查找veth接口
func (s *Service) findVethByBridge(providerInstance provider.Provider, bridge, ctid string) (string, error) {
	// 查找连接到指定bridge的veth接口
	bridgeCmd := fmt.Sprintf("bridge link show | grep %s | grep veth | head -1 | awk '{print $2}' | cut -d':' -f1", bridge)
	bridgeOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), bridgeCmd)
	if err != nil {
		return "", fmt.Errorf("failed to find veth by bridge: %w", err)
	}

	vethInterface := strings.TrimSpace(bridgeOutput)
	if vethInterface == "" {
		return "", fmt.Errorf("no veth interface found for bridge %s", bridge)
	}

	return vethInterface, nil
}

// findVMIDByInstanceName 通过实例名称查找VMID和实例类型
func (s *Service) findVMIDByInstanceName(providerInstance provider.Provider, instanceName string) (string, string, error) {
	// 首先尝试从容器列表中查找
	output, err := providerInstance.ExecuteSSHCommand(context.Background(), "pct list")
	if err == nil {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines[1:] { // 跳过标题行
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				vmid := fields[0]

				// pct list 格式: VMID Status [Lock] [Name]
				// 需要正确解析Name字段
				name := ""
				if len(fields) >= 4 {
					name = fields[3] // 通常Name在第4列
				} else if len(fields) >= 3 && fields[2] != "" {
					name = fields[2] // 有时候Lock为空，Name在第3列
				}

				// 同时匹配VMID和Name
				if vmid == instanceName || name == instanceName {
					global.APP_LOG.Debug("在容器列表中找到匹配项",
						zap.String("instanceName", instanceName),
						zap.String("vmid", vmid),
						zap.String("name", name))
					return vmid, "container", nil
				}
			}
		}
	}

	// 然后尝试从虚拟机列表中查找
	output, err = providerInstance.ExecuteSSHCommand(context.Background(), "qm list")
	if err == nil {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines[1:] { // 跳过标题行
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				vmid := fields[0]
				name := fields[1] // 对于qm list，Name确实在第2列

				// 同时匹配VMID和Name
				if vmid == instanceName || name == instanceName {
					global.APP_LOG.Debug("在虚拟机列表中找到匹配项",
						zap.String("instanceName", instanceName),
						zap.String("vmid", vmid),
						zap.String("name", name))
					return vmid, "vm", nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("instance %s not found in Proxmox", instanceName)
}

// initVnStatForInterface 为指定接口初始化vnStat
func (s *Service) initVnStatForInterface(providerInstance provider.Provider, instanceName, interfaceName string) error {
	// 统一在Provider宿主机上监控网络接口，不在虚拟机/容器内部

	// 确保宿主机有vnstat
	checkCmd := "which vnstat || (apt-get update && apt-get install -y vnstat) || (yum install -y vnstat) || (apk add --no-cache vnstat)"
	if _, err := providerInstance.ExecuteSSHCommand(context.Background(), checkCmd); err != nil {
		global.APP_LOG.Warn("检查或安装vnstat失败",
			zap.String("instance", instanceName),
			zap.String("interface", interfaceName),
			zap.String("command", checkCmd),
			zap.Error(err))
	}

	// 检测vnstat版本以确定正确的初始化参数
	versionOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), "vnstat --version")
	var initCmd string
	var isV2 bool

	if err == nil && strings.Contains(versionOutput, "vnStat 2.") {
		// vnStat 2.x 使用 --add 参数
		isV2 = true
		initCmd = fmt.Sprintf("vnstat -i %s --add", interfaceName)
		global.APP_LOG.Info("检测到vnStat 2.x版本，使用--add参数",
			zap.String("version_output", strings.TrimSpace(versionOutput)))
	} else {
		// vnStat 1.x 使用 --create 参数，或者检测失败时的默认行为
		isV2 = false
		initCmd = fmt.Sprintf("vnstat -i %s --create", interfaceName)
		if err != nil {
			global.APP_LOG.Warn("无法检测vnstat版本，使用默认--create参数",
				zap.Error(err))
		} else {
			global.APP_LOG.Info("检测到vnStat 1.x版本，使用--create参数",
				zap.String("version_output", strings.TrimSpace(versionOutput)))
		}
	}

	global.APP_LOG.Info("开始在宿主机上初始化vnstat监控",
		zap.String("provider", providerInstance.GetType()),
		zap.String("instance", instanceName),
		zap.String("interface", interfaceName),
		zap.String("init_command", initCmd),
		zap.Bool("is_v2", isV2))

	// 执行初始化命令
	output, err := providerInstance.ExecuteSSHCommand(context.Background(), initCmd)
	if err != nil {
		// 如果主要命令失败，尝试备用方案
		var fallbackCmd string
		if isV2 {
			// v2失败，尝试v1的参数
			fallbackCmd = fmt.Sprintf("vnstat -i %s --create", interfaceName)
			global.APP_LOG.Warn("vnStat --add 参数失败，尝试--create参数",
				zap.String("instance", instanceName),
				zap.String("interface", interfaceName),
				zap.Error(err))
		} else {
			// v1失败，尝试v2的参数
			fallbackCmd = fmt.Sprintf("vnstat -i %s --add", interfaceName)
			global.APP_LOG.Warn("vnStat --create 参数失败，尝试--add参数",
				zap.String("instance", instanceName),
				zap.String("interface", interfaceName),
				zap.Error(err))
		}

		output, err = providerInstance.ExecuteSSHCommand(context.Background(), fallbackCmd)
		if err != nil {
			// 如果两种方式都失败，尝试直接运行 vnstat 让它自动初始化
			global.APP_LOG.Warn("所有参数都失败，尝试自动初始化",
				zap.String("instance", instanceName),
				zap.String("interface", interfaceName),
				zap.Error(err))

			autoInitCmd := fmt.Sprintf("vnstat -i %s", interfaceName)
			output, err = providerInstance.ExecuteSSHCommand(context.Background(), autoInitCmd)
			if err != nil {
				global.APP_LOG.Error("vnstat接口初始化失败",
					zap.String("instance", instanceName),
					zap.String("interface", interfaceName),
					zap.String("last_command", autoInitCmd),
					zap.String("output", utils.TruncateString(output, 500)),
					zap.Error(err))
				return fmt.Errorf("vnstat interface initialization failed: %w", err)
			}
			global.APP_LOG.Info("vnstat接口通过自动初始化成功",
				zap.String("instance", instanceName),
				zap.String("interface", interfaceName),
				zap.String("command", autoInitCmd))
		} else {
			global.APP_LOG.Info("vnstat接口通过备用参数初始化成功",
				zap.String("instance", instanceName),
				zap.String("interface", interfaceName),
				zap.String("command", fallbackCmd))
		}
	} else {
		global.APP_LOG.Info("vnstat接口初始化成功",
			zap.String("instance", instanceName),
			zap.String("interface", interfaceName),
			zap.String("command", initCmd))
	}

	return nil
}

// collectInterfaceData 收集单个接口的vnStat数据
func (s *Service) collectInterfaceData(ctx context.Context, iface *monitoringModel.VnStatInterface) error {
	// 获取实例信息
	var instance providerModel.Instance
	if err := global.APP_DB.First(&instance, iface.InstanceID).Error; err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// 获取Provider信息
	var providerInfo providerModel.Provider
	if err := global.APP_DB.First(&providerInfo, iface.ProviderID).Error; err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	// 获取Provider实例
	providerInstance, err := provider.GetProvider(providerInfo.Type)
	if err != nil {
		return fmt.Errorf("failed to get provider instance: %w", err)
	}

	// 检查Provider连接
	if !providerInstance.IsConnected() {
		nodeConfig := provider.NodeConfig{
			Name:        providerInfo.Name,
			Host:        providerService.ExtractHostFromEndpoint(providerInfo.Endpoint),
			Port:        providerInfo.SSHPort,
			Username:    providerInfo.Username,
			Password:    providerInfo.Password,
			Type:        providerInfo.Type,
			NetworkType: providerInfo.NetworkType,
		}

		if err := providerInstance.Connect(ctx, nodeConfig); err != nil {
			return fmt.Errorf("failed to connect to provider: %w", err)
		}
	}

	// 获取vnStat数据
	vnstatData, err := s.getVnStatJSON(providerInstance, instance.Name, iface.Interface)
	if err != nil {
		return fmt.Errorf("failed to get vnstat data: %w", err)
	}

	// 解析并保存数据
	if err := s.parseAndSaveVnStatData(iface, vnstatData); err != nil {
		return fmt.Errorf("failed to parse and save vnstat data: %w", err)
	}

	// 更新接口的最后同步时间
	iface.LastSync = time.Now()
	if err := global.APP_DB.Save(iface).Error; err != nil {
		global.APP_LOG.Error("更新接口同步时间失败", zap.Error(err))
	}

	return nil
}

// getVnStatJSON 获取vnStat的JSON格式数据
func (s *Service) getVnStatJSON(providerInstance provider.Provider, instanceName, interfaceName string) (string, error) {
	// 统一在宿主机上执行vnstat命令，监控veth/桥接接口
	// 限制查询范围以减少数据传输量：最近3个月的月度数据 + 最近30天的日度数据
	// 使用 -d 30 限制只返回最近30天的数据（包含月度统计）
	vnstatCmd := fmt.Sprintf("vnstat -i %s -d 30 --json", interfaceName)

	global.APP_LOG.Debug("获取vnStat数据（宿主机接口）",
		zap.String("provider_type", providerInstance.GetType()),
		zap.String("instance", instanceName),
		zap.String("interface", interfaceName),
		zap.String("command", vnstatCmd))

	// 直接在Provider宿主机上执行SSH命令
	output, err := providerInstance.ExecuteSSHCommand(context.Background(), vnstatCmd)
	if err != nil {
		global.APP_LOG.Error("获取vnStat数据失败",
			zap.String("provider_type", providerInstance.GetType()),
			zap.String("instance", instanceName),
			zap.String("interface", interfaceName),
			zap.String("command", vnstatCmd),
			zap.String("output", utils.TruncateString(output, 500)),
			zap.Error(err))
		return "", fmt.Errorf("failed to get vnstat data: %w", err)
	}

	// 验证JSON格式
	var testData interface{}
	if err := json.Unmarshal([]byte(output), &testData); err != nil {
		global.APP_LOG.Error("vnStat返回的数据不是有效的JSON",
			zap.String("provider_type", providerInstance.GetType()),
			zap.String("instance", instanceName),
			zap.String("interface", interfaceName),
			zap.String("output", utils.TruncateString(output, 1000)),
			zap.Error(err))
		return "", fmt.Errorf("invalid JSON from vnstat: %w", err)
	}

	global.APP_LOG.Debug("成功获取vnStat数据",
		zap.String("provider_type", providerInstance.GetType()),
		zap.String("instance", instanceName),
		zap.String("interface", interfaceName),
		zap.Int("data_length", len(output)))

	return output, nil
}

// compactVnStatJSON 精简vnStat JSON数据，只保留最近的记录以减少存储空间
// 保留策略：最近3个月的月度数据 + 最近30天的日度数据 + 总流量
func (s *Service) compactVnStatJSON(vnstatData *monitoringModel.VnStatResponse, interfaceName string) string {
	// 查找目标接口
	var targetInterface *monitoringModel.VnStatInterfaceData
	for i := range vnstatData.Interfaces {
		if vnstatData.Interfaces[i].Name == interfaceName {
			targetInterface = &vnstatData.Interfaces[i]
			break
		}
	}

	if targetInterface == nil {
		// 如果找不到接口，返回空JSON
		return "{}"
	}

	// 获取标准化数据
	normalized := targetInterface.Traffic.GetNormalizedTrafficData()

	// 只保留最近3个月的月度数据
	recentMonths := normalized.Months
	if len(recentMonths) > 3 {
		recentMonths = recentMonths[len(recentMonths)-3:]
	}

	// 只保留最近30天的日度数据
	recentDays := normalized.Days
	if len(recentDays) > 30 {
		recentDays = recentDays[len(recentDays)-30:]
	}

	// 构建精简的响应结构
	compactedData := monitoringModel.VnStatResponse{
		VnStatVersion: vnstatData.VnStatVersion,
		JsonVersion:   vnstatData.JsonVersion,
		Interfaces: []monitoringModel.VnStatInterfaceData{
			{
				Name:    targetInterface.Name,
				Alias:   targetInterface.Alias,
				Created: targetInterface.Created,
				Updated: targetInterface.Updated,
				Traffic: monitoringModel.VnStatTrafficData{
					Total:  normalized.Total,
					Month:  recentMonths, // v1格式
					Months: recentMonths, // v2格式
					Day:    recentDays,   // v1格式
					Days:   recentDays,   // v2格式
					// 不保留小时数据和Top数据，它们太详细且占用空间
				},
			},
		},
	}

	// 序列化为JSON
	compactedJSON, err := json.Marshal(compactedData)
	if err != nil {
		global.APP_LOG.Error("精简vnStat JSON失败",
			zap.String("interface", interfaceName),
			zap.Error(err))
		// 如果精简失败，返回空JSON而不是原始数据
		return "{}"
	}

	global.APP_LOG.Debug("成功精简vnStat JSON",
		zap.String("interface", interfaceName),
		zap.Int("compacted_size", len(compactedJSON)),
		zap.Int("months_kept", len(recentMonths)),
		zap.Int("days_kept", len(recentDays)))

	return string(compactedJSON)
}

// parseAndSaveVnStatData 解析并保存vnStat数据
func (s *Service) parseAndSaveVnStatData(iface *monitoringModel.VnStatInterface, jsonData string) error {
	var vnstatData monitoringModel.VnStatResponse
	if err := json.Unmarshal([]byte(jsonData), &vnstatData); err != nil {
		return fmt.Errorf("failed to parse vnstat json: %w", err)
	}

	// 根据 jsonversion 进行版本兼容处理
	global.APP_LOG.Debug("解析vnStat数据",
		zap.String("vnstatversion", vnstatData.VnStatVersion),
		zap.String("jsonversion", vnstatData.JsonVersion),
		zap.String("interface", iface.Interface))

	// 查找对应的接口数据
	var interfaceData *monitoringModel.VnStatInterfaceData
	for i := range vnstatData.Interfaces {
		if vnstatData.Interfaces[i].Name == iface.Interface {
			interfaceData = &vnstatData.Interfaces[i]
			break
		}
	}

	if interfaceData == nil {
		return fmt.Errorf("interface %s not found in vnstat data", iface.Interface)
	}

	// 获取标准化的流量数据
	normalizedData := interfaceData.Traffic.GetNormalizedTrafficData()

	// 获取Provider类型
	var providerInfo providerModel.Provider
	if err := global.APP_DB.First(&providerInfo, iface.ProviderID).Error; err != nil {
		global.APP_LOG.Warn("获取Provider信息失败，使用默认类型",
			zap.Uint("provider_id", iface.ProviderID),
			zap.Error(err))
	}
	providerType := providerInfo.Type
	if providerType == "" {
		providerType = "unknown"
	}

	// 精简原始JSON数据，只保留最近的记录以减少存储空间
	compactedJSON := s.compactVnStatJSON(&vnstatData, iface.Interface)

	// 保存总流量数据
	totalRecord := &monitoringModel.VnStatTrafficRecord{
		InstanceID:   iface.InstanceID,
		ProviderID:   iface.ProviderID,
		ProviderType: providerType,
		Interface:    iface.Interface,
		RxBytes:      normalizedData.Total.Rx,
		TxBytes:      normalizedData.Total.Tx,
		TotalBytes:   normalizedData.Total.Rx + normalizedData.Total.Tx,
		Year:         0, // 0表示总计
		Month:        0,
		Day:          0,
		Hour:         0,
		RawData:      compactedJSON,
		RecordTime:   time.Now(),
	}

	// 保存或更新总流量记录
	var existingTotal monitoringModel.VnStatTrafficRecord
	err := global.APP_DB.Where("instance_id = ? AND interface = ? AND year = 0 AND month = 0 AND day = 0 AND hour = 0",
		iface.InstanceID, iface.Interface).First(&existingTotal).Error

	if err == gorm.ErrRecordNotFound {
		if err := global.APP_DB.Create(totalRecord).Error; err != nil {
			return fmt.Errorf("failed to create total traffic record: %w", err)
		}
	} else if err == nil {
		existingTotal.RxBytes = totalRecord.RxBytes
		existingTotal.TxBytes = totalRecord.TxBytes
		existingTotal.TotalBytes = totalRecord.TotalBytes
		existingTotal.RawData = totalRecord.RawData
		existingTotal.RecordTime = totalRecord.RecordTime
		if err := global.APP_DB.Save(&existingTotal).Error; err != nil {
			return fmt.Errorf("failed to update total traffic record: %w", err)
		}
	}

	// 保存月度流量数据
	for _, monthData := range normalizedData.Months {
		monthRecord := &monitoringModel.VnStatTrafficRecord{
			InstanceID:   iface.InstanceID,
			ProviderID:   iface.ProviderID,
			ProviderType: providerType,
			Interface:    iface.Interface,
			RxBytes:      monthData.Rx,
			TxBytes:      monthData.Tx,
			TotalBytes:   monthData.Rx + monthData.Tx,
			Year:         monthData.Date.Year,
			Month:        monthData.Date.Month,
			Day:          0,
			Hour:         0,
			RawData:      "",
			RecordTime:   time.Now(),
		}

		// 检查是否已存在
		var existing monitoringModel.VnStatTrafficRecord
		err := global.APP_DB.Where("instance_id = ? AND interface = ? AND year = ? AND month = ? AND day = 0 AND hour = 0",
			iface.InstanceID, iface.Interface, monthRecord.Year, monthRecord.Month).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			if err := global.APP_DB.Create(monthRecord).Error; err != nil {
				global.APP_LOG.Error("保存月度流量记录失败", zap.Error(err))
			}
		} else if err == nil {
			existing.RxBytes = monthRecord.RxBytes
			existing.TxBytes = monthRecord.TxBytes
			existing.TotalBytes = monthRecord.TotalBytes
			existing.RecordTime = monthRecord.RecordTime
			if err := global.APP_DB.Save(&existing).Error; err != nil {
				global.APP_LOG.Error("更新月度流量记录失败", zap.Error(err))
			}
		}
	}

	// 保存日度流量数据
	for _, dayData := range normalizedData.Days {
		dayRecord := &monitoringModel.VnStatTrafficRecord{
			InstanceID:   iface.InstanceID,
			ProviderID:   iface.ProviderID,
			ProviderType: providerType,
			Interface:    iface.Interface,
			RxBytes:      dayData.Rx,
			TxBytes:      dayData.Tx,
			TotalBytes:   dayData.Rx + dayData.Tx,
			Year:         dayData.Date.Year,
			Month:        dayData.Date.Month,
			Day:          dayData.Date.Day,
			Hour:         0,
			RawData:      "",
			RecordTime:   time.Now(),
		}

		// 检查是否已存在
		var existing monitoringModel.VnStatTrafficRecord
		err := global.APP_DB.Where("instance_id = ? AND interface = ? AND year = ? AND month = ? AND day = ? AND hour = 0",
			iface.InstanceID, iface.Interface, dayRecord.Year, dayRecord.Month, dayRecord.Day).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			if err := global.APP_DB.Create(dayRecord).Error; err != nil {
				global.APP_LOG.Error("保存日度流量记录失败", zap.Error(err))
			}
		} else if err == nil {
			existing.RxBytes = dayRecord.RxBytes
			existing.TxBytes = dayRecord.TxBytes
			existing.TotalBytes = dayRecord.TotalBytes
			existing.RecordTime = dayRecord.RecordTime
			if err := global.APP_DB.Save(&existing).Error; err != nil {
				global.APP_LOG.Error("更新日度流量记录失败", zap.Error(err))
			}
		}
	}

	global.APP_LOG.Debug("vnStat数据保存完成",
		zap.Uint("instance_id", iface.InstanceID),
		zap.String("interface", iface.Interface),
		zap.String("provider_type", providerType),
		zap.String("vnstat_version", vnstatData.VnStatVersion),
		zap.String("json_version", vnstatData.JsonVersion),
		zap.Int("months_count", len(normalizedData.Months)),
		zap.Int("days_count", len(normalizedData.Days)))

	return nil
}

// removeVnStatInterface 从vnstat系统中删除指定的网络接口
func (s *Service) removeVnStatInterface(providerInstance provider.Provider, interfaceName string) error {
	// 检测vnstat版本以确定正确的删除参数
	versionOutput, err := providerInstance.ExecuteSSHCommand(context.Background(), "vnstat --version")
	var removeCmd string
	var isV2 bool

	if err == nil && strings.Contains(versionOutput, "vnStat 2.") {
		// vnStat 2.x 使用 --remove 参数
		isV2 = true
		removeCmd = fmt.Sprintf("vnstat -i %s --remove --force", interfaceName)
		global.APP_LOG.Info("检测到vnStat 2.x版本，使用--remove参数",
			zap.String("interface", interfaceName),
			zap.String("version_output", strings.TrimSpace(versionOutput)))
	} else {
		// vnStat 1.x 使用 --delete 参数，或者检测失败时的默认行为
		isV2 = false
		removeCmd = fmt.Sprintf("vnstat -i %s --delete --force", interfaceName)
		if err != nil {
			global.APP_LOG.Warn("无法检测vnstat版本，使用默认--delete参数",
				zap.String("interface", interfaceName),
				zap.Error(err))
		} else {
			global.APP_LOG.Info("检测到vnStat 1.x版本，使用--delete参数",
				zap.String("interface", interfaceName),
				zap.String("version_output", strings.TrimSpace(versionOutput)))
		}
	}

	global.APP_LOG.Info("开始从vnstat系统中删除接口",
		zap.String("interface", interfaceName),
		zap.String("remove_command", removeCmd),
		zap.Bool("is_v2", isV2))

	// 执行删除命令
	output, err := providerInstance.ExecuteSSHCommand(context.Background(), removeCmd)
	if err != nil {
		// 如果主要命令失败，尝试备用方案
		var fallbackCmd string
		if isV2 {
			// v2失败，尝试v1的参数
			fallbackCmd = fmt.Sprintf("vnstat -i %s --delete --force", interfaceName)
			global.APP_LOG.Warn("vnStat --remove 参数失败，尝试--delete参数",
				zap.String("interface", interfaceName),
				zap.Error(err))
		} else {
			// v1失败，尝试v2的参数
			fallbackCmd = fmt.Sprintf("vnstat -i %s --remove --force", interfaceName)
			global.APP_LOG.Warn("vnStat --delete 参数失败，尝试--remove参数",
				zap.String("interface", interfaceName),
				zap.Error(err))
		}

		output, err = providerInstance.ExecuteSSHCommand(context.Background(), fallbackCmd)
		if err != nil {
			// 检查错误是否因为接口已经不存在
			if strings.Contains(strings.ToLower(output), "no such") ||
				strings.Contains(strings.ToLower(output), "not found") ||
				strings.Contains(strings.ToLower(output), "unknown interface") {
				global.APP_LOG.Info("vnstat接口已经不存在，跳过删除",
					zap.String("interface", interfaceName),
					zap.String("output", strings.TrimSpace(output)))
				return nil
			}

			global.APP_LOG.Error("vnstat接口删除失败",
				zap.String("interface", interfaceName),
				zap.String("last_command", fallbackCmd),
				zap.String("output", strings.TrimSpace(output)),
				zap.Error(err))
			return fmt.Errorf("vnstat interface removal failed: %w", err)
		} else {
			global.APP_LOG.Info("vnstat接口通过备用参数删除成功",
				zap.String("interface", interfaceName),
				zap.String("command", fallbackCmd))
		}
	} else {
		global.APP_LOG.Info("vnstat接口删除成功",
			zap.String("interface", interfaceName),
			zap.String("command", removeCmd),
			zap.String("output", strings.TrimSpace(output)))
	}

	return nil
}
