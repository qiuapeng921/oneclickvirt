package traffic

import (
	"net/http"
	"strconv"

	"oneclickvirt/global"
	"oneclickvirt/model/common"
	"oneclickvirt/service/traffic"
	userService "oneclickvirt/service/user"
	"oneclickvirt/service/vnstat"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserTrafficAPI 用户流量API
type UserTrafficAPI struct{}

// GetTrafficOverview 获取用户流量概览
// @Summary 获取用户流量概览
// @Description 基于vnStat获取用户流量使用情况概览
// @Tags 用户流量
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} common.Response
// @Router /api/v1/user/traffic/overview [get]
func (api *UserTrafficAPI) GetTrafficOverview(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, common.Response{
			Code: 40001,
			Msg:  "未授权访问",
		})
		return
	}

	userTrafficService := traffic.NewUserTrafficService()
	overview, err := userTrafficService.GetUserTrafficOverview(userID)
	if err != nil {
		global.APP_LOG.Error("获取用户流量概览失败",
			zap.Uint("userID", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 50000,
			Msg:  "获取流量概览失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Code: 0,
		Msg:  "获取流量概览成功",
		Data: overview,
	})
}

// GetInstanceTrafficDetail 获取实例流量详情
// @Summary 获取实例流量详情
// @Description 获取指定实例的详细流量统计信息
// @Tags 用户流量
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param instanceId path int true "实例ID"
// @Success 200 {object} common.Response
// @Router /api/v1/user/traffic/instance/{instanceId} [get]
func (api *UserTrafficAPI) GetInstanceTrafficDetail(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, common.Response{
			Code: 40001,
			Msg:  "未授权访问",
		})
		return
	}

	instanceIDStr := c.Param("instanceId")
	instanceID, err := strconv.ParseUint(instanceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 40000,
			Msg:  "实例ID格式错误",
		})
		return
	}

	userTrafficService := traffic.NewUserTrafficService()
	detail, err := userTrafficService.GetInstanceTrafficDetail(userID, uint(instanceID))
	if err != nil {
		global.APP_LOG.Error("获取实例流量详情失败",
			zap.Uint("userID", userID),
			zap.Uint("instanceID", uint(instanceID)),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 50000,
			Msg:  "获取实例流量详情失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Code: 0,
		Msg:  "获取实例流量详情成功",
		Data: detail,
	})
}

// GetInstancesTrafficSummary 获取用户所有实例流量汇总
// @Summary 获取用户所有实例流量汇总
// @Description 获取用户所有实例的流量使用汇总信息
// @Tags 用户流量
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} common.Response
// @Router /api/v1/user/traffic/instances [get]
func (api *UserTrafficAPI) GetInstancesTrafficSummary(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, common.Response{
			Code: 40001,
			Msg:  "未授权访问",
		})
		return
	}

	userTrafficService := traffic.NewUserTrafficService()
	summary, err := userTrafficService.GetUserInstancesTrafficSummary(userID)
	if err != nil {
		global.APP_LOG.Error("获取用户实例流量汇总失败",
			zap.Uint("userID", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 50000,
			Msg:  "获取实例流量汇总失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Code: 0,
		Msg:  "获取实例流量汇总成功",
		Data: summary,
	})
}

// GetTrafficLimitStatus 获取流量限制状态
// @Summary 获取流量限制状态
// @Description 获取用户的流量限制状态和受限实例信息
// @Tags 用户流量
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} common.Response
// @Router /api/v1/user/traffic/limit-status [get]
func (api *UserTrafficAPI) GetTrafficLimitStatus(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, common.Response{
			Code: 40001,
			Msg:  "未授权访问",
		})
		return
	}

	userTrafficService := traffic.NewUserTrafficService()
	status, err := userTrafficService.GetTrafficLimitStatus(userID)
	if err != nil {
		global.APP_LOG.Error("获取流量限制状态失败",
			zap.Uint("userID", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 50000,
			Msg:  "获取流量限制状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Code: 0,
		Msg:  "获取流量限制状态成功",
		Data: status,
	})
}

// GetVnStatData 获取原始vnStat数据
// @Summary 获取原始vnStat数据
// @Description 获取指定实例的原始vnStat统计数据
// @Tags 用户流量
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param instanceId path int true "实例ID"
// @Param interface query string false "网络接口名称"
// @Success 200 {object} common.Response
// @Router /api/v1/user/traffic/vnstat/{instanceId} [get]
func (api *UserTrafficAPI) GetVnStatData(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, common.Response{
			Code: 40001,
			Msg:  "未授权访问",
		})
		return
	}

	instanceIDStr := c.Param("instanceId")
	instanceID, err := strconv.ParseUint(instanceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 40000,
			Msg:  "实例ID格式错误",
		})
		return
	}

	interfaceName := c.Query("interface")

	// 验证用户权限
	userServiceInstance := userService.NewService()
	if !userServiceInstance.HasInstanceAccess(userID, uint(instanceID)) {
		c.JSON(http.StatusForbidden, common.Response{
			Code: 40003,
			Msg:  "无权限访问该实例",
		})
		return
	}

	// 获取vnStat数据
	vnstatService := vnstat.NewService()
	vnstatSummary, err := vnstatService.GetVnStatSummary(uint(instanceID), interfaceName)
	if err != nil {
		global.APP_LOG.Error("获取vnStat数据失败",
			zap.Uint("userID", userID),
			zap.Uint("instanceID", uint(instanceID)),
			zap.String("interface", interfaceName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 50000,
			Msg:  "获取vnStat数据失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Code: 0,
		Msg:  "获取vnStat数据成功",
		Data: vnstatSummary,
	})
}

// getUserIDFromContext 从上下文中获取用户ID（需要根据实际认证中间件实现）
func getUserIDFromContext(c *gin.Context) uint {
	// 这里需要根据实际的认证中间件实现
	// 例如从JWT token或session中获取用户ID
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uint); ok {
			return id
		}
	}
	return 0
}
