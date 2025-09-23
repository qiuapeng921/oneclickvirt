package auth

import (
	"time"

	"gorm.io/gorm"
)

// LoginRequest 登录请求
type LoginRequest struct {
	Username  string `json:"username" binding:"required" example:"admin"`
	Password  string `json:"password" binding:"required" example:"password"`
	Captcha   string `json:"captcha,omitempty"`
	CaptchaId string `json:"captchaId,omitempty"`
	LoginType string `json:"loginType,omitempty"` // username, email, phone
	UserType  string `json:"userType,omitempty"`  // admin, user
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username   string `json:"username" binding:"required" example:"user123"`
	Password   string `json:"password" binding:"required" example:"password123"`
	Nickname   string `json:"nickname" example:"昵称"`
	Email      string `json:"email" example:"user@example.com"`
	Phone      string `json:"phone,omitempty" example:"13800138000"`
	Telegram   string `json:"telegram,omitempty"`
	QQ         string `json:"qq,omitempty"`
	InviteCode string `json:"inviteCode" example:"INVITE123"`
	Captcha    string `json:"captcha"`
	CaptchaId  string `json:"captchaId"`
}

// ForgotPasswordRequest 忘记密码请求
type ForgotPasswordRequest struct {
	Email     string `json:"email" binding:"required,email" example:"user@example.com"`
	CaptchaId string `json:"captchaId,omitempty"`
	Captcha   string `json:"captcha,omitempty"`
	UserType  string `json:"userType,omitempty"`
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Token string `json:"token" binding:"required"`
}

// GetCaptchaRequest 获取验证码请求
type GetCaptchaRequest struct {
	Type   string `json:"type,omitempty"` // 验证码类型，可选
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string `json:"token"`
}

// CaptchaResponse 验证码响应
type CaptchaResponse struct {
	CaptchaId string `json:"captchaId"`
	PicPath   string `json:"picPath"`
	ImageData string `json:"imageData"`
}

// Permission 权限模型
type Permission struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	Name        string         `json:"name" gorm:"uniqueIndex;not null;size:64"`
	Description string         `json:"description" gorm:"size:255"`
	Resource    string         `json:"resource" gorm:"size:64"`
	Action      string         `json:"action" gorm:"size:64"`
	Status      int            `json:"status" gorm:"default:1"`
}
type Api struct {
	Path        string
	Method      string
	Description string
	Group       string
	Status      int
}

type ExportUsersRequest struct {
	UserIDs   []uint   `json:"userIds"`
	Status    *int     `json:"status"`
	StartTime string   `json:"startTime"`
	EndTime   string   `json:"endTime"`
	Format    string   `json:"format"`
	Fields    []string `json:"fields"`
}
type ExportOperationLogsRequest struct {
	UserID    *uint  `json:"userId"`
	UserIDs   []uint `json:"userIds"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Format    string `json:"format"`
}
type BatchGenerateInviteCodesRequest struct {
	Count       int    `json:"count"`
	MaxUse      int    `json:"maxUse"`
	ExpireAt    string `json:"expireAt"`
	ExpireDays  int    `json:"expireDays"`
	Remark      string `json:"remark"`
	Description string `json:"description"`
	Length      int    `json:"length"` // 邀请码长度，默认8位
}
type SearchUsersRequest struct {
	Keyword   string `json:"keyword"`
	UserType  string `json:"userType"`
	Status    *int   `json:"status"`
	RoleID    *uint  `json:"roleId"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	SortBy    string `json:"sortBy"`
	SortOrder string `json:"sortOrder"`
	Page      int    `json:"page"`
	PageSize  int    `json:"pageSize"`
}
