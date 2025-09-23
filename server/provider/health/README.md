# Health Check System

## 概述

将所有provider的健康检查功能统一到 `server/provider/health` 包中，提供：

- 统一的健康检查接口
- 支持SSH和API两种检查方式
- 详细的健康检查结果
- 易于扩展的架构

## 架构

```
server/provider/health/
├── interface.go    # 健康检查接口定义
├── base.go        # 基础健康检查器实现
├── manager.go     # 健康检查管理器
├── factory.go     # 工厂函数和适配器
├── utils.go       # 工具函数
├── docker.go      # Docker健康检查器
├── lxd.go         # LXD健康检查器
├── incus.go       # Incus健康检查器
└── proxmox.go     # Proxmox健康检查器
```

## 主要组件

### HealthChecker 接口
- `CheckHealth(ctx context.Context) (*HealthResult, error)` - 执行健康检查
- `GetHealthStatus() HealthStatus` - 获取健康状态
- `SetConfig(config HealthConfig)` - 设置配置

### HealthResult 结构
```go
type HealthResult struct {
    Status        HealthStatus          `json:"status"`        // 总体状态
    Timestamp     time.Time             `json:"timestamp"`     // 检查时间
    Duration      time.Duration         `json:"duration"`      // 检查耗时
    SSHStatus     string                `json:"ssh_status"`    // SSH状态
    APIStatus     string                `json:"api_status"`    // API状态
    ServiceStatus string                `json:"service_status"` // 服务状态
    Errors        []string              `json:"errors,omitempty"` // 错误列表
    Details       map[string]interface{} `json:"details,omitempty"` // 详细信息
}
```

### 支持的Provider类型
- Docker (API: HTTP, SSH: 标准SSH)
- LXD (API: HTTPS + 证书, SSH: 标准SSH)
- Incus (API: HTTPS + 证书, SSH: 标准SSH)
- Proxmox (API: HTTPS + Token, SSH: 标准SSH)

## 使用方法

### 在Service层中使用
```go
import "oneclickvirt/server/provider/health"

// 创建健康检查工具
healthChecker := health.NewProviderHealthChecker(logger)

// 执行健康检查
sshStatus, apiStatus, err := healthChecker.CheckProviderHealthFromConfig(
    ctx, "docker", "192.168.1.100", "root", "password", 22)
```

### 在Provider中使用
```go
// 在Connect方法中初始化健康检查器
healthConfig := health.HealthConfig{
    Host:       config.Host,
    Port:       config.Port,
    Username:   config.Username,
    Password:   config.Password,
    APIEnabled: true,
    SSHEnabled: true,
    Timeout:    30 * time.Second,
}
p.healthChecker = health.NewDockerHealthChecker(healthConfig, logger)

// 执行健康检查
result, err := p.healthChecker.CheckHealth(ctx)
```