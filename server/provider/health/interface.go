package health

import (
	"context"
	"time"
)

// HealthChecker 健康检测接口
type HealthChecker interface {
	// CheckHealth 执行健康检查
	CheckHealth(ctx context.Context) (*HealthResult, error)

	// GetHealthStatus 获取健康状态
	GetHealthStatus() HealthStatus

	// SetConfig 设置配置
	SetConfig(config HealthConfig)
}

// HealthStatus 健康状态枚举
type HealthStatus string

const (
	HealthStatusUnknown   HealthStatus = "unknown"
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusPartial   HealthStatus = "partial"
)

// HealthResult 健康检查结果
type HealthResult struct {
	Status        HealthStatus           `json:"status"`
	Timestamp     time.Time              `json:"timestamp"`
	Duration      time.Duration          `json:"duration"`
	SSHStatus     string                 `json:"ssh_status"`
	APIStatus     string                 `json:"api_status"`
	ServiceStatus string                 `json:"service_status"`
	Errors        []string               `json:"errors,omitempty"`
	Details       map[string]interface{} `json:"details,omitempty"`
	// 资源信息字段
	ResourceInfo *ResourceInfo `json:"resource_info,omitempty"`
}

// ResourceInfo 节点资源信息
type ResourceInfo struct {
	CPUCores    int        `json:"cpu_cores"`    // CPU核心数
	MemoryTotal int64      `json:"memory_total"` // 总内存（MB）
	SwapTotal   int64      `json:"swap_total"`   // 总交换空间（MB）
	DiskTotal   int64      `json:"disk_total"`   // 总磁盘空间（MB）
	DiskFree    int64      `json:"disk_free"`    // 可用磁盘空间（MB）
	Synced      bool       `json:"synced"`       // 是否已同步
	SyncedAt    *time.Time `json:"synced_at"`    // 同步时间
}

// HealthConfig 健康检查配置
type HealthConfig struct {
	// 基础连接配置
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`

	// API配置
	APIEnabled    bool   `json:"api_enabled"`
	APIPort       int    `json:"api_port"`
	APIScheme     string `json:"api_scheme"`      // http, https
	SkipTLSVerify bool   `json:"skip_tls_verify"` // 跳过TLS证书验证
	Token         string `json:"token"`
	TokenID       string `json:"token_id"`
	CertPath      string `json:"cert_path"`
	KeyPath       string `json:"key_path"`
	CertContent   string `json:"cert_content"` // 证书内容（优先于CertPath）
	KeyContent    string `json:"key_content"`  // 私钥内容（优先于KeyPath）

	// 检查配置
	Timeout        time.Duration `json:"timeout"`
	SSHEnabled     bool          `json:"ssh_enabled"`
	ServiceChecks  []string      `json:"service_checks"`  // 要检查的服务列表
	CustomCommands []string      `json:"custom_commands"` // 自定义检查命令
}

// CheckType 检查类型
type CheckType string

const (
	CheckTypeSSH     CheckType = "ssh"
	CheckTypeAPI     CheckType = "api"
	CheckTypeService CheckType = "service"
	CheckTypeCustom  CheckType = "custom"
)

// CheckResult 单个检查结果
type CheckResult struct {
	Type     CheckType              `json:"type"`
	Success  bool                   `json:"success"`
	Duration time.Duration          `json:"duration"`
	Error    string                 `json:"error,omitempty"`
	Details  map[string]interface{} `json:"details,omitempty"`
}
