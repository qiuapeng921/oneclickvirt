package config

import (
	"oneclickvirt/model/common"
	"oneclickvirt/model/config"
	"oneclickvirt/service/auth"

	"github.com/gin-gonic/gin"
)

// GetConfig 获取系统配置
// @Summary 获取系统配置
// @Description 获取当前系统的配置信息
// @Tags 系统配置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=interface{}} "获取成功"
// @Failure 500 {object} common.Response "获取失败"
// @Router /config [get]
func GetConfig(c *gin.Context) {
	configService := auth.ConfigService{}
	result := configService.GetConfig()

	common.ResponseSuccess(c, result)
}

// UpdateConfig 更新系统配置
// @Summary 更新系统配置
// @Description 更新系统的各项配置参数
// @Tags 系统配置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body config.UpdateConfigRequest true "更新配置请求参数"
// @Success 200 {object} common.Response "更新成功"
// @Failure 400 {object} common.Response "参数错误"
// @Failure 500 {object} common.Response "更新失败"
// @Router /config [put]
func UpdateConfig(c *gin.Context) {
	var req config.UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ResponseWithError(c, common.NewError(common.CodeValidationError, "参数错误"))
		return
	}

	configService := auth.ConfigService{}
	err := configService.UpdateConfig(req)
	if err != nil {
		common.ResponseWithError(c, common.NewError(common.CodeConfigError, err.Error()))
		return
	}

	common.ResponseSuccess(c, nil, "配置更新成功")
}
