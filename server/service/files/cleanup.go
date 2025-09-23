package files

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"oneclickvirt/global"

	"go.uber.org/zap"
)

// FileCleanupService 文件清理服务
type FileCleanupService struct{}

// CleanupExpiredFiles 清理过期文件（排除头像文件）
func (s *FileCleanupService) CleanupExpiredFiles(uploadDir string, retentionDays int) error {
	if retentionDays <= 0 {
		return nil // 不清理
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	avatarDir := filepath.Join(uploadDir, "avatars")

	return filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			global.APP_LOG.Warn("清理文件时遇到错误", zap.String("path", path), zap.Error(err))
			return nil // 继续处理其他文件
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 跳过头像文件（头像文件不过期）
		if strings.HasPrefix(path, avatarDir) {
			return nil
		}

		// 检查文件修改时间
		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(path); err != nil {
				global.APP_LOG.Warn("删除过期文件失败",
					zap.String("path", path),
					zap.Time("mod_time", info.ModTime()),
					zap.Error(err))
			} else {
				global.APP_LOG.Info("已删除过期文件",
					zap.String("path", path),
					zap.Time("mod_time", info.ModTime()))
			}
		}

		return nil
	})
}

// CleanupEmptyDirectories 清理空目录
func (s *FileCleanupService) CleanupEmptyDirectories(uploadDir string) error {
	return filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// 只处理目录
		if !info.IsDir() || path == uploadDir {
			return nil
		}

		// 检查目录是否为空
		if isEmpty, err := s.isDirEmpty(path); err == nil && isEmpty {
			if err := os.Remove(path); err != nil {
				global.APP_LOG.Warn("删除空目录失败", zap.String("path", path), zap.Error(err))
			} else {
				global.APP_LOG.Info("已删除空目录", zap.String("path", path))
			}
		}

		return nil
	})
}

// isDirEmpty 检查目录是否为空
func (s *FileCleanupService) isDirEmpty(dirPath string) (bool, error) {
	f, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err != nil {
		return true, nil // 目录为空
	}
	return false, nil
}

// StartCleanupScheduler 启动清理调度器
func (s *FileCleanupService) StartCleanupScheduler(uploadDir string, intervalHours, retentionDays int) {
	if intervalHours <= 0 || retentionDays <= 0 {
		return
	}

	ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)

	go func() {
		for range ticker.C {
			global.APP_LOG.Info("开始定期文件清理",
				zap.String("upload_dir", uploadDir),
				zap.Int("retention_days", retentionDays))

			// 清理过期文件
			if err := s.CleanupExpiredFiles(uploadDir, retentionDays); err != nil {
				global.APP_LOG.Error("清理过期文件失败", zap.Error(err))
			}

			// 清理空目录
			if err := s.CleanupEmptyDirectories(uploadDir); err != nil {
				global.APP_LOG.Error("清理空目录失败", zap.Error(err))
			}

			global.APP_LOG.Info("定期文件清理完成")
		}
	}()
}
