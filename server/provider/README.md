# Provider Package

## 概述

Provider Package 是 OneClickVirt 项目的核心组件，提供了对多种虚拟化平台的统一管理接口。通过抽象化的设计，支持 Docker、LXD、Incus 和 Proxmox 等主流虚拟化技术。

## 架构设计

```
server/provider/
├── provider.go          # Provider 接口定义和公共结构
├── common_stub.go       # 通用功能存根实现
├── docker_stub.go       # Docker 特定功能存根
├── docker/              # Docker Provider 实现
├── lxd/                 # LXD Provider 实现  
├── incus/               # Incus Provider 实现
├── proxmox/             # Proxmox Provider 实现
└── health/              # 统一健康检查系统
```

## 核心接口

### Provider 接口
所有虚拟化平台都实现统一的 Provider 接口：

```go
type Provider interface {
    // 基础信息
    GetType() string
    GetName() string
    GetSupportedInstanceTypes() []string
    
    // 连接管理
    Connect(ctx context.Context, config NodeConfig) error
    Disconnect() error
    CheckHealth(ctx context.Context) (string, string, error)
    
    // 实例管理
    CreateInstance(ctx context.Context, instance *Instance) error
    StartInstance(ctx context.Context, instanceID string) error
    StopInstance(ctx context.Context, instanceID string) error
    RestartInstance(ctx context.Context, instanceID string) error
    DeleteInstance(ctx context.Context, instanceID string) error
    ListInstances(ctx context.Context) ([]Instance, error)
    GetInstance(ctx context.Context, instanceID string) (*Instance, error)
    
    // 镜像管理
    ListImages(ctx context.Context) ([]Image, error)
    PullImage(ctx context.Context, imageName string) error
    DeleteImage(ctx context.Context, imageID string) error
    
    // 密码管理
    SetInstancePassword(ctx context.Context, instanceID, password string) error
    ResetInstancePassword(ctx context.Context, instanceID string) (string, error)
}
```

## 支持的平台

### [Docker Provider](./docker/README.md)
- **类型**: 容器化平台
- **实例类型**: container
- **管理方式**: SSH
- **适用场景**: 轻量级容器部署

### [LXD Provider](./lxd/README.md)
- **类型**: 系统容器和虚拟机
- **实例类型**: container, vm
- **管理方式**: API, SSH
- **适用场景**: 高性能容器和虚拟机

### [Incus Provider](./incus/README.md)
- **类型**: 系统容器和虚拟机（LXD 分支）
- **实例类型**: container, vm
- **管理方式**: API, SSH
- **适用场景**: LXD 的社区驱动版本

### [Proxmox Provider](./proxmox/README.md)
- **类型**: 企业级虚拟化平台
- **实例类型**: container, vm
- **管理方式**: API, SSH
- **适用场景**: 企业级虚拟化部署

## 健康检查系统

统一的[健康检查系统](./health/README.md)提供：
- SSH 连接状态检查
- API 服务状态检查
- 平台特定的服务状态检查
- 详细的健康报告

## 使用方式

### 1. 创建 Provider 实例
```go
import "oneclickvirt/server/provider"

// 根据类型创建对应的 Provider
var p provider.Provider
switch providerType {
case "docker":
    p = docker.NewDockerProvider()
case "lxd":
    p = lxd.NewLXDProvider()
case "incus":
    p = incus.NewIncusProvider()
case "proxmox":
    p = proxmox.NewProxmoxProvider()
}
```

### 2. 配置连接参数
```go
config := provider.NodeConfig{
    Name:     "node-name",
    Host:     "192.168.1.100",
    Port:     22,
    Username: "root",
    Password: "password",
    Type:     "docker", // docker, lxd, incus, proxmox
}
```

### 3. 连接和使用
```go
// 连接到节点
err := p.Connect(ctx, config)
if err != nil {
    return err
}
defer p.Disconnect()

// 检查健康状态
sshStatus, apiStatus, err := p.CheckHealth(ctx)

// 创建实例
instance := &provider.Instance{
    Name:         "test-instance",
    Image:        "ubuntu:20.04",
    CPU:          2,
    Memory:       2048,
    Storage:      20,
    InstanceType: "container",
}

err = p.CreateInstance(ctx, instance)
```

## 扩展新平台

要添加对新虚拟化平台的支持：

1. **实现 Provider 接口**
   ```go
   type NewProvider struct {
       // 平台特定的字段
   }
   
   func (n *NewProvider) GetType() string {
       return "new-platform"
   }
   
   // 实现其他接口方法...
   ```

2. **创建子包目录**
   ```
   provider/new-platform/
   ├── README.md
   ├── new-platform.go
   ├── api.go
   ├── ssh.go
   ├── instance.go
   ├── image.go
   └── password.go
   ```

3. **实现健康检查**
   ```go
   // 在 health/ 包中添加对应的健康检查器
   type NewPlatformHealthChecker struct {
       // 实现 HealthChecker 接口
   }
   ```

4. **注册 Provider**
   ```go
   // 在工厂函数中注册新的 Provider
   func NewProvider(providerType string) Provider {
       switch providerType {
       case "new-platform":
           return newplatform.NewProvider()
       // ...
       }
   }
   ```

## 配置文件格式

各平台的配置格式示例：

```yaml
providers:
  - name: "docker-node-1"
    type: "docker"
    host: "192.168.1.100"
    port: 22
    username: "root"
    password: "password"
    
  - name: "lxd-cluster-1"
    type: "lxd"
    host: "192.168.1.101"
    port: 22
    username: "root"
    password: "password"
    client_cert: "/path/to/client.crt"
    client_key: "/path/to/client.key"
    server_cert: "/path/to/server.crt"
    
  - name: "proxmox-node-1"
    type: "proxmox"
    host: "192.168.1.102"
    port: 22
    username: "root"
    password: "password"
    api_token: "user@pam!token=uuid"
    api_endpoint: "https://192.168.1.102:8006/api2/json"
```