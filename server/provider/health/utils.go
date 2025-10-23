package health

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"oneclickvirt/global"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// ProviderHealthChecker 为现有service层提供的健康检查工具
type ProviderHealthChecker struct {
	manager *HealthManager
	logger  *zap.Logger
}

// NewProviderHealthChecker 创建provider健康检查工具
func NewProviderHealthChecker(logger *zap.Logger) *ProviderHealthChecker {
	return &ProviderHealthChecker{
		manager: NewHealthManager(logger),
		logger:  logger,
	}
}

// ProviderAuthConfig 认证配置接口，避免循环导入
type ProviderAuthConfig interface {
	GetType() string
	GetCertificate() CertificateInfo
	GetToken() TokenInfo
}

// CertificateInfo 证书信息接口
type CertificateInfo interface {
	GetCertPath() string
	GetKeyPath() string
	GetCertContent() string
	GetKeyContent() string
}

// TokenInfo Token信息接口
type TokenInfo interface {
	GetTokenID() string
	GetTokenSecret() string
}

// CheckProviderHealthWithAuthConfig 根据认证配置执行健康检查
func (phc *ProviderHealthChecker) CheckProviderHealthWithAuthConfig(ctx context.Context, providerType, host, username, password string, port int, authConfig ProviderAuthConfig) (string, string, error) {
	config := HealthConfig{
		Host:          host,
		Port:          port,
		Username:      username,
		Password:      password,
		SSHEnabled:    true,
		APIEnabled:    true,
		SkipTLSVerify: true,
		Timeout:       30 * time.Second,
	}

	// 根据认证配置设置具体的认证信息
	switch providerType {
	case "lxd", "incus":
		cert := authConfig.GetCertificate()
		if cert != nil {
			config.APIPort = 8443
			config.APIScheme = "https"
			config.CertPath = cert.GetCertPath()
			config.KeyPath = cert.GetKeyPath()
			config.CertContent = cert.GetCertContent()
			config.KeyContent = cert.GetKeyContent()
		}
		config.ServiceChecks = []string{providerType}
	case "proxmox":
		token := authConfig.GetToken()
		if token != nil {
			config.APIPort = 8006
			config.APIScheme = "https"
			config.Token = token.GetTokenSecret()
			config.TokenID = token.GetTokenID()
		}
		config.ServiceChecks = []string{"pvestatd", "pvedaemon", "pveproxy"}
	case "docker":
		config.APIEnabled = false // docker默认不测API
		config.APIPort = 2375
		config.APIScheme = "http"
		config.ServiceChecks = []string{"docker"}
	}

	checker, err := phc.manager.CreateChecker(ProviderType(providerType), config)
	if err != nil {
		return "offline", "offline", fmt.Errorf("failed to create health checker: %w", err)
	}

	result, err := checker.CheckHealth(ctx)
	if err != nil {
		return "offline", "offline", err
	}

	// 确保释放资源
	switch c := checker.(type) {
	case *DockerHealthChecker:
		c.Close()
	case *LXDHealthChecker:
		c.Close()
	case *IncusHealthChecker:
		c.Close()
	case *ProxmoxHealthChecker:
		c.Close()
	}

	sshStatus := "unknown"
	apiStatus := "unknown"
	if result.SSHStatus != "" {
		sshStatus = result.SSHStatus
	}
	if result.APIStatus != "" {
		apiStatus = result.APIStatus
	}
	return sshStatus, apiStatus, nil
}

// CheckProviderHealthWithAuthConfig 根据认证配置执行健康检查

// CheckProviderHealthFromConfig 根据provider配置信息执行健康检查
func (phc *ProviderHealthChecker) CheckProviderHealthFromConfig(ctx context.Context, providerType, host, username, password string, port int) (string, string, error) {
	// 创建健康检查配置
	config := HealthConfig{
		Host:          host,
		Port:          port,
		Username:      username,
		Password:      password,
		SSHEnabled:    true,
		APIEnabled:    true,
		SkipTLSVerify: true, // 默认跳过TLS验证
		Timeout:       30 * time.Second,
	}
	switch providerType {
	case "docker":
		config.APIEnabled = false // docker默认不测API
		config.APIPort = 2375
		config.APIScheme = "http"
		config.ServiceChecks = []string{"docker"}
	case "lxd":
		config.APIPort = 8443
		config.APIScheme = "https"
		config.ServiceChecks = []string{"lxd"}
	case "incus":
		config.APIPort = 8443
		config.APIScheme = "https"
		config.ServiceChecks = []string{"incus"}
	case "proxmox":
		config.APIPort = 8006
		config.APIScheme = "https"
		config.ServiceChecks = []string{"pvestatd", "pvedaemon", "pveproxy"}
	}
	checker, err := phc.manager.CreateChecker(ProviderType(providerType), config)
	if err != nil {
		return "offline", "offline", fmt.Errorf("failed to create health checker: %w", err)
	}
	result, err := checker.CheckHealth(ctx)
	if err != nil {
		return "offline", "offline", err
	}
	switch c := checker.(type) {
	case *DockerHealthChecker:
		c.Close()
	case *LXDHealthChecker:
		c.Close()
	case *IncusHealthChecker:
		c.Close()
	case *ProxmoxHealthChecker:
		c.Close()
	}
	sshStatus := "unknown"
	apiStatus := "unknown"
	if result.SSHStatus != "" {
		sshStatus = result.SSHStatus
	}
	if result.APIStatus != "" {
		apiStatus = result.APIStatus
	}
	return sshStatus, apiStatus, nil
}

// CheckSSHConnection 单独检查SSH连接
func (phc *ProviderHealthChecker) CheckSSHConnection(ctx context.Context, host, username, password string, port int) error {
	config := HealthConfig{
		Host:       host,
		Port:       port,
		Username:   username,
		Password:   password,
		SSHEnabled: true,
		APIEnabled: false,
		Timeout:    30 * time.Second,
	}
	checker := NewDockerHealthChecker(config, phc.logger)
	defer checker.Close()
	result, err := checker.CheckHealth(ctx)
	if err != nil {
		return err
	}
	if result.SSHStatus == "offline" {
		return fmt.Errorf("SSH connection failed")
	}
	return nil
}

// CheckAPIConnection 单独检查API连接
func (phc *ProviderHealthChecker) CheckAPIConnection(ctx context.Context, providerType, host string, port int, token, tokenID string) error {
	config := HealthConfig{
		Host:          host,
		Port:          22, // 这里仍然使用默认值，因为API连接不需要SSH端口
		SSHEnabled:    false,
		APIEnabled:    true,
		APIPort:       port,
		SkipTLSVerify: true, // 默认跳过TLS验证
		Token:         token,
		TokenID:       tokenID,
		Timeout:       30 * time.Second,
	}
	switch providerType {
	case "docker":
		config.APIScheme = "http"
	case "lxd", "incus":
		config.APIScheme = "https"
	case "proxmox":
		config.APIScheme = "https"
	default:
		return fmt.Errorf("unsupported provider type: %s", providerType)
	}
	checker, err := phc.manager.CreateChecker(ProviderType(providerType), config)
	if err != nil {
		return fmt.Errorf("failed to create health checker: %w", err)
	}
	defer func() {
		switch c := checker.(type) {
		case *DockerHealthChecker:
			c.Close()
		case *LXDHealthChecker:
			c.Close()
		case *IncusHealthChecker:
			c.Close()
		case *ProxmoxHealthChecker:
			c.Close()
		}
	}()
	result, err := checker.CheckHealth(ctx)
	if err != nil {
		return err
	}
	if result.APIStatus == "offline" {
		// 尝试从结果中获取更详细的错误信息
		if len(result.Details) > 0 {
			if apiDetail, exists := result.Details[string(CheckTypeAPI)]; exists {
				if checkResult, ok := apiDetail.(CheckResult); ok && checkResult.Error != "" {
					return fmt.Errorf("API connection failed: %s", checkResult.Error)
				}
			}
		}
		// 如果找不到详细错误信息，检查 Errors 列表
		if len(result.Errors) > 0 {
			for _, errMsg := range result.Errors {
				if strings.Contains(errMsg, "api") || strings.Contains(errMsg, "API") {
					return fmt.Errorf("API connection failed: %s", errMsg)
				}
			}
			// 如果没有API相关错误，返回第一个错误
			return fmt.Errorf("API connection failed: %s", result.Errors[0])
		}
		return fmt.Errorf("API connection failed")
	}
	return nil
}

// GetSystemResourceInfo 通过SSH获取系统资源信息
func (phc *ProviderHealthChecker) GetSystemResourceInfo(ctx context.Context, host, username, password string, port int) (*ResourceInfo, error) {
	return phc.GetSystemResourceInfoWithKey(ctx, host, username, password, "", port)
}

// GetSystemResourceInfoWithKey 通过SSH获取系统资源信息（支持SSH密钥）
func (phc *ProviderHealthChecker) GetSystemResourceInfoWithKey(ctx context.Context, host, username, password, privateKey string, port int) (*ResourceInfo, error) {
	// 构建认证方法：优先使用SSH密钥，否则使用密码
	var authMethods []ssh.AuthMethod

	// 如果提供了SSH私钥，优先使用密钥认证
	if privateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
			if phc.logger != nil {
				phc.logger.Debug("使用SSH密钥认证获取资源信息", zap.String("host", host))
			}
		} else if phc.logger != nil {
			phc.logger.Warn("SSH私钥解析失败，将尝试使用密码认证",
				zap.String("host", host),
				zap.Error(err))
		}
	}

	// 如果没有密钥或密钥解析失败，且提供了密码，使用密码认证
	if len(authMethods) == 0 && password != "" {
		authMethods = append(authMethods, ssh.Password(password))
		if phc.logger != nil {
			phc.logger.Debug("使用SSH密码认证获取资源信息", zap.String("host", host))
		}
	}

	// 如果既没有密钥也没有密码，返回错误
	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method available: neither SSH key nor password provided")
	}

	config := &ssh.ClientConfig{
		User:            username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	// 连接SSH
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %w", err)
	}
	defer client.Close()

	resourceInfo := &ResourceInfo{}

	// 获取CPU核心数
	cpuCores, err := phc.executeSSHCommand(client, "nproc")
	if err == nil {
		if cores, parseErr := strconv.Atoi(strings.TrimSpace(cpuCores)); parseErr == nil {
			resourceInfo.CPUCores = cores
		}
	}

	// 获取内存信息（单位转换为MB）
	memInfo, err := phc.executeSSHCommand(client, "cat /proc/meminfo")
	if err == nil {
		resourceInfo.MemoryTotal = phc.parseMemoryValue(memInfo, "MemTotal")
		resourceInfo.SwapTotal = phc.parseMemoryValue(memInfo, "SwapTotal")
	}

	// 获取磁盘信息（根目录）- 使用更通用的方式
	diskInfo, err := phc.executeSSHCommand(client, "df -h / | tail -1")
	if err == nil {
		if phc.logger != nil {
			phc.logger.Debug("df -h命令输出", zap.String("output", diskInfo))
		}
		// 解析df输出，格式：Filesystem Size Used Avail Use% Mounted on
		// 示例：/dev/sda1        25G   17G  7.2G  70% /
		fields := strings.Fields(strings.TrimSpace(diskInfo))
		if len(fields) >= 4 {
			// 第二个字段(index 1)是总空间Size，第四个字段(index 3)是可用空间Avail
			if total := phc.parseDiskSize(fields[1]); total > 0 {
				resourceInfo.DiskTotal = total
			}
			if free := phc.parseDiskSize(fields[3]); free > 0 {
				resourceInfo.DiskFree = free // 现在parseDiskSize返回MB，直接使用
			}
		}
	} else if phc.logger != nil {
		phc.logger.Warn("df -h命令失败", zap.Error(err))
	}

	// 如果df -h解析失败，尝试使用statvfs系统调用的替代方案
	if resourceInfo.DiskTotal == 0 {
		// 尝试使用du和df的组合来获取更准确的信息
		statInfo, statErr := phc.executeSSHCommand(client, "stat -f / 2>/dev/null || df / | tail -1")
		if statErr == nil && statInfo != "" {
			if phc.logger != nil {
				phc.logger.Debug("备用磁盘信息命令输出", zap.String("output", statInfo))
			}
			// 如果是stat -f的输出，会包含更详细的文件系统信息
			// 如果是df的输出，格式类似但可能没有单位后缀
			if strings.Contains(statInfo, "/") {
				fields := strings.Fields(strings.TrimSpace(statInfo))
				if len(fields) >= 4 {
					// 尝试解析第二个和第四个字段，如果没有单位则假设是KB
					total := phc.parseDiskSizeWithDefault(fields[1], "K")
					if total > 0 {
						resourceInfo.DiskTotal = total
					}
					free := phc.parseDiskSizeWithDefault(fields[3], "K")
					if free > 0 {
						resourceInfo.DiskFree = free // 现在parseDiskSizeWithDefault返回MB，直接使用
					}
				}
			}
		}
	}

	now := time.Now()
	resourceInfo.Synced = true
	resourceInfo.SyncedAt = &now

	if phc.logger != nil {
		phc.logger.Info("系统资源信息获取成功",
			zap.String("host", host),
			zap.Int("cpu_cores", resourceInfo.CPUCores),
			zap.Int64("memory_total_mb", resourceInfo.MemoryTotal),
			zap.Int64("swap_total_mb", resourceInfo.SwapTotal),
			zap.Int64("disk_total_mb", resourceInfo.DiskTotal),
			zap.Int64("disk_free_mb", resourceInfo.DiskFree))
	}

	return resourceInfo, nil
}

// executeSSHCommand 执行SSH命令
func (phc *ProviderHealthChecker) executeSSHCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	// 请求PTY以模拟交互式登录shell，确保加载完整的环境变量
	err = session.RequestPty("xterm", 80, 40, ssh.TerminalModes{
		ssh.ECHO:          0,     // 禁用回显
		ssh.TTY_OP_ISPEED: 14400, // 输入速度
		ssh.TTY_OP_OSPEED: 14400, // 输出速度
	})
	if err != nil {
		return "", fmt.Errorf("failed to request PTY: %w", err)
	}

	// 设置环境变量来确保PATH正确加载，避免bash -l -c的转义问题
	envCommand := fmt.Sprintf("source /etc/profile 2>/dev/null || true; source ~/.bashrc 2>/dev/null || true; source ~/.bash_profile 2>/dev/null || true; export PATH=$PATH:/usr/local/bin:/snap/bin:/usr/sbin:/sbin; %s", command)

	output, err := session.Output(envCommand)
	if err != nil {
		// 记录执行失败的详细信息
		if global.APP_LOG != nil {
			global.APP_LOG.Debug("健康检查SSH命令执行失败",
				zap.String("original_command", command),
				zap.String("env_wrapped_command", envCommand),
				zap.Error(err),
				zap.String("output", string(output)))
		}
		return "", err
	}

	return string(output), nil
}

// parseMemoryValue 从/proc/meminfo解析内存值并转换为MB
func (phc *ProviderHealthChecker) parseMemoryValue(memInfo, field string) int64 {
	// 使用正则表达式解析，格式如：MemTotal:        8169348 kB
	pattern := fmt.Sprintf(`%s:\s*(\d+)\s*kB`, field)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(memInfo)

	if len(matches) >= 2 {
		if kb, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			return kb / 1024 // 转换为MB
		}
	}

	return 0
}

// parseDiskSize 解析磁盘大小字符串并转换为GB
// 支持格式：25G, 1.5T, 500M, 1024K, 228Gi, 10Ti等
func (phc *ProviderHealthChecker) parseDiskSize(sizeStr string) int64 {
	if sizeStr == "" {
		return 0
	}

	// 移除空格
	sizeStr = strings.TrimSpace(sizeStr)
	if len(sizeStr) == 0 {
		return 0
	}

	// 处理二进制单位（如Gi, Ti, Mi, Ki）和十进制单位（如G, T, M, K）
	var multiplier float64 = 1
	var numStr string

	// 检查是否是二进制单位（以i结尾）
	if strings.HasSuffix(sizeStr, "i") && len(sizeStr) >= 3 {
		unit := strings.ToUpper(string(sizeStr[len(sizeStr)-2]))
		numStr = sizeStr[:len(sizeStr)-2]

		switch unit {
		case "T":
			multiplier = 1024 * 1024 // TiB转MB
		case "G":
			multiplier = 1024 // GiB转MB
		case "M":
			multiplier = 1 // MiB近似等于MB
		case "K":
			multiplier = 1.0 / 1024 // KiB转MB
		default:
			return 0
		}
	} else if len(sizeStr) >= 2 {
		// 十进制单位（df -h的标准输出）
		unit := strings.ToUpper(string(sizeStr[len(sizeStr)-1]))
		numStr = sizeStr[:len(sizeStr)-1]

		switch unit {
		case "T":
			multiplier = 1024 * 1024 // TB转MB
		case "G":
			multiplier = 1024 // GB转MB
		case "M":
			multiplier = 1 // MB
		case "K":
			multiplier = 1.0 / 1024 // KB转MB
		default:
			// 如果没有单位，可能是纯数字，假设是字节
			numStr = sizeStr
			multiplier = 1.0 / (1024 * 1024) // 字节转MB
		}
	} else {
		// 纯数字，假设是字节
		numStr = sizeStr
		multiplier = 1.0 / (1024 * 1024)
	}

	// 解析数字部分
	size, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		if phc.logger != nil {
			phc.logger.Debug("解析磁盘大小失败",
				zap.String("input", sizeStr),
				zap.String("numStr", numStr),
				zap.Error(err))
		}
		return 0
	}

	result := int64(size * multiplier)
	if phc.logger != nil {
		phc.logger.Debug("磁盘大小解析结果",
			zap.String("input", sizeStr),
			zap.Float64("size", size),
			zap.Float64("multiplier", multiplier),
			zap.Int64("result_mb", result))
	}

	return result
}

// parseDiskSizeWithDefault 解析磁盘大小，如果没有单位则使用默认单位
func (phc *ProviderHealthChecker) parseDiskSizeWithDefault(sizeStr, defaultUnit string) int64 {
	if sizeStr == "" {
		return 0
	}

	// 移除空格
	sizeStr = strings.TrimSpace(sizeStr)
	if len(sizeStr) == 0 {
		return 0
	}

	// 处理二进制单位（如Gi, Ti, Mi, Ki）和十进制单位（如G, T, M, K）
	var multiplier float64 = 1
	var numStr string

	// 检查是否是二进制单位（以i结尾）
	if strings.HasSuffix(sizeStr, "i") && len(sizeStr) >= 3 {
		unit := strings.ToUpper(string(sizeStr[len(sizeStr)-2]))
		numStr = sizeStr[:len(sizeStr)-2]

		switch unit {
		case "T":
			multiplier = 1024 * 1024 // TiB转MB
		case "G":
			multiplier = 1024 // GiB转MB
		case "M":
			multiplier = 1 // MiB近似等于MB
		case "K":
			multiplier = 1.0 / 1024 // KiB转MB
		default:
			return 0
		}
	} else if len(sizeStr) >= 2 {
		// 十进制单位
		unit := strings.ToUpper(string(sizeStr[len(sizeStr)-1]))
		numStr = sizeStr[:len(sizeStr)-1]

		switch unit {
		case "T":
			multiplier = 1024 * 1024 // TB转MB
		case "G":
			multiplier = 1024 // GB转MB
		case "M":
			multiplier = 1 // MB
		case "K":
			multiplier = 1.0 / 1024 // KB转MB
		default:
			// 如果没有单位，可能是纯数字，假设是字节
			numStr = sizeStr
			multiplier = 1.0 / (1024 * 1024) // 字节转MB
		}
	} else {
		// 纯数字，假设是字节
		numStr = sizeStr
		multiplier = 1.0 / (1024 * 1024)
	}

	// 解析数字部分
	size, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		if phc.logger != nil {
			phc.logger.Debug("解析磁盘大小失败",
				zap.String("input", sizeStr),
				zap.String("numStr", numStr),
				zap.Error(err))
		}
		return 0
	}

	result := int64(size * multiplier)
	if phc.logger != nil {
		phc.logger.Debug("磁盘大小解析结果",
			zap.String("input", sizeStr),
			zap.Float64("size", size),
			zap.Float64("multiplier", multiplier),
			zap.Int64("result_mb", result))
	}

	return result
}
