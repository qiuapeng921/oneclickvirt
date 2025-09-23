package docker

import (
	"crypto/md5"
	"fmt"
	"path/filepath"
	"strings"

	"oneclickvirt/global"
	"oneclickvirt/utils"

	"go.uber.org/zap"
)

// downloadImageToRemote 在远程服务器上下载镜像
func (d *DockerProvider) downloadImageToRemote(imageURL, imageName, providerCountry, architecture string) (string, error) {
	// 根据provider类型确定远程下载目录
	downloadDir := "/usr/local/bin/docker_ct_images"

	// 在远程服务器上创建下载目录
	cmd := fmt.Sprintf("mkdir -p %s", downloadDir)
	_, err := d.sshClient.Execute(cmd)
	if err != nil {
		return "", fmt.Errorf("创建远程下载目录失败: %w", err)
	}

	// 生成文件名
	fileName := d.generateRemoteFileName(imageName, imageURL, architecture)
	remotePath := filepath.Join(downloadDir, fileName)

	// 检查远程文件是否已存在
	if d.isRemoteFileValid(remotePath) {
		global.APP_LOG.Info("远程镜像文件已存在且完整，跳过下载",
			zap.String("imageName", imageName),
			zap.String("remotePath", remotePath))
		return remotePath, nil
	}

	// 确定下载URL
	downloadURL := d.getDownloadURL(imageURL, providerCountry)

	global.APP_LOG.Info("开始在远程服务器下载镜像",
		zap.String("imageName", imageName),
		zap.String("downloadURL", downloadURL),
		zap.String("remotePath", remotePath))

	// 在远程服务器上下载文件
	if err := d.downloadFileToRemote(downloadURL, remotePath); err != nil {
		// 下载失败，删除不完整的文件
		d.removeRemoteFile(remotePath)
		return "", fmt.Errorf("远程下载镜像失败: %w", err)
	}

	global.APP_LOG.Info("远程镜像下载完成",
		zap.String("imageName", imageName),
		zap.String("remotePath", remotePath))

	return remotePath, nil
}

// cleanupRemoteImage 清理远程镜像文件
func (d *DockerProvider) cleanupRemoteImage(imageName, imageURL, architecture string) error {
	downloadDir := "/usr/local/bin/docker_ct_images"
	fileName := d.generateRemoteFileName(imageName, imageURL, architecture)
	remotePath := filepath.Join(downloadDir, fileName)

	return d.removeRemoteFile(remotePath)
}

// generateRemoteFileName 生成远程文件名
func (d *DockerProvider) generateRemoteFileName(imageName, imageURL, architecture string) string {
	// 组合字符串
	combined := fmt.Sprintf("%s_%s_%s", imageName, imageURL, architecture)

	// 计算MD5
	hasher := md5.New()
	hasher.Write([]byte(combined))
	md5Hash := fmt.Sprintf("%x", hasher.Sum(nil))

	// 使用镜像名称和MD5的前8位作为文件名，保持可读性
	safeName := strings.ReplaceAll(imageName, "/", "_")
	safeName = strings.ReplaceAll(safeName, ":", "_")

	return fmt.Sprintf("%s_%s.tar", safeName, md5Hash[:8])
}

// isRemoteFileValid 检查远程文件是否存在且完整
func (d *DockerProvider) isRemoteFileValid(remotePath string) bool {
	// 检查文件是否存在
	cmd := fmt.Sprintf("test -f %s", remotePath)
	_, err := d.sshClient.Execute(cmd)
	if err != nil {
		return false
	}

	// 检查文件大小是否大于0
	cmd = fmt.Sprintf("test -s %s", remotePath)
	_, err = d.sshClient.Execute(cmd)
	if err != nil {
		return false
	}

	return true
}

// removeRemoteFile 删除远程文件
func (d *DockerProvider) removeRemoteFile(remotePath string) error {
	cmd := fmt.Sprintf("rm -f %s", remotePath)
	_, err := d.sshClient.Execute(cmd)
	return err
}

// getDownloadURL 确定下载URL
func (d *DockerProvider) getDownloadURL(originalURL, providerCountry string) string {
	// 默认随机尝试CDN，不再限制地区
	if cdnURL := d.getCDNURL(originalURL); cdnURL != "" {
		return cdnURL
	}
	return originalURL
}

// getCDNURL 获取CDN URL - 随机测试可用性
func (d *DockerProvider) getCDNURL(originalURL string) string {
	cdnEndpoints := utils.GetCDNEndpoints()

	// 随机测试CDN端点，找到第一个可用的就使用
	for _, endpoint := range cdnEndpoints {
		cdnURL := endpoint + originalURL
		// 测试CDN可用性 - 使用更短的超时时间进行快速检测
		testCmd := fmt.Sprintf("curl -4 -sL -k --max-time 6 '%s' | head -c 1 >/dev/null 2>&1", cdnURL)
		if _, err := d.sshClient.Execute(testCmd); err == nil {
			global.APP_LOG.Info("找到可用CDN，使用CDN下载镜像",
				zap.String("originalURL", utils.TruncateString(originalURL, 100)),
				zap.String("cdnURL", utils.TruncateString(cdnURL, 100)),
				zap.String("cdnEndpoint", endpoint))
			return cdnURL
		}
		// 短暂延迟避免过于频繁的请求
		// 注意：在Go中我们不能直接sleep，但SSH执行会有自然的延迟
	}

	global.APP_LOG.Info("未找到可用CDN，使用原始URL",
		zap.String("originalURL", utils.TruncateString(originalURL, 100)))
	return ""
}

// downloadFileToRemote 在远程服务器上下载文件
func (d *DockerProvider) downloadFileToRemote(url, remotePath string) error {
	// 使用curl在远程服务器上下载文件
	tmpPath := remotePath + ".tmp"

	// 构建curl命令，包含超时和重试机制
	curlCmd := fmt.Sprintf(
		"curl -L --max-time 1800 --retry 3 --retry-delay 5 -o %s %s && mv %s %s",
		tmpPath, url, tmpPath, remotePath,
	)

	global.APP_LOG.Info("执行远程下载命令",
		zap.String("command", utils.TruncateString(curlCmd, 200)))

	output, err := d.sshClient.Execute(curlCmd)
	if err != nil {
		global.APP_LOG.Error("远程下载失败",
			zap.String("url", utils.TruncateString(url, 100)),
			zap.String("remotePath", remotePath),
			zap.String("output", utils.TruncateString(output, 500)),
			zap.Error(err))

		// 清理临时文件
		d.sshClient.Execute(fmt.Sprintf("rm -f %s", tmpPath))
		return fmt.Errorf("远程下载失败: %w", err)
	}

	global.APP_LOG.Info("远程下载成功",
		zap.String("url", utils.TruncateString(url, 100)),
		zap.String("remotePath", remotePath))

	return nil
}

// ensureSSHScriptsAvailable 确保SSH脚本文件在远程服务器上可用
func (d *DockerProvider) ensureSSHScriptsAvailable(providerCountry string) error {
	scriptsDir := "/usr/local/bin"
	scripts := []string{"ssh_bash.sh", "ssh_sh.sh"}

	// 检查脚本是否都存在
	allExist := true
	for _, script := range scripts {
		scriptPath := filepath.Join(scriptsDir, script)
		if !d.isRemoteFileValid(scriptPath) {
			allExist = false
			global.APP_LOG.Info("SSH脚本文件不存在或无效",
				zap.String("scriptPath", scriptPath))
			break
		}
	}

	if allExist {
		global.APP_LOG.Info("SSH脚本文件都已存在且有效")
		return nil
	}

	// 下载缺失的脚本
	global.APP_LOG.Info("开始下载SSH脚本文件")

	for _, script := range scripts {
		scriptPath := filepath.Join(scriptsDir, script)

		// 如果脚本已存在且有效，跳过
		if d.isRemoteFileValid(scriptPath) {
			global.APP_LOG.Info("SSH脚本已存在，跳过下载",
				zap.String("script", script))
			continue
		}

		// 构建下载URL
		baseURL := "https://raw.githubusercontent.com/oneclickvirt/docker/main/scripts/" + script
		downloadURL := d.getSSHScriptDownloadURL(baseURL, providerCountry)

		global.APP_LOG.Info("开始下载SSH脚本",
			zap.String("script", script),
			zap.String("downloadURL", downloadURL),
			zap.String("scriptPath", scriptPath))

		// 下载脚本文件
		if err := d.downloadFileToRemote(downloadURL, scriptPath); err != nil {
			global.APP_LOG.Error("下载SSH脚本失败",
				zap.String("script", script),
				zap.Error(err))
			return fmt.Errorf("下载SSH脚本 %s 失败: %w", script, err)
		}

		// 设置执行权限
		chmodCmd := fmt.Sprintf("chmod +x %s", scriptPath)
		if _, err := d.sshClient.Execute(chmodCmd); err != nil {
			global.APP_LOG.Error("设置SSH脚本执行权限失败",
				zap.String("script", script),
				zap.Error(err))
			return fmt.Errorf("设置SSH脚本 %s 执行权限失败: %w", script, err)
		}

		// 使用dos2unix处理脚本格式（如果可用）
		dos2unixCmd := fmt.Sprintf("command -v dos2unix >/dev/null 2>&1 && dos2unix %s || true", scriptPath)
		d.sshClient.Execute(dos2unixCmd)

		global.APP_LOG.Info("SSH脚本下载并设置完成",
			zap.String("script", script),
			zap.String("scriptPath", scriptPath))
	}

	global.APP_LOG.Info("所有SSH脚本文件下载完成")
	return nil
}

// getSSHScriptDownloadURL 获取SSH脚本下载URL，支持CDN
func (d *DockerProvider) getSSHScriptDownloadURL(originalURL, providerCountry string) string {
	// 如果是中国地区，尝试使用CDN
	if providerCountry == "CN" || providerCountry == "cn" {
		if cdnURL := d.getSSHScriptCDNURL(originalURL); cdnURL != "" {
			// 测试CDN可用性
			testCmd := fmt.Sprintf("curl -s -I --max-time 5 '%s' | head -n 1 | grep -q '200'", cdnURL)
			if _, err := d.sshClient.Execute(testCmd); err == nil {
				global.APP_LOG.Info("使用CDN下载SSH脚本",
					zap.String("cdnURL", cdnURL))
				return cdnURL
			}
		}
	}
	return originalURL
}

// getSSHScriptCDNURL 获取SSH脚本CDN URL
func (d *DockerProvider) getSSHScriptCDNURL(originalURL string) string {
	cdnEndpoints := utils.GetCDNEndpoints()

	// 直接在原始URL前加CDN前缀
	// 原始URL格式: https://raw.githubusercontent.com/oneclickvirt/docker/main/scripts/ssh_bash.sh
	// CDN URL格式: https://cdn0.spiritlhl.top/https://raw.githubusercontent.com/oneclickvirt/docker/main/scripts/ssh_bash.sh
	for _, endpoint := range cdnEndpoints {
		cdnURL := endpoint + originalURL
		// 测试CDN可用性
		testCmd := fmt.Sprintf("curl -s -I --max-time 5 '%s' | head -n 1 | grep -q '200'", cdnURL)
		if _, err := d.sshClient.Execute(testCmd); err == nil {
			return cdnURL
		}
	}
	return ""
}
