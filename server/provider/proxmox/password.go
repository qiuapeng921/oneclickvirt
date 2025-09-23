package proxmox

import (
	"context"
	"fmt"

	"oneclickvirt/utils"
)

// SetInstancePassword 设置实例密码
func (p *ProxmoxProvider) SetInstancePassword(ctx context.Context, instanceID, password string) error {
	if !p.connected {
		return fmt.Errorf("provider not connected")
	}

	// 根据配置选择使用API还是SSH方式
	if p.config.Token != "" && p.config.TokenID != "" {
		return p.apiSetInstancePassword(ctx, instanceID, password)
	} else {
		return p.sshSetInstancePassword(ctx, instanceID, password)
	}
}

// ResetInstancePassword 重置实例密码
func (p *ProxmoxProvider) ResetInstancePassword(ctx context.Context, instanceID string) (string, error) {
	if !p.connected {
		return "", fmt.Errorf("provider not connected")
	}

	// 生成随机密码
	newPassword := p.generateRandomPassword()

	// 设置新密码
	err := p.SetInstancePassword(ctx, instanceID, newPassword)
	if err != nil {
		return "", err
	}

	return newPassword, nil
}

// generateRandomPassword 生成随机密码（仅包含数字和大小写英文字母，长度不低于8位）
func (p *ProxmoxProvider) generateRandomPassword() string {
	return utils.GenerateInstancePassword()
}
