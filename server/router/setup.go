package router

import (
	"log"
	"net/http"
	"oneclickvirt/api/v1/public"
	"oneclickvirt/middleware"
	authModel "oneclickvirt/model/auth"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// isAPIPath 检查路径是否为API路径
func isAPIPath(path string) bool {
	return strings.HasPrefix(path, "/api/") ||
		strings.HasPrefix(path, "/swagger/") ||
		path == "/health"
}

// SetupRouter 统一的路由设置入口
func SetupRouter() *gin.Engine {
	Router := gin.Default()

	// CORS配置
	Router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length", "Authorization"},
		AllowCredentials: true,
	}))

	// 全局中间件
	Router.Use(middleware.ErrorHandler())
	Router.Use(middleware.InputValidator())

	// 健康检查 - 使用public包中的标准健康检查
	Router.GET("/health", public.HealthCheck)

	// Swagger文档路由
	Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API路由组
	ApiGroup := Router.Group("/api")
	{
		// 健康检查也在API路径下，保持与前端一致
		ApiGroup.GET("/health", public.HealthCheck)

		// 公开访问路由
		PublicGroup := ApiGroup.Group("")
		PublicGroup.Use(middleware.RequireAuth(authModel.AuthLevelPublic))
		{
			PublicGroup.GET("/ping", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "pong"})
			})

			// 初始化各模块路由
			InitAuthRouter(PublicGroup)   // 认证相关路由
			InitPublicRouter(PublicGroup) // 公开路由
			InitOAuth2Router(PublicGroup) // OAuth2路由
		}

		// 配置路由
		InitConfigRouter(ApiGroup)

		// 用户路由
		InitUserRouter(ApiGroup)

		// 管理员路由
		InitAdminRouter(ApiGroup)

		// 资源和Provider路由
		InitResourceRouter(ApiGroup)
		InitProviderRouter(ApiGroup)
	}

	// 设置静态文件路由（如果启用了嵌入模式）
	if err := setupStaticRoutes(Router); err != nil {
		log.Printf("[ERROR] 设置静态文件路由失败: %v\n", err)
		// 返回错误但不影响API服务启动
	}

	return Router
}
