package traffic

import (
	"net/http"
	"strconv"

	"oneclickvirt/global"
	"oneclickvirt/model/common"
	"oneclickvirt/service/traffic"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminTrafficAPI 管理员流量API
type AdminTrafficAPI struct{}

// GetSystemTrafficOverview 获取系统流量概览
// @Summary 获取系统流量概览
// @Description 获取整个系统的流量使用情况概览
// @Tags 管理员流量
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} common.Response
// @Router /api/v1/admin/traffic/overview [get]
func (api *AdminTrafficAPI) GetSystemTrafficOverview(c *gin.Context) {
	trafficLimitService := traffic.NewLimitService()

	// 获取系统全局流量统计
	systemStats, err := trafficLimitService.GetSystemTrafficStats()
	if err != nil {
		global.APP_LOG.Error("获取系统流量统计失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 50000,
			Msg:  "获取系统流量统计失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Code: 0,
		Msg:  "获取系统流量概览成功",
		Data: systemStats,
	})
}

// GetProviderTrafficStats 获取Provider流量统计
// @Summary 获取Provider流量统计
// @Description 获取指定Provider的流量使用情况
// @Tags 管理员流量
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param providerId path int true "Provider ID"
// @Success 200 {object} common.Response
// @Router /api/v1/admin/traffic/provider/{providerId} [get]
func (api *AdminTrafficAPI) GetProviderTrafficStats(c *gin.Context) {
	providerIDStr := c.Param("providerId")
	providerID, err := strconv.ParseUint(providerIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 40000,
			Msg:  "Provider ID格式错误",
		})
		return
	}

	trafficLimitService := traffic.NewLimitService()

	// 获取Provider流量使用情况
	providerUsage, err := trafficLimitService.GetProviderTrafficUsageWithVnStat(uint(providerID))
	if err != nil {
		global.APP_LOG.Error("获取Provider流量统计失败",
			zap.Uint("providerID", uint(providerID)),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 50000,
			Msg:  "获取Provider流量统计失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Code: 0,
		Msg:  "获取Provider流量统计成功",
		Data: providerUsage,
	})
}

// GetUserTrafficStats 获取用户流量统计
// @Summary 获取用户流量统计
// @Description 获取指定用户的流量使用情况
// @Tags 管理员流量
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param userId path int true "用户ID"
// @Success 200 {object} common.Response
// @Router /api/v1/admin/traffic/user/{userId} [get]
func (api *AdminTrafficAPI) GetUserTrafficStats(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 40000,
			Msg:  "用户ID格式错误",
		})
		return
	}

	trafficLimitService := traffic.NewLimitService()

	// 获取用户流量使用情况
	userUsage, err := trafficLimitService.GetUserTrafficUsageWithVnStat(uint(userID))
	if err != nil {
		global.APP_LOG.Error("获取用户流量统计失败",
			zap.Uint("userID", uint(userID)),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 50000,
			Msg:  "获取用户流量统计失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Code: 0,
		Msg:  "获取用户流量统计成功",
		Data: userUsage,
	})
}

// GetAllUsersTrafficRank 获取所有用户流量排行
// @Summary 获取用户流量排行榜
// @Description 获取系统中所有用户的流量使用排行榜
// @Tags 管理员流量
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param limit query int false "返回结果数量限制，默认20"
// @Success 200 {object} common.Response
// @Router /api/v1/admin/traffic/users/rank [get]
func (api *AdminTrafficAPI) GetAllUsersTrafficRank(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	trafficLimitService := traffic.NewLimitService()

	// 获取用户流量排行榜
	userRankings, err := trafficLimitService.GetUsersTrafficRanking(limit)
	if err != nil {
		global.APP_LOG.Error("获取用户流量排行榜失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 50000,
			Msg:  "获取用户流量排行榜失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Code: 0,
		Msg:  "获取用户流量排行榜成功",
		Data: map[string]interface{}{
			"rankings": userRankings,
			"total":    len(userRankings),
			"limit":    limit,
		},
	})
}

// ManageTrafficLimits 管理流量限制
// @Summary 管理流量限制
// @Description 手动设置或解除用户/Provider的流量限制
// @Tags 管理员流量
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body ManageTrafficLimitRequest true "流量限制管理请求"
// @Success 200 {object} common.Response
// @Router /api/v1/admin/traffic/manage [post]
func (api *AdminTrafficAPI) ManageTrafficLimits(c *gin.Context) {
	var req ManageTrafficLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 40000,
			Msg:  "请求参数错误: " + err.Error(),
		})
		return
	}

	trafficLimitService := traffic.NewLimitService()

	var err error
	var result string

	switch req.Type {
	case "user":
		if req.Action == "limit" {
			err = trafficLimitService.SetUserTrafficLimit(req.TargetID, req.Reason)
			result = "设置用户流量限制"
		} else if req.Action == "unlimit" {
			err = trafficLimitService.RemoveUserTrafficLimit(req.TargetID)
			result = "解除用户流量限制"
		} else {
			c.JSON(http.StatusBadRequest, common.Response{
				Code: 40000,
				Msg:  "不支持的操作类型",
			})
			return
		}
	case "provider":
		if req.Action == "limit" {
			err = trafficLimitService.SetProviderTrafficLimit(req.TargetID, req.Reason)
			result = "设置Provider流量限制"
		} else if req.Action == "unlimit" {
			err = trafficLimitService.RemoveProviderTrafficLimit(req.TargetID)
			result = "解除Provider流量限制"
		} else {
			c.JSON(http.StatusBadRequest, common.Response{
				Code: 40000,
				Msg:  "不支持的操作类型",
			})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 40000,
			Msg:  "不支持的目标类型",
		})
		return
	}

	if err != nil {
		global.APP_LOG.Error("管理流量限制失败",
			zap.String("type", req.Type),
			zap.String("action", req.Action),
			zap.Uint("targetID", req.TargetID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 50000,
			Msg:  result + "失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Code: 0,
		Msg:  result + "成功",
		Data: map[string]interface{}{
			"type":      req.Type,
			"action":    req.Action,
			"target_id": req.TargetID,
			"reason":    req.Reason,
		},
	})
}

// ManageTrafficLimitRequest 流量限制管理请求
type ManageTrafficLimitRequest struct {
	Type     string `json:"type" binding:"required"`      // "user" 或 "provider"
	Action   string `json:"action" binding:"required"`    // "limit" 或 "unlimit"
	TargetID uint   `json:"target_id" binding:"required"` // 目标用户ID或Provider ID
	Reason   string `json:"reason"`                       // 限制原因（仅在action为limit时需要）
}
