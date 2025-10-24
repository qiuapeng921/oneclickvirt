package instance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"oneclickvirt/service/database"
	"oneclickvirt/service/interfaces"
	"oneclickvirt/service/resources"
	"strings"
	"time"

	"oneclickvirt/global"
	"oneclickvirt/model/admin"
	adminModel "oneclickvirt/model/admin"
	providerModel "oneclickvirt/model/provider"
	userModel "oneclickvirt/model/user"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service 管理员实例管理服务
type Service struct {
	taskService interfaces.TaskServiceInterface
}

// NewService 创建实例管理服务
func NewService(taskService interfaces.TaskServiceInterface) *Service {
	return &Service{
		taskService: taskService,
	}
}

// GetInstanceByID 根据ID获取实例详情
func (s *Service) GetInstanceByID(instanceID uint) (*providerModel.Instance, error) {
	var instance providerModel.Instance

	if err := global.APP_DB.First(&instance, instanceID).Error; err != nil {
		global.APP_LOG.Error("获取实例详情失败", zap.Error(err), zap.Uint("instanceID", instanceID))
		return nil, err
	}

	return &instance, nil
}

// GetInstanceList 获取实例列表
func (s *Service) GetInstanceList(req admin.InstanceListRequest) ([]admin.InstanceManageResponse, int64, error) {
	var instances []providerModel.Instance
	var total int64

	// 管理员查看所有实例，不限制user_id
	query := global.APP_DB.Model(&providerModel.Instance{})

	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.ProviderName != "" {
		query = query.Where("provider LIKE ?", "%"+req.ProviderName+"%")
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.InstanceType != "" {
		query = query.Where("instance_type = ?", req.InstanceType)
	}
	// 如果指定了用户ID，则按用户筛选
	if req.UserID != 0 {
		query = query.Where("user_id = ?", req.UserID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&instances).Error; err != nil {
		return nil, 0, err
	}

	var instanceResponses []admin.InstanceManageResponse
	for _, instance := range instances {
		var userName, providerName string

		// 获取用户名
		if instance.UserID != 0 {
			var user userModel.User
			if err := global.APP_DB.First(&user, instance.UserID).Error; err == nil {
				userName = user.Username
			} else {
				userName = "未知用户"
			}
		} else {
			userName = "系统"
		}

		// 获取Provider名称
		if instance.Provider != "" {
			providerName = instance.Provider
		} else {
			providerName = "未知提供商"
		}

		// 获取SSH端口映射的公网端口
		var sshPort int
		var sshPortMapping providerModel.Port
		if err := global.APP_DB.Where("instance_id = ? AND is_ssh = true AND status = 'active'", instance.ID).First(&sshPortMapping).Error; err == nil {
			sshPort = sshPortMapping.HostPort // 使用映射的公网端口
		} else {
			sshPort = instance.SSHPort // fallback到默认值
		}

		// 获取纯净的公网IP（移除端口号）
		var cleanIPAddress string
		if instance.ProviderID > 0 {
			var providerInfo providerModel.Provider
			if err := global.APP_DB.Where("id = ?", instance.ProviderID).First(&providerInfo).Error; err == nil {
				endpoint := providerInfo.Endpoint
				if endpoint != "" {
					// 移除端口号部分，只保留IP
					if colonIndex := strings.LastIndex(endpoint, ":"); colonIndex > 0 {
						// 检查是否是IPv6地址
						if strings.Count(endpoint, ":") > 1 && !strings.HasPrefix(endpoint, "[") {
							// IPv6地址
							cleanIPAddress = endpoint
						} else {
							// IPv4地址，移除端口部分
							cleanIPAddress = endpoint[:colonIndex]
						}
					} else {
						cleanIPAddress = endpoint
					}
				}
			}
		}

		// 如果没有从Provider获取到IP，使用实例自身的IP地址字段
		if cleanIPAddress == "" && instance.PublicIP != "" {
			// 同样处理实例IP地址，移除可能的端口号
			endpoint := instance.PublicIP
			if colonIndex := strings.LastIndex(endpoint, ":"); colonIndex > 0 {
				if strings.Count(endpoint, ":") > 1 && !strings.HasPrefix(endpoint, "[") {
					cleanIPAddress = endpoint // IPv6
				} else {
					cleanIPAddress = endpoint[:colonIndex] // IPv4，移除端口
				}
			} else {
				cleanIPAddress = endpoint
			}
		}

		// 创建修改后的实例副本
		modifiedInstance := instance
		if sshPort > 0 {
			modifiedInstance.SSHPort = sshPort
		}
		if cleanIPAddress != "" {
			modifiedInstance.PublicIP = cleanIPAddress
		}

		instanceResponse := admin.InstanceManageResponse{
			Instance:     modifiedInstance,
			UserName:     userName,
			ProviderName: providerName,
			ProviderType: "",
			HealthStatus: "healthy",
		}

		// 如果关联了 ProviderID，尝试查找 Provider 的类型并填充
		if instance.ProviderID > 0 {
			var prov providerModel.Provider
			if err := global.APP_DB.Where("id = ?", instance.ProviderID).First(&prov).Error; err == nil {
				instanceResponse.ProviderType = prov.Type
			}
		}
		instanceResponses = append(instanceResponses, instanceResponse)
	}

	return instanceResponses, total, nil
}

// CreateInstance 创建实例
func (s *Service) CreateInstance(req admin.CreateInstanceRequest) error {
	// 使用新的配额验证服务，即使是管理员也需要检查用户配额
	quotaService := resources.NewQuotaService()

	// 构建资源请求
	quotaReq := resources.ResourceRequest{
		UserID:       req.UserID,
		CPU:          req.CPU,
		Memory:       req.Memory,
		Disk:         req.Disk,
		InstanceType: req.InstanceType,
	}

	// 验证用户配额（管理员创建也要遵守用户限制）
	quotaResult, err := quotaService.ValidateAdminInstanceCreation(quotaReq)
	if err != nil {
		return fmt.Errorf("配额验证失败: %v", err)
	}

	if !quotaResult.Allowed {
		return fmt.Errorf("无法为用户创建实例: %s", quotaResult.Reason)
	}

	// 检查提供商是否存在和冻结状态
	var provider providerModel.Provider
	if err := global.APP_DB.Where("name = ?", req.Provider).First(&provider).Error; err != nil {
		return fmt.Errorf("提供商不存在: %s", req.Provider)
	}

	// 检查提供商是否冻结
	if provider.IsFrozen {
		return fmt.Errorf("提供商 %s 已被冻结，无法创建实例", req.Provider)
	}

	// 检查提供商是否过期
	if provider.ExpiresAt != nil && provider.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("提供商 %s 已过期，无法创建实例", req.Provider)
	}

	// 设置实例到期时间，与Provider的到期时间同步
	var expiredAt time.Time
	if provider.ExpiresAt != nil {
		// 如果Provider有到期时间，使用Provider的到期时间
		expiredAt = *provider.ExpiresAt
	} else {
		// 如果Provider没有到期时间，默认为1年后
		expiredAt = time.Now().AddDate(1, 0, 0)
	}

	// 创建实例
	instance := providerModel.Instance{
		Name:         req.Name,
		Provider:     req.Provider,
		ProviderID:   provider.ID,
		Image:        req.Image,
		CPU:          req.CPU,
		Memory:       req.Memory,
		Disk:         req.Disk,
		InstanceType: req.InstanceType,
		UserID:       req.UserID,
		Status:       "creating",
		ExpiredAt:    expiredAt,
		PublicIP:     provider.Endpoint, // 设置公网IP为Provider的地址
	}

	// 初始化数据库服务
	dbService := database.GetDatabaseService()

	// 在单个事务中创建实例并更新配额
	return dbService.ExecuteTransaction(context.Background(), func(tx *gorm.DB) error {
		// 创建实例
		if err := tx.Create(&instance).Error; err != nil {
			return fmt.Errorf("创建实例失败: %v", err)
		}

		// 在同一事务中更新用户配额
		resourceUsage := resources.ResourceUsage{
			CPU:    req.CPU,
			Memory: req.Memory,
			Disk:   req.Disk,
		}

		if err := quotaService.UpdateUserQuotaAfterCreationWithTx(tx, req.UserID, resourceUsage); err != nil {
			return fmt.Errorf("更新用户配额失败: %v", err)
		}

		// 创建默认端口映射
		portMappingService := resources.PortMappingService{}
		if err := portMappingService.CreateDefaultPortMappings(instance.ID, provider.ID); err != nil {
			// 端口映射创建失败不应该阻止实例创建，只记录警告
			global.APP_LOG.Warn("创建默认端口映射失败",
				zap.Uint("instance_id", instance.ID),
				zap.Error(err))
		}

		return nil
	})
}

// UpdateInstance 更新实例
func (s *Service) UpdateInstance(req admin.UpdateInstanceRequest) error {
	var instance providerModel.Instance
	if err := global.APP_DB.First(&instance, req.ID).Error; err != nil {
		return err
	}

	instance.Name = req.Name
	instance.CPU = req.CPU
	instance.Memory = req.Memory
	instance.Disk = req.Disk
	instance.Status = req.Status

	dbService := database.GetDatabaseService()
	return dbService.ExecuteTransaction(context.Background(), func(tx *gorm.DB) error {
		return tx.Save(&instance).Error
	})
}

// DeleteInstance 删除实例 - 使用异步任务机制
func (s *Service) DeleteInstance(instanceID uint) error {
	// 获取实例信息
	var instance providerModel.Instance
	if err := global.APP_DB.First(&instance, instanceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("实例不存在")
		}
		return fmt.Errorf("获取实例信息失败: %v", err)
	}

	// 检查实例状态，避免重复删除
	if instance.Status == "deleting" {
		return fmt.Errorf("实例正在删除中")
	}

	// 检查是否已有进行中的删除任务
	var existingTask adminModel.Task
	if err := global.APP_DB.Where("instance_id = ? AND task_type = 'delete' AND status IN ('pending', 'running')", instance.ID).First(&existingTask).Error; err == nil {
		return fmt.Errorf("实例已有删除任务正在进行")
	}

	// 创建管理员删除任务
	taskData := map[string]interface{}{
		"instanceId":     instanceID,
		"providerId":     instance.ProviderID,
		"adminOperation": true, // 标记为管理员操作
	}

	taskDataJSON, err := json.Marshal(taskData)
	if err != nil {
		return fmt.Errorf("序列化任务数据失败: %v", err)
	}

	// 创建删除任务，设置为不可被用户取消
	task, err := s.taskService.CreateTask(instance.UserID, &instance.ProviderID, &instanceID, "delete", string(taskDataJSON), 1800)
	if err != nil {
		return fmt.Errorf("创建删除任务失败: %v", err)
	}

	// 标记任务为管理员操作，不允许用户取消
	if err := global.APP_DB.Model(task).Update("is_force_stoppable", false).Error; err != nil {
		global.APP_LOG.Warn("更新任务可取消状态失败", zap.Uint("taskId", task.ID), zap.Error(err))
	}

	// 更新实例状态为删除中
	if err := global.APP_DB.Model(&instance).Update("status", "deleting").Error; err != nil {
		global.APP_LOG.Warn("更新实例状态失败", zap.Uint("instanceId", instanceID), zap.Error(err))
	}

	global.APP_LOG.Info("管理员创建删除任务成功",
		zap.Uint("instanceId", instanceID),
		zap.String("instanceName", instance.Name),
		zap.Uint("taskId", task.ID))

	return nil
}

// InstanceAction 管理员执行实例操作
func (s *Service) InstanceAction(instanceID uint, req admin.InstanceActionRequest) error {
	// 获取实例信息
	var instance providerModel.Instance
	if err := global.APP_DB.First(&instance, instanceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("实例不存在")
		}
		return fmt.Errorf("获取实例信息失败: %v", err)
	}

	// 根据操作类型执行相应的操作
	switch req.Action {
	case "start", "stop", "restart", "reset":
		// 创建异步任务
		taskData := map[string]interface{}{
			"instanceId": instanceID,
			"providerId": instance.ProviderID,
		}

		// 将taskData序列化为JSON字符串
		taskDataJSON, err := json.Marshal(taskData)
		if err != nil {
			return fmt.Errorf("序列化任务数据失败: %v", err)
		}

		_, err = s.taskService.CreateTask(instance.UserID, &instance.ProviderID, &instanceID, req.Action, string(taskDataJSON), 1800)
		if err != nil {
			return fmt.Errorf("创建任务失败: %v", err)
		}

		// 更新实例状态
		statusMap := map[string]string{
			"start":   "starting",
			"stop":    "stopping",
			"restart": "restarting",
			"reset":   "resetting",
		}
		if newStatus, exists := statusMap[req.Action]; exists {
			instance.Status = newStatus
			if err := global.APP_DB.Save(&instance).Error; err != nil {
				return fmt.Errorf("更新实例状态失败: %v", err)
			}
		}

	case "delete":
		// 创建管理员删除任务（不允许用户取消）
		taskData := map[string]interface{}{
			"instanceId":     instanceID,
			"providerId":     instance.ProviderID,
			"adminOperation": true, // 标记为管理员操作
		}

		// 将taskData序列化为JSON字符串
		taskDataJSON, err := json.Marshal(taskData)
		if err != nil {
			return fmt.Errorf("序列化任务数据失败: %v", err)
		}

		// 创建管理员删除任务，设置为不可被用户取消
		task, err := s.taskService.CreateTask(instance.UserID, &instance.ProviderID, &instanceID, "delete", string(taskDataJSON), 1800)
		if err != nil {
			return fmt.Errorf("创建删除任务失败: %v", err)
		}

		// 标记任务为管理员操作，不允许用户取消
		if err := global.APP_DB.Model(task).Update("is_force_stoppable", false).Error; err != nil {
			return fmt.Errorf("更新任务权限失败: %v", err)
		}

		// 更新实例状态为删除中
		instance.Status = "deleting"
		if err := global.APP_DB.Save(&instance).Error; err != nil {
			return fmt.Errorf("更新实例状态失败: %v", err)
		}

	default:
		return errors.New("不支持的操作类型")
	}

	return nil
}

// ResetInstancePassword 管理员重置实例密码（异步任务）
func (s *Service) ResetInstancePassword(instanceID uint) (uint, error) {
	// 获取实例信息
	var instance providerModel.Instance
	if err := global.APP_DB.Where("id = ?", instanceID).First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("实例不存在")
		}
		return 0, err
	}

	// 检查实例状态
	if instance.Status != "running" {
		return 0, errors.New("只有运行中的实例才能重置密码")
	}

	// 检查是否已有进行中的密码重置任务
	var existingTask adminModel.Task
	if err := global.APP_DB.Where("instance_id = ? AND task_type = 'reset-password' AND status IN ('pending', 'running')", instance.ID).First(&existingTask).Error; err == nil {
		return 0, errors.New("该实例已有进行中的密码重置任务，请稍后重试")
	}

	// 创建任务数据
	taskData := map[string]interface{}{
		"instanceId": instance.ID,
		"providerId": instance.ProviderID,
	}

	taskDataJSON, err := json.Marshal(taskData)
	if err != nil {
		return 0, fmt.Errorf("序列化任务数据失败: %v", err)
	}

	// 管理员任务使用实例的用户ID
	task, err := s.taskService.CreateTask(instance.UserID, &instance.ProviderID, &instance.ID, "reset-password", string(taskDataJSON), 600) // 10分钟超时
	if err != nil {
		global.APP_LOG.Error("管理员创建密码重置任务失败",
			zap.Uint("instanceID", instanceID),
			zap.Error(err))
		return 0, fmt.Errorf("创建密码重置任务失败: %v", err)
	}

	global.APP_LOG.Info("管理员创建密码重置任务成功",
		zap.Uint("instanceID", instanceID),
		zap.Uint("taskID", task.ID),
		zap.String("instanceName", instance.Name),
		zap.Uint("userID", instance.UserID))

	return task.ID, nil
}

// GetInstanceNewPassword 管理员获取实例重置后的新密码（通过任务ID）
func (s *Service) GetInstanceNewPassword(instanceID uint, taskID uint) (string, int64, error) {
	// 获取实例信息
	var instance providerModel.Instance
	if err := global.APP_DB.Where("id = ?", instanceID).First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", 0, errors.New("实例不存在")
		}
		return "", 0, err
	}

	// 获取任务信息
	var task adminModel.Task
	if err := global.APP_DB.Where("id = ? AND instance_id = ? AND task_type = 'reset-password'", taskID, instanceID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", 0, errors.New("任务不存在")
		}
		return "", 0, err
	}

	// 检查任务状态
	if task.Status != "completed" {
		return "", 0, errors.New("任务尚未完成")
	}

	// 解析任务结果
	var taskResult adminModel.ResetPasswordTaskResult

	if err := json.Unmarshal([]byte(task.TaskData), &taskResult); err != nil {
		return "", 0, errors.New("解析任务结果失败")
	}

	if taskResult.NewPassword == "" {
		return "", 0, errors.New("任务结果中没有新密码")
	}

	return taskResult.NewPassword, taskResult.ResetTime, nil
}
