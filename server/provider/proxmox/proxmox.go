package proxmox

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"oneclickvirt/global"
	"oneclickvirt/provider"
	"oneclickvirt/provider/health"
	"oneclickvirt/utils"

	"go.uber.org/zap"
)

type ProxmoxProvider struct {
	config        provider.NodeConfig
	sshClient     *utils.SSHClient
	apiClient     *http.Client
	connected     bool
	node          string // Proxmox 节点名
	providerUUID  string // Provider UUID，用于查询数据库中的配置
	healthChecker health.HealthChecker
}

func NewProxmoxProvider() provider.Provider {
	return &ProxmoxProvider{
		apiClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *ProxmoxProvider) GetType() string {
	return "proxmox"
}

func (p *ProxmoxProvider) GetName() string {
	return p.config.Name
}

func (p *ProxmoxProvider) GetSupportedInstanceTypes() []string {
	return []string{"container", "vm"}
}

func (p *ProxmoxProvider) Connect(ctx context.Context, config provider.NodeConfig) error {
	p.config = config
	p.providerUUID = config.UUID // 存储Provider UUID

	// 如果有本地存储的 Token 文件，尝试从文件加载 Token 信息
	if err := p.loadTokenFromFiles(); err != nil {
		global.APP_LOG.Warn("从本地文件加载token失败，使用配置值", zap.Error(err))
	}

	// 如果本地文件没有 Token，尝试从 NodeConfig 的扩展配置中解析
	if !p.hasAPIAccess() {
		if err := p.loadTokenFromConfig(); err != nil {
			global.APP_LOG.Warn("从配置加载token失败，将仅使用SSH", zap.Error(err))
		}
	}

	// 设置SSH超时配置
	sshConnectTimeout := config.SSHConnectTimeout
	sshExecuteTimeout := config.SSHExecuteTimeout
	if sshConnectTimeout <= 0 {
		sshConnectTimeout = 30 // 默认30秒
	}
	if sshExecuteTimeout <= 0 {
		sshExecuteTimeout = 300 // 默认300秒
	}

	// 尝试 SSH 连接
	sshConfig := utils.SSHConfig{
		Host:           config.Host,
		Port:           config.Port,
		Username:       config.Username,
		Password:       config.Password,
		PrivateKey:     config.PrivateKey,
		ConnectTimeout: time.Duration(sshConnectTimeout) * time.Second,
		ExecuteTimeout: time.Duration(sshExecuteTimeout) * time.Second,
	}

	client, err := utils.NewSSHClient(sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect via SSH: %w", err)
	}

	p.sshClient = client
	p.connected = true

	// 获取节点名
	if err := p.getNodeName(ctx); err != nil {
		global.APP_LOG.Warn("获取节点名称失败", zap.Error(err))
		p.node = "pve" // 默认节点名
	}

	// 初始化健康检查器
	healthConfig := health.HealthConfig{
		Host:          config.Host,
		Port:          config.Port,
		Username:      config.Username,
		Password:      config.Password,
		PrivateKey:    config.PrivateKey,
		APIEnabled:    p.hasAPIAccess(),
		APIPort:       8006,
		APIScheme:     "https",
		SSHEnabled:    true,
		Timeout:       30 * time.Second,
		ServiceChecks: []string{"pvestatd", "pvedaemon", "pveproxy"},
		Token:         config.Token,
		TokenID:       config.TokenID,
	}

	zapLogger, _ := zap.NewProduction()
	p.healthChecker = health.NewProxmoxHealthChecker(healthConfig, zapLogger)

	global.APP_LOG.Info("Proxmox provider SSH连接成功",
		zap.String("host", utils.TruncateString(config.Host, 32)),
		zap.Int("port", config.Port),
		zap.String("node", utils.TruncateString(p.node, 32)),
		zap.Bool("hasToken", p.hasAPIAccess()))

	return nil
}

func (p *ProxmoxProvider) Disconnect(ctx context.Context) error {
	if p.sshClient != nil {
		p.sshClient.Close()
		p.sshClient = nil
	}
	p.connected = false
	return nil
}

func (p *ProxmoxProvider) IsConnected() bool {
	return p.connected && p.sshClient != nil && p.sshClient.IsHealthy()
}

// EnsureConnection 确保SSH连接可用，如果连接不健康则尝试重连
func (p *ProxmoxProvider) EnsureConnection() error {
	if p.sshClient == nil {
		return fmt.Errorf("SSH client not initialized")
	}

	if !p.sshClient.IsHealthy() {
		global.APP_LOG.Warn("Proxmox Provider SSH连接不健康，尝试重连",
			zap.String("host", utils.TruncateString(p.config.Host, 32)),
			zap.Int("port", p.config.Port))

		if err := p.sshClient.Reconnect(); err != nil {
			p.connected = false
			return fmt.Errorf("failed to reconnect SSH: %w", err)
		}

		global.APP_LOG.Info("Proxmox Provider SSH连接重建成功",
			zap.String("host", utils.TruncateString(p.config.Host, 32)),
			zap.Int("port", p.config.Port))
	}

	return nil
}

func (p *ProxmoxProvider) HealthCheck(ctx context.Context) (*health.HealthResult, error) {
	if p.healthChecker == nil {
		return nil, fmt.Errorf("health checker not initialized")
	}
	return p.healthChecker.CheckHealth(ctx)
}

func (p *ProxmoxProvider) GetHealthChecker() health.HealthChecker {
	return p.healthChecker
}

// 获取节点名
func (p *ProxmoxProvider) getNodeName(ctx context.Context) error {
	output, err := p.sshClient.Execute("hostname")
	if err != nil {
		return err
	}
	p.node = strings.TrimSpace(output)
	return nil
}

// ExecuteSSHCommand 执行SSH命令
func (p *ProxmoxProvider) ExecuteSSHCommand(ctx context.Context, command string) (string, error) {
	if !p.connected || p.sshClient == nil {
		return "", fmt.Errorf("Proxmox provider not connected")
	}

	global.APP_LOG.Debug("执行SSH命令",
		zap.String("command", utils.TruncateString(command, 200)))

	output, err := p.sshClient.Execute(command)
	if err != nil {
		global.APP_LOG.Error("SSH命令执行失败",
			zap.String("command", utils.TruncateString(command, 200)),
			zap.String("output", utils.TruncateString(output, 500)),
			zap.Error(err))
		return "", fmt.Errorf("SSH command execution failed: %w", err)
	}

	return output, nil
}

// 检查是否有 API 访问权限
func (p *ProxmoxProvider) hasAPIAccess() bool {
	// 检查是否配置了 API Token ID 和 Token Secret
	return p.config.TokenID != "" && p.config.Token != ""
}

// setAPIAuth 为 HTTP 请求设置 API 认证头
func (p *ProxmoxProvider) setAPIAuth(req *http.Request) {
	if p.config.TokenID != "" && p.config.Token != "" {
		// 清理Token ID和Token中的不可见字符（换行符、回车符、制表符等）
		cleanTokenID := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(p.config.TokenID), "\n", ""), "\r", "")
		cleanToken := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(p.config.Token), "\n", ""), "\r", "")

		// 使用 API Token 认证，格式: PVEAPIToken=USER@REALM!TOKENID=SECRET
		authHeader := fmt.Sprintf("PVEAPIToken=%s=%s", cleanTokenID, cleanToken)
		req.Header.Set("Authorization", authHeader)
	}
}

// shouldUseAPI 根据执行规则判断是否应该使用API
func (p *ProxmoxProvider) shouldUseAPI() bool {
	switch p.config.ExecutionRule {
	case "api_only":
		return p.hasAPIAccess()
	case "ssh_only":
		return false
	case "auto":
		fallthrough
	default:
		return p.hasAPIAccess()
	}
}

// shouldUseSSH 根据执行规则判断是否应该使用SSH
func (p *ProxmoxProvider) shouldUseSSH() bool {
	switch p.config.ExecutionRule {
	case "api_only":
		return false
	case "ssh_only":
		return true
	case "auto":
		fallthrough
	default:
		return true
	}
}

// shouldFallbackToSSH 根据执行规则判断API失败时是否可以回退到SSH
func (p *ProxmoxProvider) shouldFallbackToSSH() bool {
	switch p.config.ExecutionRule {
	case "api_only":
		return false
	case "ssh_only":
		return false
	case "auto":
		fallthrough
	default:
		return true
	}
}

func init() {
	provider.RegisterProvider("proxmox", NewProxmoxProvider)
}
