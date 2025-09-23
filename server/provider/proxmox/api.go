package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"oneclickvirt/global"
	"oneclickvirt/provider"

	"go.uber.org/zap"
)

func (p *ProxmoxProvider) apiListInstances(ctx context.Context) ([]provider.Instance, error) {
	var instances []provider.Instance

	// 获取虚拟机列表
	vmURL := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/qemu", p.config.Host, p.node)
	vmReq, err := http.NewRequestWithContext(ctx, "GET", vmURL, nil)
	if err != nil {
		return nil, err
	}

	// 设置认证头
	p.setAPIAuth(vmReq)

	vmResp, err := p.apiClient.Do(vmReq)
	if err != nil {
		global.APP_LOG.Warn("获取虚拟机列表失败", zap.Error(err))
	} else {
		defer vmResp.Body.Close()

		var vmResponse map[string]interface{}
		if err := json.NewDecoder(vmResp.Body).Decode(&vmResponse); err == nil {
			if data, ok := vmResponse["data"].([]interface{}); ok {
				for _, item := range data {
					if vmData, ok := item.(map[string]interface{}); ok {
						status := "stopped"
						if vmData["status"].(string) == "running" {
							status = "running"
						}

						instance := provider.Instance{
							ID:     fmt.Sprintf("%v", vmData["vmid"]),
							Name:   vmData["name"].(string),
							Status: status,
							Type:   "vm",
							CPU:    fmt.Sprintf("%v", vmData["cpus"]),
							Memory: fmt.Sprintf("%.0f MB", vmData["mem"].(float64)/1024/1024),
						}

						// 获取VM的IP地址
						if ipAddress, err := p.getInstanceIPAddress(ctx, instance.ID, "vm"); err == nil && ipAddress != "" {
							instance.IP = ipAddress
							instance.PrivateIP = ipAddress
						}
						instances = append(instances, instance)
					}
				}
			}
		}
	}

	// 获取容器列表
	ctURL := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/lxc", p.config.Host, p.node)
	ctReq, err := http.NewRequestWithContext(ctx, "GET", ctURL, nil)
	if err != nil {
		global.APP_LOG.Warn("创建容器请求失败", zap.Error(err))
	} else {
		// 设置认证头
		p.setAPIAuth(ctReq)

		ctResp, err := p.apiClient.Do(ctReq)
		if err != nil {
			global.APP_LOG.Warn("获取容器列表失败", zap.Error(err))
		} else {
			defer ctResp.Body.Close()

			var ctResponse map[string]interface{}
			if err := json.NewDecoder(ctResp.Body).Decode(&ctResponse); err == nil {
				if data, ok := ctResponse["data"].([]interface{}); ok {
					for _, item := range data {
						if ctData, ok := item.(map[string]interface{}); ok {
							status := "stopped"
							if ctData["status"].(string) == "running" {
								status = "running"
							}

							instance := provider.Instance{
								ID:     fmt.Sprintf("%v", ctData["vmid"]),
								Name:   ctData["name"].(string),
								Status: status,
								Type:   "container",
								CPU:    fmt.Sprintf("%v", ctData["cpus"]),
								Memory: fmt.Sprintf("%.0f MB", ctData["mem"].(float64)/1024/1024),
							}

							// 获取容器的IP地址
							if ipAddress, err := p.getInstanceIPAddress(ctx, instance.ID, "container"); err == nil && ipAddress != "" {
								instance.IP = ipAddress
								instance.PrivateIP = ipAddress
							}
							instances = append(instances, instance)
						}
					}
				}
			}
		}
	}

	global.APP_LOG.Info("通过API成功获取Proxmox实例列表",
		zap.Int("totalCount", len(instances)))
	return instances, nil
}

func (p *ProxmoxProvider) apiCreateInstance(ctx context.Context, config provider.InstanceConfig) error {
	return p.apiCreateInstanceWithProgress(ctx, config, nil)
}

func (p *ProxmoxProvider) apiCreateInstanceWithProgress(ctx context.Context, config provider.InstanceConfig, progressCallback provider.ProgressCallback) error {
	// 进度更新辅助函数
	updateProgress := func(percentage int, message string) {
		if progressCallback != nil {
			progressCallback(percentage, message)
		}
		global.APP_LOG.Info("Proxmox API实例创建进度",
			zap.String("instance", config.Name),
			zap.Int("percentage", percentage),
			zap.String("message", message))
	}

	updateProgress(10, "开始Proxmox API创建实例...")

	// 在API创建之前，处理镜像下载和导入
	updateProgress(30, "处理镜像下载和导入...")
	if err := p.handleImageDownloadAndImport(ctx, &config); err != nil {
		return fmt.Errorf("镜像处理失败: %w", err)
	}

	updateProgress(50, "调用Proxmox API创建实例...")
	// TODO: 实现 Proxmox API 创建实例的具体逻辑
	updateProgress(100, "Proxmox API实例创建完成")
	return fmt.Errorf("API create instance not implemented yet")
}

func (p *ProxmoxProvider) apiStartInstance(ctx context.Context, id string) error {
	url := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/qemu/%s/status/start", p.config.Host, p.node, id)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}

	// 设置认证头
	p.setAPIAuth(req)

	resp, err := p.apiClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to start VM: %d", resp.StatusCode)
	}

	return nil
}

func (p *ProxmoxProvider) apiStopInstance(ctx context.Context, id string) error {
	url := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/qemu/%s/status/stop", p.config.Host, p.node, id)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}

	// 设置认证头
	p.setAPIAuth(req)

	resp, err := p.apiClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to stop VM: %d", resp.StatusCode)
	}

	return nil
}

func (p *ProxmoxProvider) apiRestartInstance(ctx context.Context, id string) error {
	url := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/qemu/%s/status/reboot", p.config.Host, p.node, id)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}

	// 设置认证头
	p.setAPIAuth(req)

	resp, err := p.apiClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to restart VM: %d", resp.StatusCode)
	}

	return nil
}

func (p *ProxmoxProvider) apiDeleteInstance(ctx context.Context, id string) error {
	// 先通过SSH查找实例信息（API可能无法直接获取所有必要信息）
	vmid, instanceType, err := p.findVMIDByNameOrID(ctx, id)
	if err != nil {
		global.APP_LOG.Error("API删除: 无法找到实例对应的VMID",
			zap.String("id", id),
			zap.Error(err))
		return fmt.Errorf("无法找到实例 %s 对应的VMID: %w", id, err)
	}

	// 获取实例IP地址用于后续清理
	ipAddress, err := p.getInstanceIPAddress(ctx, vmid, instanceType)
	if err != nil {
		global.APP_LOG.Warn("API删除: 无法获取实例IP地址",
			zap.String("id", id),
			zap.String("vmid", vmid),
			zap.Error(err))
		ipAddress = "" // 继续执行，但IP地址为空
	}

	global.APP_LOG.Info("开始API删除Proxmox实例",
		zap.String("id", id),
		zap.String("vmid", vmid),
		zap.String("type", instanceType),
		zap.String("ip", ipAddress))

	// 在删除实例前先清理vnstat监控
	if err := p.cleanupVnStatMonitoring(ctx, id); err != nil {
		global.APP_LOG.Warn("API删除: 清理vnstat监控失败",
			zap.String("id", id),
			zap.String("vmid", vmid),
			zap.Error(err))
	}

	// 根据实例类型选择不同的API端点
	if instanceType == "container" {
		return p.apiDeleteContainer(ctx, vmid, ipAddress)
	} else {
		return p.apiDeleteVM(ctx, vmid, ipAddress)
	}
}

// apiDeleteVM 通过API删除虚拟机
func (p *ProxmoxProvider) apiDeleteVM(ctx context.Context, vmid string, ipAddress string) error {
	global.APP_LOG.Info("开始API删除VM流程",
		zap.String("vmid", vmid),
		zap.String("ip", ipAddress))

	// 1. 解锁VM（通过SSH，因为API可能不支持unlock操作）
	global.APP_LOG.Info("解锁VM", zap.String("vmid", vmid))
	_, err := p.sshClient.Execute(fmt.Sprintf("qm unlock %s 2>/dev/null || true", vmid))
	if err != nil {
		global.APP_LOG.Warn("解锁VM失败", zap.String("vmid", vmid), zap.Error(err))
	}

	// 2. 停止VM
	global.APP_LOG.Info("停止VM", zap.String("vmid", vmid))
	stopURL := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/qemu/%s/status/stop", p.config.Host, p.node, vmid)
	stopReq, err := http.NewRequestWithContext(ctx, "POST", stopURL, nil)
	if err != nil {
		return fmt.Errorf("创建停止请求失败: %w", err)
	}
	p.setAPIAuth(stopReq)

	stopResp, err := p.apiClient.Do(stopReq)
	if err != nil {
		global.APP_LOG.Warn("API停止VM失败，尝试SSH方式", zap.String("vmid", vmid), zap.Error(err))
		_, _ = p.sshClient.Execute(fmt.Sprintf("qm stop %s 2>/dev/null || true", vmid))
	} else {
		stopResp.Body.Close()
	}

	// 3. 检查VM是否完全停止
	if err := p.checkVMCTStatus(ctx, vmid, "vm"); err != nil {
		global.APP_LOG.Warn("VM未完全停止", zap.String("vmid", vmid), zap.Error(err))
		// 继续执行删除，但记录警告
	}

	// 4. 删除VM
	global.APP_LOG.Info("销毁VM", zap.String("vmid", vmid))
	deleteURL := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/qemu/%s", p.config.Host, p.node, vmid)
	deleteReq, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("创建删除请求失败: %w", err)
	}
	p.setAPIAuth(deleteReq)

	deleteResp, err := p.apiClient.Do(deleteReq)
	if err != nil {
		return fmt.Errorf("API删除VM失败: %w", err)
	}
	defer deleteResp.Body.Close()

	if deleteResp.StatusCode != http.StatusOK {
		return fmt.Errorf("API删除VM失败，状态码: %d", deleteResp.StatusCode)
	}

	// 执行后续清理工作（通过SSH，因为这些操作API通常不支持）
	return p.performPostDeletionCleanup(ctx, vmid, ipAddress, "vm")
}

// apiDeleteContainer 通过API删除容器
func (p *ProxmoxProvider) apiDeleteContainer(ctx context.Context, ctid string, ipAddress string) error {
	global.APP_LOG.Info("开始API删除CT流程",
		zap.String("ctid", ctid),
		zap.String("ip", ipAddress))

	// 1. 停止容器
	global.APP_LOG.Info("停止CT", zap.String("ctid", ctid))
	stopURL := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/lxc/%s/status/stop", p.config.Host, p.node, ctid)
	stopReq, err := http.NewRequestWithContext(ctx, "POST", stopURL, nil)
	if err != nil {
		return fmt.Errorf("创建停止请求失败: %w", err)
	}
	p.setAPIAuth(stopReq)

	stopResp, err := p.apiClient.Do(stopReq)
	if err != nil {
		global.APP_LOG.Warn("API停止CT失败，尝试SSH方式", zap.String("ctid", ctid), zap.Error(err))
		_, _ = p.sshClient.Execute(fmt.Sprintf("pct stop %s 2>/dev/null || true", ctid))
	} else {
		stopResp.Body.Close()
	}

	// 2. 检查容器是否完全停止
	if err := p.checkVMCTStatus(ctx, ctid, "container"); err != nil {
		global.APP_LOG.Warn("CT未完全停止", zap.String("ctid", ctid), zap.Error(err))
		// 继续执行删除，但记录警告
	}

	// 3. 删除容器
	global.APP_LOG.Info("销毁CT", zap.String("ctid", ctid))
	deleteURL := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/lxc/%s", p.config.Host, p.node, ctid)
	deleteReq, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("创建删除请求失败: %w", err)
	}
	p.setAPIAuth(deleteReq)

	deleteResp, err := p.apiClient.Do(deleteReq)
	if err != nil {
		return fmt.Errorf("API删除CT失败: %w", err)
	}
	defer deleteResp.Body.Close()

	if deleteResp.StatusCode != http.StatusOK {
		return fmt.Errorf("API删除CT失败，状态码: %d", deleteResp.StatusCode)
	}

	// 执行后续清理工作（通过SSH）
	return p.performPostDeletionCleanup(ctx, ctid, ipAddress, "container")
}

// performPostDeletionCleanup 执行删除后的清理工作
func (p *ProxmoxProvider) performPostDeletionCleanup(ctx context.Context, vmctid string, ipAddress string, instanceType string) error {
	global.APP_LOG.Info("执行删除后清理工作",
		zap.String("vmctid", vmctid),
		zap.String("type", instanceType),
		zap.String("ip", ipAddress))

	// 清理IPv6 NAT映射规则
	if err := p.cleanupIPv6NATRules(ctx, vmctid); err != nil {
		global.APP_LOG.Warn("清理IPv6 NAT规则失败", zap.String("vmctid", vmctid), zap.Error(err))
	}

	// 清理文件
	if instanceType == "vm" {
		if err := p.cleanupVMFiles(ctx, vmctid); err != nil {
			global.APP_LOG.Warn("清理VM文件失败", zap.String("vmid", vmctid), zap.Error(err))
		}
	} else {
		if err := p.cleanupCTFiles(ctx, vmctid); err != nil {
			global.APP_LOG.Warn("清理CT文件失败", zap.String("ctid", vmctid), zap.Error(err))
		}
	}

	// 更新iptables规则
	if ipAddress != "" {
		if err := p.updateIPTablesRules(ctx, ipAddress); err != nil {
			global.APP_LOG.Warn("更新iptables规则失败", zap.String("ip", ipAddress), zap.Error(err))
		}
	}

	// 重建iptables规则
	if err := p.rebuildIPTablesRules(ctx); err != nil {
		global.APP_LOG.Warn("重建iptables规则失败", zap.Error(err))
	}

	// 重启ndpresponder服务
	if err := p.restartNDPResponder(ctx); err != nil {
		global.APP_LOG.Warn("重启ndpresponder服务失败", zap.Error(err))
	}

	global.APP_LOG.Info("通过API成功删除Proxmox实例",
		zap.String("vmctid", vmctid),
		zap.String("type", instanceType))
	return nil
}

func (p *ProxmoxProvider) apiListImages(ctx context.Context) ([]provider.Image, error) {
	url := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/storage/local/content", p.config.Host, p.node)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 设置认证头
	p.setAPIAuth(req)

	resp, err := p.apiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	var images []provider.Image
	if data, ok := response["data"].([]interface{}); ok {
		for _, item := range data {
			if imageData, ok := item.(map[string]interface{}); ok {
				if imageData["content"].(string) == "iso" {
					image := provider.Image{
						ID:   imageData["volid"].(string),
						Name: imageData["volid"].(string),
						Tag:  "iso",
						Size: fmt.Sprintf("%.2f MB", imageData["size"].(float64)/1024/1024),
					}
					images = append(images, image)
				}
			}
		}
	}

	return images, nil
}

func (p *ProxmoxProvider) apiPullImage(ctx context.Context, image string) error {
	// 实现 Proxmox API 下载镜像
	return fmt.Errorf("API pull image not implemented yet")
}

func (p *ProxmoxProvider) apiDeleteImage(ctx context.Context, id string) error {
	url := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/storage/local/content/%s", p.config.Host, p.node, id)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	// 设置认证头
	p.setAPIAuth(req)

	resp, err := p.apiClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete image: %d", resp.StatusCode)
	}

	return nil
}

// apiSetInstancePassword 通过API设置实例密码
func (p *ProxmoxProvider) apiSetInstancePassword(ctx context.Context, instanceID, password string) error {
	// TODO: Proxmox API方式设置密码
	// Proxmox的密码设置需要通过特定的API调用
	// 这里先返回错误，建议使用SSH方式
	return fmt.Errorf("Proxmox API密码设置暂未实现，请使用SSH方式")
}
