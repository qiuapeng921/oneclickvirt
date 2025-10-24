package health

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// DockerHealthChecker Docker健康检查器
type DockerHealthChecker struct {
	*BaseHealthChecker
	sshClient *ssh.Client
}

// NewDockerHealthChecker 创建Docker健康检查器
func NewDockerHealthChecker(config HealthConfig, logger *zap.Logger) *DockerHealthChecker {
	return &DockerHealthChecker{
		BaseHealthChecker: NewBaseHealthChecker(config, logger),
	}
}

// CheckHealth 执行Docker健康检查
func (d *DockerHealthChecker) CheckHealth(ctx context.Context) (*HealthResult, error) {
	checks := []func(context.Context) CheckResult{}

	// SSH检查
	if d.config.SSHEnabled {
		checks = append(checks, d.createCheckFunc(CheckTypeSSH, d.checkSSH))
	}

	// API检查
	if d.config.APIEnabled {
		checks = append(checks, d.createCheckFunc(CheckTypeAPI, d.checkAPI))
	}

	// Docker服务检查
	if len(d.config.ServiceChecks) > 0 {
		checks = append(checks, d.createCheckFunc(CheckTypeService, d.checkDockerService))
	}

	result := d.executeChecks(ctx, checks)
	return result, nil
}

// checkSSH 检查SSH连接
func (d *DockerHealthChecker) checkSSH(ctx context.Context) error {
	if d.sshClient != nil {
		// 测试现有连接
		_, err := d.sshClient.NewSession()
		if err == nil {
			return nil
		}
	}

	// 构建认证方法：优先使用SSH密钥，否则使用密码
	// 构建认证方法：支持密钥和密码，SSH客户端会按顺序尝试
	var authMethods []ssh.AuthMethod

	// 如果提供了SSH私钥，添加密钥认证
	if d.config.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(d.config.PrivateKey))
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
			if d.logger != nil {
				d.logger.Debug("已添加SSH密钥认证方法", zap.String("host", d.config.Host))
			}
		} else if d.logger != nil {
			d.logger.Warn("SSH私钥解析失败，将尝试使用密码认证",
				zap.String("host", d.config.Host),
				zap.Error(err))
		}
	}

	// 如果提供了密码，添加密码认证（无论是否有密钥，都添加作为备用方案）
	if d.config.Password != "" {
		authMethods = append(authMethods, ssh.Password(d.config.Password))
		if d.logger != nil {
			d.logger.Debug("已添加SSH密码认证方法", zap.String("host", d.config.Host))
		}
	}

	// 如果既没有密钥也没有密码，返回错误
	if len(authMethods) == 0 {
		return fmt.Errorf("no authentication method available: neither SSH key nor password provided")
	}

	// 建立新连接
	config := &ssh.ClientConfig{
		User:            d.config.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         d.config.Timeout,
	}

	address := fmt.Sprintf("%s:%d", d.config.Host, d.config.Port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %w", err)
	}

	d.sshClient = client
	if d.logger != nil {
		d.logger.Debug("Docker SSH连接成功", zap.String("host", d.config.Host), zap.Int("port", d.config.Port))
	}
	return nil
}

// checkAPI 检查Docker API
func (d *DockerHealthChecker) checkAPI(ctx context.Context) error {
	url := fmt.Sprintf("%s://%s:%d/version", d.config.APIScheme, d.config.Host, d.config.APIPort)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建API请求失败: %w", err)
	}

	// 如果有证书配置，设置TLS
	if d.config.CertPath != "" && d.config.KeyPath != "" {
		// 不做处理
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}

	if d.logger != nil {
		d.logger.Debug("Docker API检查成功", zap.String("url", url), zap.Int("status", resp.StatusCode))
	}
	return nil
}

// checkDockerService 检查Docker服务状态
func (d *DockerHealthChecker) checkDockerService(ctx context.Context) error {
	if d.sshClient == nil {
		// 如果没有SSH连接，先建立连接
		if err := d.checkSSH(ctx); err != nil {
			return fmt.Errorf("无法建立SSH连接进行服务检查: %w", err)
		}
	}

	// 执行Docker版本检查
	session, err := d.sshClient.NewSession()
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
	envCommand := "source /etc/profile 2>/dev/null || true; source ~/.bashrc 2>/dev/null || true; source ~/.bash_profile 2>/dev/null || true; export PATH=$PATH:/usr/local/bin:/snap/bin:/usr/sbin:/sbin; docker version"
	output, err := session.CombinedOutput(envCommand)
	if err != nil {
		return fmt.Errorf("Docker服务不可用: %w", err)
	}

	if !strings.Contains(string(output), "Server:") {
		return fmt.Errorf("Docker守护进程未运行")
	}

	if d.logger != nil {
		d.logger.Debug("Docker服务检查成功", zap.String("host", d.config.Host))
	}
	return nil
}

// Close 关闭连接
func (d *DockerHealthChecker) Close() error {
	if d.sshClient != nil {
		err := d.sshClient.Close()
		d.sshClient = nil
		return err
	}
	return nil
}
