package health

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// ProxmoxHealthChecker Proxmox健康检查器
type ProxmoxHealthChecker struct {
	*BaseHealthChecker
	sshClient *ssh.Client
}

// NewProxmoxHealthChecker 创建Proxmox健康检查器
func NewProxmoxHealthChecker(config HealthConfig, logger *zap.Logger) *ProxmoxHealthChecker {
	return &ProxmoxHealthChecker{
		BaseHealthChecker: NewBaseHealthChecker(config, logger),
	}
}

// CheckHealth 执行Proxmox健康检查
func (p *ProxmoxHealthChecker) CheckHealth(ctx context.Context) (*HealthResult, error) {
	checks := []func(context.Context) CheckResult{}

	// SSH检查
	if p.config.SSHEnabled {
		checks = append(checks, p.createCheckFunc(CheckTypeSSH, p.checkSSH))
	}

	// API检查
	if p.config.APIEnabled {
		checks = append(checks, p.createCheckFunc(CheckTypeAPI, p.checkAPI))
	}

	// Proxmox服务检查
	if len(p.config.ServiceChecks) > 0 {
		checks = append(checks, p.createCheckFunc(CheckTypeService, p.checkProxmoxService))
	}

	result := p.executeChecks(ctx, checks)
	return result, nil
}

// checkSSH 检查SSH连接
func (p *ProxmoxHealthChecker) checkSSH(ctx context.Context) error {
	if p.sshClient != nil {
		// 测试现有连接
		_, err := p.sshClient.NewSession()
		if err == nil {
			return nil
		}
	}

	// 构建认证方法：优先使用SSH密钥，否则使用密码
	var authMethods []ssh.AuthMethod

	// 如果提供了SSH私钥，优先使用密钥认证
	if p.config.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(p.config.PrivateKey))
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
			if p.logger != nil {
				p.logger.Debug("使用SSH密钥认证", zap.String("host", p.config.Host))
			}
		} else if p.logger != nil {
			p.logger.Warn("SSH私钥解析失败，将尝试使用密码认证",
				zap.String("host", p.config.Host),
				zap.Error(err))
		}
	}

	// 如果没有密钥或密钥解析失败，且提供了密码，使用密码认证
	if len(authMethods) == 0 && p.config.Password != "" {
		authMethods = append(authMethods, ssh.Password(p.config.Password))
		if p.logger != nil {
			p.logger.Debug("使用SSH密码认证", zap.String("host", p.config.Host))
		}
	}

	// 如果既没有密钥也没有密码，返回错误
	if len(authMethods) == 0 {
		return fmt.Errorf("no authentication method available: neither SSH key nor password provided")
	}

	// 建立新连接
	config := &ssh.ClientConfig{
		User:            p.config.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         p.config.Timeout,
	}

	address := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %w", err)
	}

	p.sshClient = client
	if p.logger != nil {
		p.logger.Debug("Proxmox SSH连接成功", zap.String("host", p.config.Host), zap.Int("port", p.config.Port))
	}
	return nil
}

// checkAPI 检查Proxmox API
func (p *ProxmoxHealthChecker) checkAPI(ctx context.Context) error {
	// Proxmox API标准端口是8006
	url := fmt.Sprintf("https://%s:8006/api2/json/nodes", p.config.Host)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建API请求失败: %w", err)
	}

	// 设置认证头
	if p.config.Token != "" && p.config.TokenID != "" {
		// 清理Token ID和Token中的不可见字符（换行符、回车符、制表符等）
		cleanTokenID := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(p.config.TokenID), "\n", ""), "\r", "")
		cleanToken := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(p.config.Token), "\n", ""), "\r", "")
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", cleanTokenID, cleanToken))
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Proxmox API连接失败 (检查Proxmox是否运行且API端口8006可访问，以及Token配置是否正确): %w", err)
	}
	defer resp.Body.Close()

	// 只有成功获取到节点列表才认为API健康
	// 401表示认证失败，说明Token配置有问题
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("Proxmox API认证失败 (状态码: %d) - 请检查API Token和TokenID配置", resp.StatusCode)
		}
		return fmt.Errorf("Proxmox API返回错误状态码: %d", resp.StatusCode)
	}

	if p.logger != nil {
		p.logger.Debug("Proxmox API检查成功", zap.String("url", url), zap.Int("status", resp.StatusCode))
	}
	return nil
}

// checkProxmoxService 检查Proxmox服务状态
func (p *ProxmoxHealthChecker) checkProxmoxService(ctx context.Context) error {
	if p.sshClient == nil {
		// 如果没有SSH连接，先建立连接
		if err := p.checkSSH(ctx); err != nil {
			return fmt.Errorf("无法建立SSH连接进行服务检查: %w", err)
		}
	}

	// 检查PVE版本
	session, err := p.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	// 请求PTY以模拟交互式登录shell，确保加载完整的环境变量
	err = session.RequestPty("xterm", 80, 40, ssh.TerminalModes{
		ssh.ECHO:          0,     // 禁用回显
		ssh.TTY_OP_ISPEED: 14400, // 输入速度
		ssh.TTY_OP_OSPEED: 14400, // 输出速度
	})
	if err != nil {
		return fmt.Errorf("请求PTY失败: %w", err)
	}

	// 设置环境变量来确保PATH正确加载，避免bash -l -c的转义问题
	envCommand := "source /etc/profile 2>/dev/null || true; source ~/.bashrc 2>/dev/null || true; source ~/.bash_profile 2>/dev/null || true; export PATH=$PATH:/usr/local/bin:/snap/bin:/usr/sbin:/sbin; pveversion"
	output, err := session.CombinedOutput(envCommand)
	if err != nil {
		return fmt.Errorf("Proxmox服务不可用: %w", err)
	}

	if !strings.Contains(string(output), "proxmox-ve") {
		return fmt.Errorf("Proxmox VE未正确安装")
	}

	// 检查关键服务状态
	services := []string{"pvedaemon", "pveproxy", "pvestatd"}
	for _, service := range services {
		session, err := p.sshClient.NewSession()
		if err != nil {
			return fmt.Errorf("创建SSH会话失败: %w", err)
		}

		// 请求PTY以模拟交互式登录shell
		err = session.RequestPty("xterm", 80, 40, ssh.TerminalModes{
			ssh.ECHO:          0,     // 禁用回显
			ssh.TTY_OP_ISPEED: 14400, // 输入速度
			ssh.TTY_OP_OSPEED: 14400, // 输出速度
		})
		if err != nil {
			session.Close()
			return fmt.Errorf("请求PTY失败: %w", err)
		}

		// 设置环境变量来确保PATH正确加载
		envCommand := fmt.Sprintf("source /etc/profile 2>/dev/null || true; source ~/.bashrc 2>/dev/null || true; source ~/.bash_profile 2>/dev/null || true; export PATH=$PATH:/usr/local/bin:/snap/bin:/usr/sbin:/sbin; systemctl is-active %s", service)
		_, err = session.CombinedOutput(envCommand)
		session.Close()

		if err != nil {
			return fmt.Errorf("Proxmox服务 %s 未运行: %w", service, err)
		}
	}

	if p.logger != nil {
		p.logger.Debug("Proxmox服务检查成功", zap.String("host", p.config.Host), zap.Strings("services", services))
	}
	return nil
}

// Close 关闭连接
func (p *ProxmoxHealthChecker) Close() error {
	if p.sshClient != nil {
		err := p.sshClient.Close()
		p.sshClient = nil
		return err
	}
	return nil
}
