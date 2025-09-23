package utils

import (
	"fmt"
	"os"
)

// PathExists 检查路径是否存在
func PathExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err == nil {
		if fi.IsDir() {
			return true, nil
		}
		return false, fmt.Errorf("存在同名文件")
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// EnsureDir 确保目录存在，如果不存在则创建
func EnsureDir(path string) error {
	if exists, err := PathExists(path); err != nil {
		return fmt.Errorf("检查目录失败: %w", err)
	} else if !exists {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}
	}
	return nil
}
