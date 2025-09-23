package invite

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"oneclickvirt/service/database"
	"time"

	"oneclickvirt/global"
	"oneclickvirt/model/admin"
	"oneclickvirt/model/system"
	userModel "oneclickvirt/model/user"

	"gorm.io/gorm"
)

// Service 管理员邀请码管理服务
type Service struct{}

// NewService 创建邀请码管理服务
func NewService() *Service {
	return &Service{}
}

// GetInviteCodeList 获取邀请码列表
func (s *Service) GetInviteCodeList(req admin.InviteCodeListRequest) ([]admin.InviteCodeResponse, int64, error) {
	var inviteCodes []system.InviteCode
	var total int64

	query := global.APP_DB.Model(&system.InviteCode{})

	if req.Code != "" {
		query = query.Where("code LIKE ?", "%"+req.Code+"%")
	}
	if req.Used != nil {
		query = query.Where("used = ?", *req.Used)
	}
	if req.Status != 0 {
		query = query.Where("status = ?", req.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&inviteCodes).Error; err != nil {
		return nil, 0, err
	}

	var codeResponses []admin.InviteCodeResponse
	for _, code := range inviteCodes {
		var createdByUser, usedByUser string
		if code.CreatorID != 0 {
			var user userModel.User
			if err := global.APP_DB.First(&user, code.CreatorID).Error; err == nil {
				createdByUser = user.Username
			}
		}
		if code.UsedBy != nil && *code.UsedBy != 0 {
			var user userModel.User
			if err := global.APP_DB.First(&user, *code.UsedBy).Error; err == nil {
				usedByUser = user.Username
			}
		}

		codeResponse := admin.InviteCodeResponse{
			InviteCode:    code,
			CreatedByUser: createdByUser,
			UsedByUser:    usedByUser,
		}
		codeResponses = append(codeResponses, codeResponse)
	}

	return codeResponses, total, nil
}

// CreateInviteCode 创建邀请码
func (s *Service) CreateInviteCode(req admin.CreateInviteCodeRequest, createdBy uint) error {
	// 如果指定了自定义邀请码
	if req.Code != "" {
		// 验证自定义邀请码格式（仅允许数字和大写字母）
		for _, c := range req.Code {
			if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z')) {
				return fmt.Errorf("自定义邀请码只能包含数字和英文大写字母")
			}
		}

		// 验证邀请码是否已存在
		var existingCode system.InviteCode
		if err := global.APP_DB.Where("code = ?", req.Code).First(&existingCode).Error; err == nil {
			return fmt.Errorf("邀请码 %s 已存在", req.Code)
		}
		var expiresAt *time.Time
		if req.ExpiresAt != "" {
			if parsedTime, err := time.Parse("2006-01-02 15:04:05", req.ExpiresAt); err == nil {
				expiresAt = &parsedTime
			}
		}
		inviteCode := system.InviteCode{
			Code:        req.Code,
			CreatorID:   createdBy,
			CreatorName: "", // 将由数据库触发器或其他逻辑填充
			Description: req.Remark,
			MaxUses:     req.MaxUses,
			ExpiresAt:   expiresAt,
			Status:      1,
		}
		dbService := database.GetDatabaseService()
		if err := dbService.ExecuteTransaction(context.Background(), func(tx *gorm.DB) error {
			return tx.Create(&inviteCode).Error
		}); err != nil {
			return err
		}
		return nil
	}
	// 如果没有指定自定义邀请码，按原来的逻辑批量生成
	codeLength := req.Length
	if codeLength <= 0 {
		codeLength = 8 // 默认8位
	}

	for i := 0; i < req.Count; i++ {
		code := s.generateInviteCodeWithLength(codeLength)
		// 确保生成的邀请码不重复
		var existingCode system.InviteCode
		for {
			if err := global.APP_DB.Where("code = ?", code).First(&existingCode).Error; err != nil {
				if err.Error() == "record not found" {
					break
				}
				return fmt.Errorf("检查邀请码唯一性失败: %v", err)
			}
			// 如果邀请码已存在，重新生成
			code = s.generateInviteCodeWithLength(codeLength)
		}
		var expiresAt *time.Time
		if req.ExpiresAt != "" {
			if parsedTime, err := time.Parse("2006-01-02 15:04:05", req.ExpiresAt); err == nil {
				expiresAt = &parsedTime
			}
		}
		inviteCode := system.InviteCode{
			Code:        code,
			CreatorID:   createdBy,
			CreatorName: "", // 将由数据库触发器或其他逻辑填充
			Description: req.Remark,
			MaxUses:     req.MaxUses,
			ExpiresAt:   expiresAt,
			Status:      1,
		}
		dbService := database.GetDatabaseService()
		if err := dbService.ExecuteTransaction(context.Background(), func(tx *gorm.DB) error {
			return tx.Create(&inviteCode).Error
		}); err != nil {
			return err
		}
	}
	return nil
}

// GenerateInviteCodes 生成批量邀请码
func (s *Service) GenerateInviteCodes(req admin.CreateInviteCodeRequest, createdBy uint) ([]string, error) {
	var codes []string

	codeLength := req.Length
	if codeLength <= 0 {
		codeLength = 8 // 默认8位
	}

	for i := 0; i < req.Count; i++ {
		code := s.generateInviteCodeWithLength(codeLength)

		var expiresAt *time.Time
		if req.ExpiresAt != "" {
			if parsedTime, err := time.Parse("2006-01-02 15:04:05", req.ExpiresAt); err == nil {
				expiresAt = &parsedTime
			}
		}

		inviteCode := system.InviteCode{
			Code:        code,
			CreatorID:   createdBy,
			CreatorName: "", // 将由数据库触发器或其他逻辑填充
			Description: req.Remark,
			MaxUses:     req.MaxUses,
			ExpiresAt:   expiresAt,
			Status:      1,
		}

		dbService := database.GetDatabaseService()
		if err := dbService.ExecuteTransaction(context.Background(), func(tx *gorm.DB) error {
			return tx.Create(&inviteCode).Error
		}); err != nil {
			return nil, err
		}

		codes = append(codes, code)
	}

	return codes, nil
}

// generateInviteCodeWithLength 生成指定长度的随机邀请码 (仅数字和英文大写字母)
func (s *Service) generateInviteCodeWithLength(length int) string {
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := make([]byte, length)

	for i := range bytes {
		randBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// 如果随机数生成失败，使用默认字符
			bytes[i] = charset[0]
		} else {
			bytes[i] = charset[randBig.Int64()]
		}
	}

	return string(bytes)
}

// DeleteInviteCode 删除邀请码
func (s *Service) DeleteInviteCode(codeID uint) error {
	dbService := database.GetDatabaseService()
	return dbService.ExecuteTransaction(context.Background(), func(tx *gorm.DB) error {
		return tx.Delete(&system.InviteCode{}, codeID).Error
	})
}

// ExportInviteCodes 导出邀请码为CSV格式
func (s *Service) ExportInviteCodes() (string, error) {
	var codes []system.InviteCode
	if err := global.APP_DB.Find(&codes).Error; err != nil {
		return "", err
	}

	csvContent := "Code,MaxUses,UsedCount,ExpiresAt,Description,Status,CreatedAt\n"
	for _, code := range codes {
		expiresAt := ""
		if code.ExpiresAt != nil {
			expiresAt = code.ExpiresAt.Format("2006-01-02 15:04:05")
		}
		status := "启用"
		if code.Status == 0 {
			status = "禁用"
		}
		csvContent += fmt.Sprintf("%s,%d,%d,%s,%s,%s,%s\n",
			code.Code,
			code.MaxUses,
			code.UsedCount,
			expiresAt,
			code.Description,
			status,
			code.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	return csvContent, nil
}
