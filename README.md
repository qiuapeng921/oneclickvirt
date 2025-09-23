# OneClickVirt 虚拟化管理平台

一个可扩展的支持 lxd/incus/docker/proxmoxve 的通用虚拟化控制平台。

## 技术栈

### 后端
- Gin Web框架
- GORM ORM
- MySQL数据库
- JWT认证

### 前端
- Vue 3
- Vite
- Element Plus
- Pinia
- Vue Router

## 快速开始

### 环境要求
- Go 1.24.5
- Node.js 19+
- npm/yarn

### 访问地址

- 前端地址：http://localhost:8080
- 后端API：http://localhost:8888
- API文档：http://localhost:8888/swagger/index.html

### 首次使用

1. 访问 http://localhost:8080 会自动跳转到系统初始化页面
2. 设置管理员账户信息
3. 完成初始化后即可正常使用

## 默认账户

系统初始化后会创建以下默认账户：

- **管理员账户**：admin / Admin123!@#
- **普通用户**：testuser / TestUser123!@#

> **注意**：首次登录后请立即修改默认密码

## 配置文件

主要配置文件位于 `server/config.yaml`，包含：
- 数据库配置
- 服务器端口设置
- JWT密钥配置
- 虚拟化提供商配置

## 部署说明

### 生产环境部署

1. 构建前端
```bash
cd web
npm run build
```

2. 构建后端
```bash
cd server
go build -o oneclickvirt main.go
```

3. 配置反向代理（推荐使用Nginx）
4. 设置环境变量 `GIN_MODE=release`

## 赞助

<img src="https://github.com/user-attachments/assets/78bab50f-9e65-42ef-bad5-9430799afc1b" width="400" />

https://ko-fi.com/spiritlhl

## 演示图

[](./.back/1.png)

[](./.back/2.png)

[](./.back/3.png)

[](./.back/4.png)