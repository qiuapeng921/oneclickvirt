# 任务系统 (Task System)

## 概述

这是一个基于 Go 语言开发的高性能异步任务管理系统，专为云主机管理平台设计。系统采用现代化的 **Channel 工作池 (Worker Pool)** 架构，提供强大的并发控制、任务调度和状态管理能力。

## 核心特性

- **Channel 工作池**: 基于 Go Channel 实现的高效并发控制
- **Provider 级别隔离**: 每个云服务商独立的工作池，避免相互影响
- **动态并发调整**: 支持运行时调整 Provider 的并发数配置
- **内存友好**: 无锁设计，自动垃圾回收，避免内存泄漏

## 支持的任务类型

- **create**: 创建云主机实例 (默认30分钟超时)
- **start**: 启动实例 (5分钟超时)
- **stop**: 停止实例 (5分钟超时)
- **restart**: 重启实例 (10分钟超时)
- **delete**: 删除实例 (10分钟超时)
- **reset**: 重置实例 (20分钟超时)
- **reset-password**: 重置密码 (5分钟超时)

## 任务状态管理

完整的任务生命周期管理，支持以下状态：

```
pending → running → completed
   ↓         ↓         ↑
cancelled   failed ←───┘
   ↓         ↓
timeout   cancelling
```

**状态说明：**

- `pending`: 任务已创建，等待执行
- `running`: 任务正在执行
- `completed`: 任务成功完成
- `failed`: 任务执行失败
- `cancelled`: 任务已取消
- `cancelling`: 任务取消中
- `timeout`: 任务执行超时

## 并发控制

### 🛡️ 并发模式

- **串行模式**: `AllowConcurrentTasks = false` (默认)
- **并发模式**: `AllowConcurrentTasks = true` + `MaxConcurrentTasks` 配置
- **队列缓冲**: 支持任务排队，避免拥塞
- **超时保护**: 任务级别和系统级别的超时机制

## 核心组件

### TaskService

主要的任务管理服务，提供：

- 任务创建、启动、取消
- 工作池管理
- 状态查询和监控
- 优雅关闭

**核心结构：**

```go
type TaskService struct {
    dbService       *database.DatabaseService
    runningContexts map[uint]*TaskContext
    contextMutex    sync.RWMutex
    providerPools   map[uint]*ProviderWorkerPool
    poolMutex       sync.RWMutex
    shutdown        chan struct{}
    wg              sync.WaitGroup
    ctx             context.Context
    cancel          context.CancelFunc
}
```

### ProviderWorkerPool

Provider 专用工作池，特性：

- 独立的任务队列
- 可配置的工作者数量
- 上下文取消支持
- 自动负载均衡

**核心结构：**

```go
type ProviderWorkerPool struct {
    ProviderID  uint
    TaskQueue   chan TaskRequest
    WorkerCount int
    Ctx         context.Context
    Cancel      context.CancelFunc
    TaskService *TaskService
}
```

### TaskStateManager

统一的任务状态管理器：

- 跨表状态同步
- 事务安全更新
- 状态流转验证
- 错误处理

## 使用示例

### 创建任务

```go
taskService := task.GetTaskService()
task, err := taskService.CreateTask(
    userID,      // 用户ID
    &providerID, // Provider ID
    &instanceID, // 实例ID
    "create",    // 任务类型
    taskData,    // 任务数据 (JSON)
    1800,        // 超时时间(秒)
)
```

### 启动任务

```go
err := taskService.StartTask(taskID)
```

### 查询任务状态

```go
tasks, total, err := taskService.GetAdminTasks(request)
```

### 取消任务

```go
err := taskService.CancelTask(taskID, userID)
```

## 配置参数

### Provider 级别配置

```yaml
allowConcurrentTasks: true    # 是否允许并发
maxConcurrentTasks: 3         # 最大并发数
taskPollInterval: 60          # 轮询间隔(秒)
enableTaskPolling: true       # 是否启用轮询
```

### 系统级别配置

- 默认超时时间: 30分钟
- 队列缓冲大小: 并发数 × 2
- 取消监听间隔: 1秒
- 优雅关闭等待: 5秒

## 监控指标

### 任务统计

- 总任务数
- 各状态任务数量
- Provider 任务分布
- 执行时间统计

### 性能指标

- 队列长度
- 工作者利用率
- 平均响应时间
- 错误率统计

## 最佳实践

### 1. 并发配置

```go
// 计算密集型任务建议较低并发
maxConcurrentTasks: 1-2

// I/O 密集型任务可以较高并发
maxConcurrentTasks: 3-5
```

### 2. 错误处理

- 设置合理的超时时间
- 实现重试机制
- 记录详细的错误日志
- 监控异常任务

### 3. 资源管理

- 定期清理超时任务
- 监控内存使用
- 合理配置队列大小
- 及时释放资源

## 技术架构

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Task API      │───▶│   TaskService    │───▶│ ProviderPool    │
│   (HTTP/gRPC)   │    │   (Singleton)    │    │ (Per Provider)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                       ┌────────▼────────┐    ┌─────────▼─────────┐
                       │ TaskStateManager│    │   Worker Pool     │
                       │ (Unified State) │    │ (Channel Based)   │
                       └─────────────────┘    └───────────────────┘
                                │                        │
                       ┌────────▼────────┐    ┌─────────▼─────────┐
                       │    Database     │    │  Task Execution   │
                       │   (GORM/MySQL)  │    │ (Provider APIs)   │
                       └─────────────────┘    └───────────────────┘
```

## API 接口

### GetTaskService

获取任务服务单例实例。

```go
func GetTaskService() *TaskService
```

### CreateTask

创建新任务。

```go
func (s *TaskService) CreateTask(
    userID uint,
    providerID *uint,
    instanceID *uint,
    taskType string,
    taskData string,
    timeoutDuration int,
) (*adminModel.Task, error)
```

**参数说明：**

- `userID`: 用户ID
- `providerID`: Provider ID（可选）
- `instanceID`: 实例ID（可选）
- `taskType`: 任务类型
- `taskData`: 任务数据（JSON格式）
- `timeoutDuration`: 超时时间（秒），0表示使用默认值

### StartTask

启动任务执行。

```go
func (s *TaskService) StartTask(taskID uint) error
```

### CancelTask

用户取消任务。

```go
func (s *TaskService) CancelTask(taskID uint, userID uint) error
```

### CancelTaskByAdmin

管理员取消任务。

```go
func (s *TaskService) CancelTaskByAdmin(taskID uint, reason string) error
```

### GetUserTasks

获取用户任务列表。

```go
func (s *TaskService) GetUserTasks(
    userID uint,
    req userModel.UserTasksRequest,
) ([]userModel.TaskResponse, int64, error)
```

### GetAdminTasks

获取管理员任务列表。

```go
func (s *TaskService) GetAdminTasks(
    req adminModel.AdminTaskListRequest,
) ([]adminModel.AdminTaskResponse, int64, error)
```

### Shutdown

优雅关闭任务服务。

```go
func (s *TaskService) Shutdown()
```

## 错误处理

任务执行过程中的错误会被记录到任务的 `error_message` 字段，并自动更新任务状态为 `failed`。

**常见错误场景：**

- Provider连接失败
- 实例操作超时
- 资源不足
- 配置错误
- 网络异常

## 超时机制

### 任务级别超时

每个任务类型都有默认的超时时间，也可以在创建任务时指定自定义超时时间。

### 上下文超时

使用 Context 实现超时控制，超时后会自动取消任务执行并清理资源。

### 取消监听

后台监听任务取消信号，支持用户主动取消正在执行的任务。

## 资源管理

### 自动清理

- 任务完成后自动清理运行时上下文
- 释放 Provider 资源计数
- 清理临时数据

### 启动时恢复

服务启动时自动将所有 `running` 状态的任务标记为 `failed`，避免状态不一致。

## 日志记录

任务服务使用结构化日志记录所有关键操作：

- 任务创建/启动/完成
- 状态变更
- 错误信息
- 性能指标

## 性能考虑

- Channel 工作池无锁设计，减少锁竞争
- Provider 级别隔离，提高并发性能
- 自动垃圾回收，避免内存泄漏
- 任务批量查询优化数据库访问

## 架构设计

```
server/service/task/
├── service.go          # 任务服务主实现
└── state_manager.go    # 任务状态管理器
```
