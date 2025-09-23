package utils

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"oneclickvirt/global"

	"github.com/pkg/sftp"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type SSHConfig struct {
	Host           string
	Port           int
	Username       string
	Password       string
	ConnectTimeout time.Duration
	ExecuteTimeout time.Duration
}

type SSHClient struct {
	client *ssh.Client
	config SSHConfig
}

func NewSSHClient(config SSHConfig) (*SSHClient, error) {
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = 12 * time.Second // 用户要求的12秒连接等待
	}
	if config.ExecuteTimeout == 0 {
		config.ExecuteTimeout = 300 * time.Second // 5分钟执行超时，足够处理复杂配置
	}
	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         config.ConnectTimeout,
	}

	// 构建连接地址，如果Host已经包含端口则直接使用，否则拼接端口
	var addr string
	if strings.Contains(config.Host, ":") {
		// Host已经包含端口（如 "192.168.1.1:22"），直接使用
		addr = config.Host
	} else {
		// Host不包含端口，拼接端口号
		addr = fmt.Sprintf("%s:%d", config.Host, config.Port)
	}

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	return &SSHClient{
		client: client,
		config: config,
	}, nil
}

func (c *SSHClient) Execute(command string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// 请求PTY以模拟交互式登录shell，确保加载完整的环境变量
	err = session.RequestPty("xterm", 80, 40, ssh.TerminalModes{
		ssh.ECHO:          0,     // 禁用回显
		ssh.TTY_OP_ISPEED: 14400, // 输入速度
		ssh.TTY_OP_OSPEED: 14400, // 输出速度
	})
	if err != nil {
		return "", fmt.Errorf("failed to request PTY: %w", err)
	}

	// 设置环境变量来确保PATH正确加载，避免使用bash -l -c的转义问题
	// 这种方式更安全，不需要处理复杂的命令转义
	envCommand := fmt.Sprintf("source /etc/profile 2>/dev/null || true; source ~/.bashrc 2>/dev/null || true; source ~/.bash_profile 2>/dev/null || true; export PATH=$PATH:/usr/local/bin:/snap/bin:/usr/sbin:/sbin; %s", command)

	// 创建一个通道来处理命令执行的超时
	done := make(chan struct{})
	var output []byte
	var execErr error

	go func() {
		output, execErr = session.CombinedOutput(envCommand)
		close(done)
	}()

	// 等待命令完成或超时
	select {
	case <-done:
		if execErr != nil {
			// 记录执行失败的详细信息，包括原始命令和转换后的命令
			if global.APP_LOG != nil {
				global.APP_LOG.Debug("SSH命令执行失败",
					zap.String("original_command", command),
					zap.String("env_wrapped_command", envCommand),
					zap.Error(execErr),
					zap.String("output", string(output)))
			}
			return string(output), fmt.Errorf("command execution failed: %w", execErr)
		}
		return string(output), nil
	case <-time.After(c.config.ExecuteTimeout):
		session.Signal(ssh.SIGKILL) // 强制终止会话
		return "", fmt.Errorf("command execution timeout after %v", c.config.ExecuteTimeout)
	}
}

func (c *SSHClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// ExecuteWithLogging 执行命令并记录详细的调试信息，用于排查复杂命令的执行问题
func (c *SSHClient) ExecuteWithLogging(command string, logPrefix string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// 请求PTY以模拟交互式登录shell，确保加载完整的环境变量
	err = session.RequestPty("xterm", 80, 40, ssh.TerminalModes{
		ssh.ECHO:          0,     // 禁用回显
		ssh.TTY_OP_ISPEED: 14400, // 输入速度
		ssh.TTY_OP_OSPEED: 14400, // 输出速度
	})
	if err != nil {
		return "", fmt.Errorf("failed to request PTY: %w", err)
	}

	// 设置环境变量来确保PATH正确加载
	envCommand := fmt.Sprintf("source /etc/profile 2>/dev/null || true; source ~/.bashrc 2>/dev/null || true; source ~/.bash_profile 2>/dev/null || true; export PATH=$PATH:/usr/local/bin:/snap/bin:/usr/sbin:/sbin; %s", command)

	// 记录执行前的信息
	if global.APP_LOG != nil {
		global.APP_LOG.Debug("SSH命令执行开始",
			zap.String("log_prefix", logPrefix),
			zap.String("original_command", command),
			zap.String("wrapped_command", envCommand))
	}

	// 创建一个通道来处理命令执行的超时
	done := make(chan struct{})
	var output []byte
	var execErr error

	go func() {
		output, execErr = session.CombinedOutput(envCommand)
		close(done)
	}()

	// 等待命令完成或超时
	select {
	case <-done:
		if execErr != nil {
			// 记录执行失败的详细信息
			if global.APP_LOG != nil {
				global.APP_LOG.Error("SSH命令执行失败",
					zap.String("log_prefix", logPrefix),
					zap.String("original_command", command),
					zap.String("wrapped_command", envCommand),
					zap.Error(execErr),
					zap.String("output", string(output)))
			}
			return string(output), fmt.Errorf("command execution failed: %w", execErr)
		}
		if global.APP_LOG != nil {
			global.APP_LOG.Debug("SSH命令执行成功",
				zap.String("log_prefix", logPrefix),
				zap.String("original_command", command),
				zap.Int("output_length", len(output)))
		}
		return string(output), nil
	case <-time.After(c.config.ExecuteTimeout):
		session.Signal(ssh.SIGKILL) // 强制终止会话
		if global.APP_LOG != nil {
			global.APP_LOG.Warn("SSH命令执行超时",
				zap.String("log_prefix", logPrefix),
				zap.String("original_command", command),
				zap.Duration("timeout", c.config.ExecuteTimeout))
		}
		return "", fmt.Errorf("command execution timeout after %v", c.config.ExecuteTimeout)
	}
}

// UploadContent 上传内容到远程服务器指定路径
func (c *SSHClient) UploadContent(content, remotePath string, perm os.FileMode) error {
	// 创建SFTP客户端
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// 创建远程文件的目录（如果不存在）
	remoteDir := remotePath
	if lastSlash := strings.LastIndex(remotePath, "/"); lastSlash != -1 {
		remoteDir = remotePath[:lastSlash]
	}

	if remoteDir != "" && remoteDir != remotePath {
		err = sftpClient.MkdirAll(remoteDir)
		if err != nil {
			return fmt.Errorf("failed to create remote directory %s: %w", remoteDir, err)
		}
	}

	// 创建远程文件
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file %s: %w", remotePath, err)
	}
	defer remoteFile.Close()

	// 写入内容
	_, err = io.WriteString(remoteFile, content)
	if err != nil {
		return fmt.Errorf("failed to write content to remote file: %w", err)
	}

	// 设置文件权限
	err = sftpClient.Chmod(remotePath, perm)
	if err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}
