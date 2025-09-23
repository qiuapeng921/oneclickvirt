package config

import (
	"net/http"
	"oneclickvirt/service/api"
	"strconv"

	"github.com/gin-gonic/gin"
	"oneclickvirt/global"
	"oneclickvirt/model/common"
)

// GetUserRoleList 获取用户角色列表
// @Summary 获取用户角色列表
// @Description 获取用户的角色列表
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=[]interface{}} "获取成功"
// @Router /user/roles [get]
func GetUserRoleList(c *gin.Context) {
	c.JSON(http.StatusOK, common.Response{
		Code: 200,
		Msg:  "获取成功",
		Data: []interface{}{},
	})
}

// GetPermissionList 获取权限列表
// @Summary 获取权限列表
// @Description 获取系统的权限列表
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=map[string]interface{}} "获取成功"
// @Router /permission/list [get]
func GetPermissionList(c *gin.Context) {
	var pageInfo common.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		c.JSON(http.StatusBadRequest, common.Error("请求参数错误: "+err.Error()))
		return
	}

	// 设置默认值
	if pageInfo.Page <= 0 {
		pageInfo.Page = 1
	}
	if pageInfo.PageSize <= 0 {
		pageInfo.PageSize = 10
	}

	// permissionService := service.PermissionService{}
	// 暂时注释掉，将在后续实现
	// result, err := permissionService.GetPermissionList(pageInfo)
	result := map[string]interface{}{
		"list":     []interface{}{},
		"total":    0,
		"page":     pageInfo.Page,
		"pageSize": pageInfo.PageSize,
	}
	var err error = nil
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, common.Success(result))
}

// GetMenuList 菜单相关占位函数
func GetMenuList(c *gin.Context) {
	c.JSON(http.StatusOK, common.Response{
		Code: 200,
		Msg:  "获取成功",
		Data: map[string]interface{}{
			"list":  []interface{}{},
			"total": 0,
		},
	})
}

func CreateMenu(c *gin.Context) {
	c.JSON(http.StatusOK, common.Response{
		Code: 200,
		Msg:  "创建成功",
	})
}

func UpdateMenu(c *gin.Context) {
	c.JSON(http.StatusOK, common.Response{
		Code: 200,
		Msg:  "更新成功",
	})
}

func DeleteMenu(c *gin.Context) {
	c.JSON(http.StatusOK, common.Response{
		Code: 200,
		Msg:  "删除成功",
	})
}

// GetApiList 获取API列表
// @Summary 获取API列表
// @Description 获取系统的API接口列表
// @Tags API管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=map[string]interface{}} "获取成功"
// @Router /api/list [get]
func GetApiList(c *gin.Context) {
	var pageInfo common.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		c.JSON(http.StatusBadRequest, common.Error("请求参数错误: "+err.Error()))
		return
	}

	// 设置默认值
	if pageInfo.Page <= 0 {
		pageInfo.Page = 1
	}
	if pageInfo.PageSize <= 0 {
		pageInfo.PageSize = 10
	}

	apiService := api.ApiService{}
	result, err := apiService.GetApiList(pageInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, common.Success(result))
}

// CreateApi 创建API接口
// @Summary 创建API接口
// @Description 创建新的API接口记录
// @Tags API管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response "创建成功"
// @Router /api/create [post]
func CreateApi(c *gin.Context) {
	var req struct {
		Path        string `json:"path" binding:"required"`
		Method      string `json:"method" binding:"required"`
		Description string `json:"description"`
		Group       string `json:"group"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Error("请求参数错误: "+err.Error()))
		return
	}

	apiService := api.ApiService{}
	if err := apiService.CreateApi(req.Path, req.Method, req.Description, req.Group); err != nil {
		c.JSON(http.StatusInternalServerError, common.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, common.Success("创建成功"))
}

// UpdateApi 更新API接口
// @Summary 更新API接口
// @Description 更新API接口信息
// @Tags API管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "API ID"
// @Param request body object true "更新API请求参数"
// @Success 200 {object} common.Response "更新成功"
// @Router /api/{id} [put]
func UpdateApi(c *gin.Context) {
	idStr := c.Param("id")
	apiID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Error("无效的API ID"))
		return
	}

	var req struct {
		Path        string `json:"path" binding:"required"`
		Method      string `json:"method" binding:"required"`
		Description string `json:"description"`
		Group       string `json:"group"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Error("请求参数错误: "+err.Error()))
		return
	}

	apiService := api.ApiService{}
	if err := apiService.UpdateApi(uint(apiID), req.Path, req.Method, req.Description, req.Group); err != nil {
		c.JSON(http.StatusInternalServerError, common.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, common.Success("更新成功"))
}

// DeleteApi 删除API接口
// @Summary 删除API接口
// @Description 删除API接口记录
// @Tags API管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "API ID"
// @Success 200 {object} common.Response "删除成功"
// @Router /api/{id} [delete]
func DeleteApi(c *gin.Context) {
	idStr := c.Param("id")
	apiID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Error("无效的API ID"))
		return
	}

	apiService := api.ApiService{}
	if err := apiService.DeleteApi(uint(apiID)); err != nil {
		c.JSON(http.StatusInternalServerError, common.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, common.Success("删除成功"))
}

// SyncApis 同步API接口
// @Summary 同步API接口
// @Description 从路由中自动发现并同步API接口到数据库
// @Tags API管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=map[string]interface{}} "同步成功"
// @Router /apis/sync [post]
func SyncApis(c *gin.Context) {
	apiService := api.ApiService{}
	result, err := apiService.SyncApisFromRoutes(global.APP_ENGINE)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, common.Success(result))
}

// GetRoutes 获取当前系统路由信息
// @Summary 获取路由信息
// @Description 获取当前系统的所有路由信息
// @Tags API管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=[]interface{}} "获取成功"
// @Router /apis/routes [get]
func GetRoutes(c *gin.Context) {
	apiService := api.ApiService{}
	routes := apiService.GetAllRoutes(global.APP_ENGINE)

	c.JSON(http.StatusOK, common.Success(map[string]interface{}{
		"routes": routes,
		"total":  len(routes),
	}))
}
