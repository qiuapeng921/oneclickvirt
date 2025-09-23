package files

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"oneclickvirt/global"
	"oneclickvirt/model/system"
	"oneclickvirt/utils"

	"go.uber.org/zap"
)

// FileStatsService 文件统计服务
type FileStatsService struct{}

// GetUploadStats 获取实际的上传统计信息
func (s *FileStatsService) GetUploadStats(uploadDir string, retentionDays int) (*system.FileStats, error) {
	stats := &system.FileStats{}
	avatarDir := filepath.Join(uploadDir, "avatars")
	today := time.Now().Truncate(24 * time.Hour)
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	err := filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			global.APP_LOG.Debug("统计文件时遇到错误", zap.String("path", path), zap.String("error", utils.FormatError(err)))
			return nil // 继续处理其他文件
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 统计总文件数和大小
		stats.TotalFiles++
		stats.TotalSize += info.Size()

		// 统计头像文件
		if strings.HasPrefix(path, avatarDir) {
			stats.AvatarCount++
		} else {
			// 非头像文件，检查是否可清理
			if retentionDays > 0 && info.ModTime().Before(cutoffTime) {
				stats.CleanableFiles++
			}
		}

		// 统计今日上传文件
		if info.ModTime().After(today) {
			stats.TodayUploads++
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 计算平均文件大小
	if stats.TotalFiles > 0 {
		stats.AvgFileSize = stats.TotalSize / int64(stats.TotalFiles)
	}

	return stats, nil
}

// GetCleanableFilesCount 获取可清理文件数量（排除头像）
func (s *FileStatsService) GetCleanableFilesCount(uploadDir string, retentionDays int) (int, error) {
	if retentionDays <= 0 {
		return 0, nil
	}

	count := 0
	avatarDir := filepath.Join(uploadDir, "avatars")
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	err := filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// 跳过目录和头像文件
		if info.IsDir() || strings.HasPrefix(path, avatarDir) {
			return nil
		}

		// 检查是否过期
		if info.ModTime().Before(cutoffTime) {
			count++
		}

		return nil
	})

	return count, err
}
