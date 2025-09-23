package incus

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// convertMemoryFormat 转换内存格式为Incus支持的格式
func convertMemoryFormat(memory string) string {
	if memory == "" {
		return ""
	}

	// 检查是否已经是正确的格式（以 iB 结尾）
	if strings.HasSuffix(memory, "iB") {
		return memory
	}

	// 处理MB格式
	if strings.HasSuffix(memory, "M") || strings.HasSuffix(memory, "MB") || strings.HasSuffix(memory, "m") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(memory, "MB"), "M"), "m")
		if num, err := strconv.Atoi(numStr); err == nil {
			return fmt.Sprintf("%dMiB", num)
		}
	}

	// 处理GB格式
	if strings.HasSuffix(memory, "G") || strings.HasSuffix(memory, "GB") || strings.HasSuffix(memory, "g") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(memory, "GB"), "G"), "g")
		if num, err := strconv.Atoi(numStr); err == nil {
			return fmt.Sprintf("%dGiB", num)
		}
	}

	// 如果没有单位，假设是MB
	if num, err := strconv.Atoi(memory); err == nil {
		return fmt.Sprintf("%dMiB", num)
	}

	// 默认返回原值
	return memory
}

// convertDiskFormat 转换磁盘格式为Incus支持的格式
func convertDiskFormat(disk string) string {
	if disk == "" {
		return ""
	}

	// 检查是否已经是正确的格式（以 iB 或 B 结尾）
	if strings.HasSuffix(disk, "iB") || strings.HasSuffix(disk, "B") {
		return disk
	}

	// 处理MB格式
	if strings.HasSuffix(disk, "M") || strings.HasSuffix(disk, "MB") || strings.HasSuffix(disk, "m") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(disk, "MB"), "M"), "m")
		if num, err := strconv.Atoi(numStr); err == nil {
			return fmt.Sprintf("%dMiB", num)
		}
	}

	// 处理GB格式
	if strings.HasSuffix(disk, "G") || strings.HasSuffix(disk, "GB") || strings.HasSuffix(disk, "g") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(disk, "GB"), "G"), "g")
		if num, err := strconv.Atoi(numStr); err == nil {
			return fmt.Sprintf("%dGiB", num)
		}
	}

	// 如果没有单位，假设是MB
	if num, err := strconv.Atoi(disk); err == nil {
		return fmt.Sprintf("%dMiB", num)
	}

	// 默认返回原值
	return disk
}

// isValidInstanceName 检查实例名称是否有效
func isValidInstanceName(name string) bool {
	if name == "" {
		return false
	}

	// Incus实例名称规则：
	// - 长度不超过63个字符
	// - 只能包含字母、数字、连字符和下划线
	// - 不能以连字符开头或结尾
	// - 不能包含连续的连字符
	if len(name) > 63 {
		return false
	}

	// 使用正则表达式验证格式
	pattern := `^[a-zA-Z0-9]([a-zA-Z0-9\-_]*[a-zA-Z0-9])?$`
	matched, err := regexp.MatchString(pattern, name)
	if err != nil || !matched {
		return false
	}

	// 检查是否包含连续的连字符
	if strings.Contains(name, "--") {
		return false
	}

	return true
}

// m 辅助函数，返回两个整数中的较小值
func m(a, b int) int {
	if a < b {
		return a
	}
	return b
}
