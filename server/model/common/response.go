package common

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

func Success(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"code": 0,
		"data": data,
		"msg":  "success",
	}
}

func Error(msg string) map[string]interface{} {
	return map[string]interface{}{
		"code": 7,
		"data": nil,
		"msg":  msg,
	}
}

// ParseUintParam 从 URL 参数中解析 uint 值
func ParseUintParam(c *gin.Context, param string) (uint, error) {
	paramStr := c.Param(param)
	value, err := strconv.ParseUint(paramStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(value), nil
}
