package api

import (
	"context"
	"errors"
	"fmt"
	"oneclickvirt/service/database"
	"strconv"
	"strings"

	"oneclickvirt/global"
	"oneclickvirt/model/api"
	"oneclickvirt/model/auth"
	"oneclickvirt/model/common"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ApiService struct{}

// GetApiList 获取API列表
func (s *ApiService) GetApiList(req common.PageInfo) (interface{}, error) {
	var apis []auth.Api
	var total int64

	offset := (req.Page - 1) * req.PageSize

	// 获取总数
	if err := global.APP_DB.Model(&auth.Api{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	if err := global.APP_DB.Offset(offset).Limit(req.PageSize).Find(&apis).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"list":     apis,
		"total":    total,
		"page":     req.Page,
		"pageSize": req.PageSize,
	}, nil
}

// CreateApi 创建API接口
func (s *ApiService) CreateApi(path, method, description, group string) error {
	api := auth.Api{
		Path:        path,
		Method:      method,
		Description: description,
		Group:       group,
	}

	// 检查是否已存在
	var existingApi auth.Api
	if err := global.APP_DB.Where("path = ? AND method = ?", path, method).First(&existingApi).Error; err == nil {
		return errors.New("API接口已存在")
	}

	dbService := database.GetDatabaseService()
	return dbService.ExecuteTransaction(context.Background(), func(tx *gorm.DB) error {
		return tx.Create(&api).Error
	})
}

// UpdateApi 更新API接口
func (s *ApiService) UpdateApi(apiID uint, path, method, description, group string) error {
	var api auth.Api
	if err := global.APP_DB.First(&api, apiID).Error; err != nil {
		return err
	}

	// 检查是否与其他API冲突
	var existingApi auth.Api
	if err := global.APP_DB.Where("path = ? AND method = ? AND id != ?", path, method, apiID).First(&existingApi).Error; err == nil {
		return errors.New("API接口已存在")
	}

	api.Path = path
	api.Method = method
	api.Description = description
	api.Group = group

	dbService := database.GetDatabaseService()
	return dbService.ExecuteTransaction(context.Background(), func(tx *gorm.DB) error {
		return tx.Save(&api).Error
	})
}

// DeleteApi 删除API接口
func (s *ApiService) DeleteApi(apiID uint) error {
	// 先检查是否有权限关联到此API
	var count int64
	if err := global.APP_DB.Model(&auth.Permission{}).Where("resource = ?", "api_"+strconv.FormatUint(uint64(apiID), 10)).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该API接口已被权限关联，无法删除")
	}

	dbService := database.GetDatabaseService()
	return dbService.ExecuteTransaction(context.Background(), func(tx *gorm.DB) error {
		return tx.Delete(&auth.Api{}, apiID).Error
	})
}

// GetApiByID 根据ID获取API
func (s *ApiService) GetApiByID(apiID uint) (*auth.Api, error) {
	var api auth.Api
	if err := global.APP_DB.First(&api, apiID).Error; err != nil {
		return nil, err
	}
	return &api, nil
}

// GetAllApis 获取所有API
func (s *ApiService) GetAllApis() ([]auth.Api, error) {
	var apis []auth.Api
	if err := global.APP_DB.Find(&apis).Error; err != nil {
		return nil, err
	}
	return apis, nil
}

// GetApisByGroup 按分组获取API
func (s *ApiService) GetApisByGroup() (map[string][]auth.Api, error) {
	apis, err := s.GetAllApis()
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]auth.Api)
	for _, api := range apis {
		group := api.Group
		if group == "" {
			group = "未分组"
		}
		grouped[group] = append(grouped[group], api)
	}

	return grouped, nil
}

// GetAllRoutes 获取所有路由信息
func (s *ApiService) GetAllRoutes(engine *gin.Engine) []api.RouteInfo {
	var routes []api.RouteInfo

	// 通过反射获取路由信息
	for _, route := range engine.Routes() {
		group := s.extractGroupFromPath(route.Path)
		routes = append(routes, api.RouteInfo{
			Method: route.Method,
			Path:   route.Path,
			Group:  group,
			Name:   route.Handler,
		})
	}

	return routes
}

// SyncApisFromRoutes 从路由同步API到数据库
func (s *ApiService) SyncApisFromRoutes(engine *gin.Engine) (map[string]interface{}, error) {
	routes := s.GetAllRoutes(engine)

	var created, updated, skipped int
	var errors []string

	for _, route := range routes {
		// 跳过一些不需要同步的路由
		if s.shouldSkipRoute(route.Path) {
			skipped++
			continue
		}

		// 检查API是否已存在
		var existingApi auth.Api
		err := global.APP_DB.Where("path = ? AND method = ?", route.Path, route.Method).First(&existingApi).Error

		if err != nil {
			// API不存在，创建新的
			api := auth.Api{
				Path:        route.Path,
				Method:      route.Method,
				Description: s.generateDescription(route),
				Group:       route.Group,
				Status:      1,
			}

			dbService := database.GetDatabaseService()
			if err := dbService.ExecuteTransaction(context.Background(), func(tx *gorm.DB) error {
				return tx.Create(&api).Error
			}); err != nil {
				errors = append(errors, fmt.Sprintf("创建API失败 %s %s: %v", route.Method, route.Path, err))
			} else {
				created++
			}
		} else {
			// API已存在，更新信息
			existingApi.Description = s.generateDescription(route)
			existingApi.Group = route.Group

			dbService := database.GetDatabaseService()
			if err := dbService.ExecuteTransaction(context.Background(), func(tx *gorm.DB) error {
				return tx.Save(&existingApi).Error
			}); err != nil {
				errors = append(errors, fmt.Sprintf("更新API失败 %s %s: %v", route.Method, route.Path, err))
			} else {
				updated++
			}
		}
	}

	result := map[string]interface{}{
		"total_routes": len(routes),
		"created":      created,
		"updated":      updated,
		"skipped":      skipped,
		"errors":       errors,
	}

	if len(errors) > 0 {
		return result, fmt.Errorf("同步过程中发生 %d 个错误", len(errors))
	}

	return result, nil
}

// shouldSkipRoute 判断是否应该跳过某个路由
func (s *ApiService) shouldSkipRoute(path string) bool {
	skipPaths := []string{
		"/favicon.ico",
		"/static/",
		"/assets/",
		"/swagger/",
		"/debug/",
		"/metrics",
		"/health",
		"/ping",
	}
	for _, skipPath := range skipPaths {
		if strings.Contains(path, skipPath) {
			return true
		}
	}
	return false
}

// extractGroupFromPath 从路径中提取分组信息
func (s *ApiService) extractGroupFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) >= 3 && parts[0] == "api" {
		if len(parts) >= 4 {
			return parts[3] // 例如 /api/v1/users -> users
		}
		return parts[2] // 例如 /api/v1 -> v1
	}

	if len(parts) >= 2 {
		return parts[1] // 例如 /admin/users -> admin
	}

	if len(parts) >= 1 && parts[0] != "" {
		return parts[0]
	}

	return "默认分组"
}

// generateDescription 生成API描述
func (s *ApiService) generateDescription(route api.RouteInfo) string {
	methodDescMap := map[string]string{
		"GET":    "查询",
		"POST":   "创建",
		"PUT":    "更新",
		"DELETE": "删除",
		"PATCH":  "部分更新",
	}

	methodDesc := methodDescMap[route.Method]
	if methodDesc == "" {
		methodDesc = route.Method
	}

	// 从路径生成更友好的描述
	pathParts := strings.Split(strings.Trim(route.Path, "/"), "/")
	resource := "资源"

	if len(pathParts) > 0 {
		lastPart := pathParts[len(pathParts)-1]
		if !strings.Contains(lastPart, ":") && lastPart != "" {
			resource = lastPart
		} else if len(pathParts) > 1 {
			resource = pathParts[len(pathParts)-2]
		}
	}

	return fmt.Sprintf("%s%s", methodDesc, resource)
}
