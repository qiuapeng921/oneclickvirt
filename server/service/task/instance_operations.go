package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"oneclickvirt/global"
	adminModel "oneclickvirt/model/admin"
	providerModel "oneclickvirt/model/provider"
	provider2 "oneclickvirt/service/provider"
	"oneclickvirt/service/traffic"
	"oneclickvirt/service/vnstat"
	"oneclickvirt/utils"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// executeStartInstanceTask 执行启动实例任务
func (s *TaskService) executeStartInstanceTask(ctx context.Context, task *adminModel.Task) error {
	// 初始化进度
	s.updateTaskProgress(task.ID, 10, "正在解析任务数据...")

	// 解析任务数据
	var taskReq adminModel.InstanceOperationTaskRequest
	if err := json.Unmarshal([]byte(task.TaskData), &taskReq); err != nil {
		return fmt.Errorf("解析任务数据失败: %v", err)
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 20, "正在获取实例信息...")

	// 获取实例信息
	var instance providerModel.Instance
	if err := global.APP_DB.First(&instance, taskReq.InstanceId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("实例不存在")
		}
		return fmt.Errorf("获取实例信息失败: %v", err)
	}

	// 验证实例所有权
	if instance.UserID != task.UserID {
		return fmt.Errorf("无权限操作此实例")
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 30, "正在获取Provider配置...")

	// 获取Provider配置
	var provider providerModel.Provider
	if err := global.APP_DB.First(&provider, instance.ProviderID).Error; err != nil {
		return fmt.Errorf("获取Provider配置失败: %v", err)
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 50, "正在启动实例...")

	// 调用Provider启动实例
	providerApiService := &provider2.ProviderApiService{}
	if err := providerApiService.StartInstanceByProviderID(ctx, provider.ID, instance.Name); err != nil {
		global.APP_LOG.Error("Provider启动实例失败",
			zap.Uint("taskId", task.ID),
			zap.String("instanceName", instance.Name),
			zap.String("provider", provider.Name),
			zap.Error(err))

		// 更新实例状态为启动失败
		global.APP_DB.Model(&instance).Update("status", "stopped")
		return fmt.Errorf("启动实例失败: %v", err)
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 70, "正在更新实例状态...")

	// 更新实例状态为运行中
	if err := global.APP_DB.Model(&instance).Update("status", "running").Error; err != nil {
		global.APP_LOG.Error("更新实例状态失败", zap.Error(err))
		return fmt.Errorf("更新实例状态失败: %v", err)
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 80, "正在初始化监控服务...")

	// 实例启动成功后，异步初始化vnStat监控和流量同步
	s.wg.Add(1)
	go func(instanceID uint, taskID uint) {
		defer s.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				global.APP_LOG.Error("启动实例后处理任务发生panic",
					zap.Uint("instanceId", instanceID),
					zap.Any("panic", r))
				stateManager := GetTaskStateManager()
				if err := stateManager.CompleteMainTask(taskID, true, "实例启动成功，但部分监控服务初始化失败", nil); err != nil {
					global.APP_LOG.Error("完成任务失败", zap.Uint("taskId", taskID), zap.Error(err))
				}
			}
		}()

		// 使用可取消的等待
		select {
		case <-time.After(30 * time.Second):
		case <-ctx.Done():
			global.APP_LOG.Info("启动实例后处理被取消",
				zap.Uint("instanceId", instanceID),
				zap.Uint("taskId", taskID))
			return
		}

		// 更新进度
		s.updateTaskProgress(taskID, 90, "正在初始化vnStat监控...")

		vnstatService := vnstat.NewService()
		vnstatSuccess := true
		if vnstatErr := vnstatService.InitializeVnStatForInstance(instanceID); vnstatErr != nil {
			global.APP_LOG.Warn("启动实例后初始化vnStat监控失败",
				zap.Uint("instanceId", instanceID),
				zap.Error(vnstatErr))
			vnstatSuccess = false
		} else {
			global.APP_LOG.Info("启动实例后vnStat监控初始化成功",
				zap.Uint("instanceId", instanceID))
		}

		// 更新进度
		s.updateTaskProgress(taskID, 95, "正在同步流量数据...")

		// 实例启动后同步流量数据
		syncTrigger := traffic.NewSyncTriggerService()
		syncTrigger.TriggerInstanceTrafficSync(instanceID, "实例启动后同步")

		// 标记任务完成
		completionMessage := "实例启动成功"
		if !vnstatSuccess {
			completionMessage = "实例启动成功，但vnStat监控初始化失败"
		}
		stateManager := GetTaskStateManager()
		if err := stateManager.CompleteMainTask(taskID, true, completionMessage, nil); err != nil {
			global.APP_LOG.Error("完成任务失败", zap.Uint("taskId", taskID), zap.Error(err))
		}

		global.APP_LOG.Info("启动实例后处理任务完成",
			zap.Uint("instanceId", instanceID),
			zap.Bool("vnstatSuccess", vnstatSuccess))
	}(instance.ID, task.ID)

	global.APP_LOG.Info("用户实例启动API调用成功",
		zap.Uint("taskId", task.ID),
		zap.Uint("instanceId", instance.ID),
		zap.String("instanceName", instance.Name),
		zap.Uint("userId", instance.UserID))

	return nil
}

// executeStopInstanceTask 执行停止实例任务
func (s *TaskService) executeStopInstanceTask(ctx context.Context, task *adminModel.Task) error {
	// 初始化进度
	s.updateTaskProgress(task.ID, 10, "正在解析任务数据...")

	// 解析任务数据
	var taskReq adminModel.InstanceOperationTaskRequest
	if err := json.Unmarshal([]byte(task.TaskData), &taskReq); err != nil {
		return fmt.Errorf("解析任务数据失败: %v", err)
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 20, "正在获取实例信息...")

	// 获取实例信息
	var instance providerModel.Instance
	if err := global.APP_DB.First(&instance, taskReq.InstanceId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("实例不存在")
		}
		return fmt.Errorf("获取实例信息失败: %v", err)
	}

	// 验证实例所有权
	if instance.UserID != task.UserID {
		return fmt.Errorf("无权限操作此实例")
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 30, "正在获取Provider配置...")

	// 获取Provider配置
	var provider providerModel.Provider
	if err := global.APP_DB.First(&provider, instance.ProviderID).Error; err != nil {
		return fmt.Errorf("获取Provider配置失败: %v", err)
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 50, "正在同步流量数据...")

	// 停止前同步流量数据
	syncTrigger := traffic.NewSyncTriggerService()
	syncTrigger.TriggerInstanceTrafficSync(instance.ID, "实例停止前同步")

	// 使用可取消的等待
	select {
	case <-time.After(3 * time.Second):
	case <-ctx.Done():
		return fmt.Errorf("任务已取消")
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 70, "正在停止实例...")

	// 调用Provider停止实例
	providerApiService := &provider2.ProviderApiService{}
	if err := providerApiService.StopInstanceByProviderID(ctx, provider.ID, instance.Name); err != nil {
		global.APP_LOG.Error("Provider停止实例失败",
			zap.Uint("taskId", task.ID),
			zap.String("instanceName", instance.Name),
			zap.String("provider", provider.Name),
			zap.Error(err))

		// 更新实例状态为停止失败
		global.APP_DB.Model(&instance).Update("status", "running")
		return fmt.Errorf("停止实例失败: %v", err)
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 90, "正在更新实例状态...")

	// 更新实例状态为已停止
	if err := global.APP_DB.Model(&instance).Update("status", "stopped").Error; err != nil {
		global.APP_LOG.Error("更新实例状态失败", zap.Error(err))
		return fmt.Errorf("更新实例状态失败: %v", err)
	}

	// 标记任务完成
	stateManager := GetTaskStateManager()
	if err := stateManager.CompleteMainTask(task.ID, true, "实例停止成功", nil); err != nil {
		global.APP_LOG.Error("完成任务失败", zap.Uint("taskId", task.ID), zap.Error(err))
	}

	global.APP_LOG.Info("用户实例停止成功",
		zap.Uint("taskId", task.ID),
		zap.Uint("instanceId", instance.ID),
		zap.String("instanceName", instance.Name),
		zap.Uint("userId", instance.UserID))

	return nil
}

// executeRestartInstanceTask 执行重启实例任务
func (s *TaskService) executeRestartInstanceTask(ctx context.Context, task *adminModel.Task) error {
	// 初始化进度
	s.updateTaskProgress(task.ID, 10, "正在解析任务数据...")

	// 解析任务数据
	var taskReq adminModel.InstanceOperationTaskRequest
	if err := json.Unmarshal([]byte(task.TaskData), &taskReq); err != nil {
		return fmt.Errorf("解析任务数据失败: %v", err)
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 20, "正在获取实例信息...")

	// 获取实例信息
	var instance providerModel.Instance
	if err := global.APP_DB.First(&instance, taskReq.InstanceId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("实例不存在")
		}
		return fmt.Errorf("获取实例信息失败: %v", err)
	}

	// 验证实例所有权
	if instance.UserID != task.UserID {
		return fmt.Errorf("无权限操作此实例")
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 30, "正在获取Provider配置...")

	// 获取Provider配置
	var provider providerModel.Provider
	if err := global.APP_DB.First(&provider, instance.ProviderID).Error; err != nil {
		return fmt.Errorf("获取Provider配置失败: %v", err)
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 40, "正在同步流量数据...")

	// 重启前同步流量数据
	syncTrigger := traffic.NewSyncTriggerService()
	syncTrigger.TriggerInstanceTrafficSync(instance.ID, "实例重启前同步")

	// 使用可取消的等待
	select {
	case <-time.After(3 * time.Second):
	case <-ctx.Done():
		return fmt.Errorf("任务已取消")
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 60, "正在重启实例...")

	// 调用Provider重启实例
	providerApiService := &provider2.ProviderApiService{}
	if err := providerApiService.RestartInstanceByProviderID(ctx, provider.ID, instance.Name); err != nil {
		global.APP_LOG.Error("Provider重启实例失败",
			zap.Uint("taskId", task.ID),
			zap.String("instanceName", instance.Name),
			zap.String("provider", provider.Name),
			zap.Error(err))

		// 更新实例状态为重启失败
		global.APP_DB.Model(&instance).Update("status", "running")
		return fmt.Errorf("重启实例失败: %v", err)
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 80, "正在更新实例状态...")

	// 更新实例状态为运行中
	if err := global.APP_DB.Model(&instance).Update("status", "running").Error; err != nil {
		global.APP_LOG.Error("更新实例状态失败", zap.Error(err))
		return fmt.Errorf("更新实例状态失败: %v", err)
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 85, "正在重新初始化监控服务...")

	// 实例重启成功后，异步重新初始化vnStat监控
	s.wg.Add(1)
	go func(instanceID uint, taskID uint) {
		defer s.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				global.APP_LOG.Error("重启实例后处理任务发生panic",
					zap.Uint("instanceId", instanceID),
					zap.Any("panic", r))
				stateManager := GetTaskStateManager()
				if err := stateManager.CompleteMainTask(taskID, true, "实例重启成功，但部分监控服务初始化失败", nil); err != nil {
					global.APP_LOG.Error("完成任务失败", zap.Uint("taskId", taskID), zap.Error(err))
				}
			}
		}()

		// 使用可取消的等待
		select {
		case <-time.After(30 * time.Second):
		case <-ctx.Done():
			global.APP_LOG.Info("重启实例后处理被取消",
				zap.Uint("instanceId", instanceID),
				zap.Uint("taskId", taskID))
			return
		}

		// 更新进度
		s.updateTaskProgress(taskID, 90, "正在重新初始化vnStat监控...")

		vnstatService := vnstat.NewService()
		vnstatSuccess := true
		if vnstatErr := vnstatService.InitializeVnStatForInstance(instanceID); vnstatErr != nil {
			global.APP_LOG.Warn("重启实例后重新初始化vnStat监控失败",
				zap.Uint("instanceId", instanceID),
				zap.Error(vnstatErr))
			vnstatSuccess = false
		} else {
			global.APP_LOG.Info("重启实例后vnStat监控重新初始化成功",
				zap.Uint("instanceId", instanceID))
		}

		// 更新进度
		s.updateTaskProgress(taskID, 95, "正在同步流量数据...")

		// 重启后同步流量数据
		syncTrigger := traffic.NewSyncTriggerService()
		syncTrigger.TriggerInstanceTrafficSync(instanceID, "实例重启后同步")

		// 标记任务完成
		completionMessage := "实例重启成功"
		if !vnstatSuccess {
			completionMessage = "实例重启成功，但vnStat监控重新初始化失败"
		}
		stateManager := GetTaskStateManager()
		if err := stateManager.CompleteMainTask(taskID, true, completionMessage, nil); err != nil {
			global.APP_LOG.Error("完成任务失败", zap.Uint("taskId", taskID), zap.Error(err))
		}

		global.APP_LOG.Info("重启实例后处理任务完成",
			zap.Uint("instanceId", instanceID),
			zap.Bool("vnstatSuccess", vnstatSuccess))
	}(instance.ID, task.ID)

	global.APP_LOG.Info("用户实例重启API调用成功",
		zap.Uint("taskId", task.ID),
		zap.Uint("instanceId", instance.ID),
		zap.String("instanceName", instance.Name),
		zap.Uint("userId", instance.UserID))

	return nil
}

// executeResetPasswordTask 执行重置实例密码任务
func (s *TaskService) executeResetPasswordTask(ctx context.Context, task *adminModel.Task) error {
	// 初始化进度
	s.updateTaskProgress(task.ID, 10, "正在解析任务数据...")

	// 解析任务数据
	var taskReq adminModel.ResetPasswordTaskRequest
	if err := json.Unmarshal([]byte(task.TaskData), &taskReq); err != nil {
		return fmt.Errorf("解析任务数据失败: %v", err)
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 20, "正在获取实例信息...")

	// 获取实例信息
	var instance providerModel.Instance
	if err := global.APP_DB.First(&instance, taskReq.InstanceId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("实例不存在")
		}
		return fmt.Errorf("获取实例信息失败: %v", err)
	}

	// 验证实例所有权
	if instance.UserID != task.UserID {
		return fmt.Errorf("无权限操作此实例")
	}

	// 检查实例状态
	if instance.Status != "running" {
		return fmt.Errorf("只有运行中的实例才能重置密码")
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 30, "正在生成新密码...")

	// 生成新密码
	newPassword := utils.GenerateStrongPassword(12)

	global.APP_LOG.Info("开始重置实例密码",
		zap.Uint("taskId", task.ID),
		zap.Uint("instanceId", instance.ID),
		zap.String("instanceName", instance.Name),
		zap.Uint("userId", instance.UserID))

	// 更新进度
	s.updateTaskProgress(task.ID, 50, "正在设置新密码...")

	// 通过Provider重置实例密码
	providerService := provider2.GetProviderService()
	maxRetries := 3
	var lastErr error
	passwordSetSuccess := false

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			s.updateTaskProgress(task.ID, 50+attempt*10, fmt.Sprintf("正在设置新密码（第%d次尝试）...", attempt))
		}

		err := providerService.SetInstancePassword(ctx, instance.ProviderID, instance.Name, newPassword)
		if err != nil {
			lastErr = err
			global.APP_LOG.Warn("重置实例密码失败，准备重试",
				zap.Uint("taskId", task.ID),
				zap.Uint("instanceId", instance.ID),
				zap.String("instanceName", instance.Name),
				zap.Int("attempt", attempt),
				zap.Int("maxRetries", maxRetries),
				zap.Error(err))
			if attempt < maxRetries {
				select {
				case <-time.After(5 * time.Second):
				case <-ctx.Done():
					return fmt.Errorf("任务已取消")
				}
			}
		} else {
			passwordSetSuccess = true
			break
		}
	}

	if !passwordSetSuccess {
		global.APP_LOG.Error("重置实例密码最终失败",
			zap.Uint("taskId", task.ID),
			zap.Uint("instanceId", instance.ID),
			zap.String("instanceName", instance.Name),
			zap.Error(lastErr))
		return fmt.Errorf("重置密码失败: %v", lastErr)
	}

	// 更新进度
	s.updateTaskProgress(task.ID, 90, "正在更新数据库记录...")

	// 更新数据库中的密码
	err := global.APP_DB.Model(&instance).Update("password", newPassword).Error
	if err != nil {
		global.APP_LOG.Error("更新实例密码到数据库失败",
			zap.Uint("taskId", task.ID),
			zap.Uint("instanceId", instance.ID),
			zap.Error(err))
	}

	// 将新密码存储到任务结果中
	taskResult := map[string]interface{}{
		"instanceId":  instance.ID,
		"providerId":  instance.ProviderID,
		"newPassword": newPassword,
		"resetTime":   time.Now().Unix(),
	}
	taskResultJSON, _ := json.Marshal(taskResult)
	global.APP_DB.Model(task).Update("task_data", string(taskResultJSON))

	// 标记任务完成
	stateManager := GetTaskStateManager()
	if err := stateManager.CompleteMainTask(task.ID, true, "密码重置成功", taskResult); err != nil {
		global.APP_LOG.Error("完成任务失败", zap.Uint("taskId", task.ID), zap.Error(err))
	}

	global.APP_LOG.Info("实例密码重置成功",
		zap.Uint("taskId", task.ID),
		zap.Uint("instanceId", instance.ID),
		zap.String("instanceName", instance.Name),
		zap.Uint("userId", instance.UserID))

	return nil
}
