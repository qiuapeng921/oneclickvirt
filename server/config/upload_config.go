package config

// UploadConfig 文件上传配置
type UploadConfig struct {
	// 基础配置
	MaxAvatarSize int64  `json:"max_avatar_size" yaml:"max_avatar_size"`
	MaxFileSize   int64  `json:"max_file_size" yaml:"max_file_size"`
	UploadDir     string `json:"upload_dir" yaml:"upload_dir"`

	// 安全配置
	EnableSecurityScan bool     `json:"enable_security_scan" yaml:"enable_security_scan"`
	AllowedTypes       []string `json:"allowed_types" yaml:"allowed_types"`
	DeniedExts         []string `json:"denied_exts" yaml:"denied_exts"`

	// 存储配置
	CleanupInterval int `json:"cleanup_interval" yaml:"cleanup_interval"` // 清理间隔（小时）
	RetentionDays   int `json:"retention_days" yaml:"retention_days"`     // 文件保留天数
}

// DefaultUploadConfig 默认上传配置
var DefaultUploadConfig = UploadConfig{
	MaxAvatarSize:      2 * 1024 * 1024,  // 2MB
	MaxFileSize:        10 * 1024 * 1024, // 10MB
	UploadDir:          "./uploads",
	EnableSecurityScan: true,
	AllowedTypes: []string{
		"image/jpeg",
		"image/png",
		"image/webp",
		"image/gif",
	},
	DeniedExts: []string{
		".exe", ".bat", ".cmd", ".com", ".scr", ".pif", ".msi", ".dll",
		".sh", ".bash", ".zsh", ".fish", ".ps1", ".vbs", ".js", ".jar",
		".php", ".asp", ".jsp", ".py", ".rb", ".pl", ".cgi", ".htaccess",
	},
	CleanupInterval: 24, // 24小时
	RetentionDays:   30, // 30天
}
