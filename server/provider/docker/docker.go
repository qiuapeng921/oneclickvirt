package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"oneclickvirt/global"
	"oneclickvirt/provider"
	"oneclickvirt/provider/health"
	"oneclickvirt/utils"

	"go.uber.org/zap"
)

type DockerProvider struct {
	config        provider.NodeConfig
	sshClient     *utils.SSHClient
	connected     bool
	healthChecker health.HealthChecker
}

func NewDockerProvider() provider.Provider {
	return &DockerProvider{}
}

func (d *DockerProvider) GetType() string {
	return "docker"
}

func (d *DockerProvider) GetName() string {
	return d.config.Name
}

func (d *DockerProvider) GetSupportedInstanceTypes() []string {
	return []string{"container"}
}

func (d *DockerProvider) Connect(ctx context.Context, config provider.NodeConfig) error {
	d.config = config
	global.APP_LOG.Info("Docker provider开始连接",
		zap.String("host", utils.TruncateString(config.Host, 32)),
		zap.Int("port", config.Port))

	// 设置SSH超时配置
	sshConnectTimeout := config.SSHConnectTimeout
	sshExecuteTimeout := config.SSHExecuteTimeout
	if sshConnectTimeout <= 0 {
		sshConnectTimeout = 30 // 默认30秒
	}
	if sshExecuteTimeout <= 0 {
		sshExecuteTimeout = 300 // 默认300秒
	}

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

	d.sshClient = client
	d.connected = true

	// 初始化健康检查器
	healthConfig := health.HealthConfig{
		Host:          config.Host,
		Port:          config.Port,
		Username:      config.Username,
		Password:      config.Password,
		PrivateKey:    config.PrivateKey,
		APIEnabled:    false, // Docker Provider 不使用 API
		SSHEnabled:    true,
		Timeout:       30 * time.Second,
		ServiceChecks: []string{"docker"},
	}

	// 创建一个简单的zap logger实例给健康检查器使用
	zapLogger, _ := zap.NewProduction()
	d.healthChecker = health.NewDockerHealthChecker(healthConfig, zapLogger)

	global.APP_LOG.Info("Docker provider连接成功",
		zap.String("host", utils.TruncateString(config.Host, 32)),
		zap.Int("port", config.Port))

	return nil
}

func (d *DockerProvider) Disconnect(ctx context.Context) error {
	if d.sshClient != nil {
		d.sshClient.Close()
		d.connected = false
	}
	return nil
}

func (d *DockerProvider) IsConnected() bool {
	return d.connected && d.sshClient != nil && d.sshClient.IsHealthy()
}

// EnsureConnection 确保SSH连接可用，如果连接不健康则尝试重连
func (d *DockerProvider) EnsureConnection() error {
	if d.sshClient == nil {
		return fmt.Errorf("SSH client not initialized")
	}

	if !d.sshClient.IsHealthy() {
		global.APP_LOG.Warn("Docker Provider SSH连接不健康，尝试重连",
			zap.String("host", utils.TruncateString(d.config.Host, 32)),
			zap.Int("port", d.config.Port))

		if err := d.sshClient.Reconnect(); err != nil {
			d.connected = false
			return fmt.Errorf("failed to reconnect SSH: %w", err)
		}

		global.APP_LOG.Info("Docker Provider SSH连接重建成功",
			zap.String("host", utils.TruncateString(d.config.Host, 32)),
			zap.Int("port", d.config.Port))
	}

	return nil
}

func (d *DockerProvider) HealthCheck(ctx context.Context) (*health.HealthResult, error) {
	if d.healthChecker == nil {
		return nil, fmt.Errorf("health checker not initialized")
	}
	return d.healthChecker.CheckHealth(ctx)
}

func (d *DockerProvider) GetHealthChecker() health.HealthChecker {
	return d.healthChecker
}

func (d *DockerProvider) ListInstances(ctx context.Context) ([]provider.Instance, error) {
	if !d.connected {
		return nil, fmt.Errorf("not connected")
	}

	return d.sshListInstances(ctx)
}

func (d *DockerProvider) CreateInstance(ctx context.Context, config provider.InstanceConfig) error {
	if !d.connected {
		return fmt.Errorf("not connected")
	}

	// Docker provider只支持SSH，检查执行规则
	if d.config.ExecutionRule == "api_only" {
		return fmt.Errorf("Docker provider不支持API调用，无法使用api_only执行规则")
	}

	return d.sshCreateInstance(ctx, config)
}

func (d *DockerProvider) CreateInstanceWithProgress(ctx context.Context, config provider.InstanceConfig, progressCallback provider.ProgressCallback) error {
	if !d.connected {
		return fmt.Errorf("not connected")
	}

	// Docker provider只支持SSH，检查执行规则
	if d.config.ExecutionRule == "api_only" {
		return fmt.Errorf("Docker provider不支持API调用，无法使用api_only执行规则")
	}

	return d.sshCreateInstanceWithProgress(ctx, config, progressCallback)
}

func (d *DockerProvider) StartInstance(ctx context.Context, id string) error {
	if !d.connected {
		return fmt.Errorf("not connected")
	}

	// Docker provider只支持SSH，检查执行规则
	if d.config.ExecutionRule == "api_only" {
		return fmt.Errorf("Docker provider不支持API调用，无法使用api_only执行规则")
	}

	return d.sshStartInstance(ctx, id)
}

func (d *DockerProvider) StopInstance(ctx context.Context, id string) error {
	if !d.connected {
		return fmt.Errorf("not connected")
	}

	// Docker provider只支持SSH，检查执行规则
	if d.config.ExecutionRule == "api_only" {
		return fmt.Errorf("Docker provider不支持API调用，无法使用api_only执行规则")
	}

	return d.sshStopInstance(ctx, id)
}

func (d *DockerProvider) RestartInstance(ctx context.Context, id string) error {
	if !d.connected {
		return fmt.Errorf("not connected")
	}

	// Docker provider只支持SSH，检查执行规则
	if d.config.ExecutionRule == "api_only" {
		return fmt.Errorf("Docker provider不支持API调用，无法使用api_only执行规则")
	}

	return d.sshRestartInstance(ctx, id)
}

func (d *DockerProvider) DeleteInstance(ctx context.Context, id string) error {
	// Docker provider只支持SSH，检查执行规则
	if d.config.ExecutionRule == "api_only" {
		return fmt.Errorf("Docker provider不支持API调用，无法使用api_only执行规则")
	}

	// 增强版删除实例，带重连机制
	maxReconnectAttempts := 3
	for attempt := 1; attempt <= maxReconnectAttempts; attempt++ {
		// 检查连接状态
		if !d.connected {
			global.APP_LOG.Warn("Docker Provider未连接，尝试重连",
				zap.String("id", utils.TruncateString(id, 32)),
				zap.Int("attempt", attempt))

			// 尝试重连
			if err := d.Connect(ctx, d.config); err != nil {
				global.APP_LOG.Error("Docker Provider重连失败",
					zap.String("id", utils.TruncateString(id, 32)),
					zap.Int("attempt", attempt),
					zap.Error(err))

				if attempt == maxReconnectAttempts {
					return fmt.Errorf("重连失败，已达最大重试次数: %w", err)
				}
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
		}

		// 尝试删除实例
		err := d.sshDeleteInstance(ctx, id)
		if err != nil {
			// 如果是连接相关错误，标记为未连接并重试
			if d.isConnectionError(err) {
				global.APP_LOG.Warn("检测到连接错误，标记为未连接",
					zap.String("id", utils.TruncateString(id, 32)),
					zap.Int("attempt", attempt),
					zap.Error(err))
				d.connected = false

				if attempt < maxReconnectAttempts {
					time.Sleep(time.Duration(attempt) * time.Second)
					continue
				}
			}
			return err
		}

		// 删除成功
		return nil
	}

	return fmt.Errorf("删除实例失败，已达最大重连尝试次数")
}

// isConnectionError 判断是否是连接相关的错误
func (d *DockerProvider) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := strings.ToLower(err.Error())
	connectionErrors := []string{
		"connection refused",
		"connection lost",
		"connection reset",
		"network is unreachable",
		"no route to host",
		"connection timed out",
		"broken pipe",
		"eof",
		"ssh: connection lost",
		"ssh: handshake failed",
		"ssh: unable to authenticate",
	}

	for _, connErr := range connectionErrors {
		if strings.Contains(errorStr, connErr) {
			return true
		}
	}

	return false
}

func (d *DockerProvider) ListImages(ctx context.Context) ([]provider.Image, error) {
	if !d.connected {
		return nil, fmt.Errorf("not connected")
	}

	return d.sshListImages(ctx)
}

func (d *DockerProvider) PullImage(ctx context.Context, image string) error {
	if !d.connected {
		return fmt.Errorf("not connected")
	}

	return d.sshPullImage(ctx, image)
}

func (d *DockerProvider) DeleteImage(ctx context.Context, id string) error {
	if !d.connected {
		return fmt.Errorf("not connected")
	}

	return d.sshDeleteImage(ctx, id)
}

func (d *DockerProvider) GetInstance(ctx context.Context, id string) (*provider.Instance, error) {
	if !d.connected {
		return nil, fmt.Errorf("not connected")
	}

	// 使用简单的分隔符格式获取信息，避免table格式的解析问题
	output, err := d.sshClient.ExecuteWithLogging(fmt.Sprintf("docker inspect %s --format '{{.Name}}|{{.State.Status}}|{{.Config.Image}}|{{.Id}}|{{.Created}}'", id), "DOCKER_INSPECT")
	if err != nil {
		global.APP_LOG.Debug("Docker inspect命令执行失败",
			zap.String("id", utils.TruncateString(id, 32)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// 解析输出
	output = strings.TrimSpace(output)
	if output == "" {
		global.APP_LOG.Debug("Docker inspect返回空输出",
			zap.String("id", utils.TruncateString(id, 32)))
		return nil, fmt.Errorf("instance not found")
	}

	// 按|分割字段
	fields := strings.Split(output, "|")
	if len(fields) < 4 {
		global.APP_LOG.Warn("Docker inspect输出格式不正确",
			zap.String("id", utils.TruncateString(id, 32)),
			zap.String("output", utils.TruncateString(output, 200)),
			zap.Int("fields_count", len(fields)))
		return nil, fmt.Errorf("invalid instance data: unexpected format")
	}

	status := "unknown"
	statusField := strings.ToLower(fields[1])
	if strings.Contains(statusField, "running") {
		status = "running"
	} else if strings.Contains(statusField, "exited") {
		status = "stopped"
	} else if strings.Contains(statusField, "paused") {
		status = "paused"
	}

	instance := &provider.Instance{
		ID:     fields[3],
		Name:   strings.TrimPrefix(fields[0], "/"),
		Status: status,
		Image:  fields[2],
	}

	global.APP_LOG.Debug("Docker实例信息获取成功",
		zap.String("id", utils.TruncateString(id, 32)),
		zap.String("name", instance.Name),
		zap.String("status", instance.Status))

	return instance, nil
}

// checkIPv6NetworkAvailable 检查IPv6网络是否可用
func (d *DockerProvider) checkIPv6NetworkAvailable() bool {
	if !d.connected || d.sshClient == nil {
		return false
	}

	// 检查 ipv6_net 网络是否存在
	_, err := d.sshClient.Execute("docker network inspect ipv6_net")
	if err != nil {
		global.APP_LOG.Debug("IPv6网络检查失败: ipv6_net网络不存在",
			zap.String("provider", d.config.Name),
			zap.Error(err))
		return false
	}

	// 检查 ndpresponder 容器是否存在且正在运行
	ndpresponderCmd := "docker inspect -f '{{.State.Status}}' ndpresponder 2>/dev/null"
	ndpresponderOutput, err := d.sshClient.Execute(ndpresponderCmd)
	if err != nil {
		global.APP_LOG.Debug("IPv6网络检查: ndpresponder容器不存在",
			zap.String("provider", d.config.Name))
		return false
	}

	ndpresponderStatus := strings.TrimSpace(ndpresponderOutput)
	if ndpresponderStatus != "running" {
		global.APP_LOG.Debug("IPv6网络检查: ndpresponder容器未运行",
			zap.String("provider", d.config.Name),
			zap.String("status", ndpresponderStatus))
		return false
	}

	// 检查IPv6地址配置文件是否存在且非空
	ipv6ConfigCmd := "[ -f /usr/local/bin/docker_check_ipv6 ] && [ -s /usr/local/bin/docker_check_ipv6 ] && [ \"$(sed -e '/^[[:space:]]*$/d' /usr/local/bin/docker_check_ipv6)\" != \"\" ] && echo 'valid' || echo 'invalid'"
	ipv6ConfigOutput, err := d.sshClient.Execute(ipv6ConfigCmd)
	if err != nil || strings.TrimSpace(ipv6ConfigOutput) != "valid" {
		global.APP_LOG.Debug("IPv6网络检查: IPv6地址配置文件无效或不存在",
			zap.String("provider", d.config.Name))
		return false
	}

	global.APP_LOG.Debug("IPv6网络检查成功: 所有组件都可用",
		zap.String("provider", d.config.Name))
	return true
}

// ExecuteSSHCommand 执行SSH命令
func (d *DockerProvider) ExecuteSSHCommand(ctx context.Context, command string) (string, error) {
	if !d.connected || d.sshClient == nil {
		return "", fmt.Errorf("Docker provider not connected")
	}

	global.APP_LOG.Debug("执行SSH命令",
		zap.String("command", utils.TruncateString(command, 200)))

	output, err := d.sshClient.Execute(command)
	if err != nil {
		global.APP_LOG.Error("SSH命令执行失败",
			zap.String("command", utils.TruncateString(command, 200)),
			zap.String("output", utils.TruncateString(output, 500)),
			zap.Error(err))
		return "", fmt.Errorf("SSH command execution failed: %w", err)
	}

	return output, nil
}

// SSH 实现方法

func init() {
	provider.RegisterProvider("docker", NewDockerProvider)
}
