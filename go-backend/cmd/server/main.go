package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/lijt/flux-panel/internal/config"
	"github.com/lijt/flux-panel/internal/handler"
	"github.com/lijt/flux-panel/internal/middleware"
	"github.com/lijt/flux-panel/internal/service"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// 載入配置
	cfg := config.Load()

	// 配置結構化日誌
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// 連線資料庫
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		slog.Error("資料庫連線失敗", "error", err)
		os.Exit(1)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)

	slog.Info("資料庫連線成功")

	// 初始化 Service
	userSvc := &service.UserService{DB: db, Cfg: cfg}
	nodeSvc := &service.NodeService{DB: db}

	// 初始化 Handler
	userH := &handler.UserHandler{Svc: userSvc}
	nodeH := &handler.NodeHandler{Svc: nodeSvc}
	configH := &handler.ConfigHandler{DB: db}
	captchaH := &handler.CaptchaHandler{DB: db}
	openAPIH := &handler.OpenAPIHandler{DB: db}

	// 設定 Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 全局中間件
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS(cfg.CORSOrigins))

	// 健康檢查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ===== 公開路由（無需認證）=====
	setupPublicRoutes(r, userH, configH, captchaH, openAPIH)

	// ===== 認證路由 =====
	auth := r.Group("")
	auth.Use(middleware.JWTAuth(cfg.JWTSecret))
	setupAuthRoutes(auth, userH)

	// ===== 管理員路由 =====
	admin := r.Group("")
	admin.Use(middleware.JWTAuth(cfg.JWTSecret))
	admin.Use(middleware.AdminOnly())
	setupAdminRoutes(admin, userH, nodeH, configH)

	// 啟動伺服器
	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	slog.Info("伺服器啟動", "addr", addr)
	if err := r.Run(addr); err != nil {
		slog.Error("伺服器啟動失敗", "error", err)
		os.Exit(1)
	}
}

// setupPublicRoutes 公開路由（無需認證）
func setupPublicRoutes(r *gin.Engine, userH *handler.UserHandler, configH *handler.ConfigHandler, captchaH *handler.CaptchaHandler, openAPIH *handler.OpenAPIHandler) {
	user := r.Group("/api/v1/user")
	{
		user.POST("/login", userH.Login)
	}

	captcha := r.Group("/api/v1/captcha")
	{
		captcha.POST("/check", captchaH.Check)
		captcha.POST("/generate", captchaH.Generate)
		captcha.POST("/verify", captchaH.Verify)
	}

	configPub := r.Group("/api/v1/config")
	{
		configPub.POST("/list", configH.List)
		configPub.POST("/get", configH.Get)
	}

	openAPI := r.Group("/api/v1/open_api")
	{
		openAPI.GET("/sub_store", openAPIH.SubStore)
	}
}

// setupAuthRoutes 需登入的路由
func setupAuthRoutes(auth *gin.RouterGroup, userH *handler.UserHandler) {
	user := auth.Group("/api/v1/user")
	{
		user.POST("/package", userH.Package)
		user.POST("/updatePassword", userH.UpdatePassword)
	}
}

// setupAdminRoutes 管理員路由
func setupAdminRoutes(admin *gin.RouterGroup, userH *handler.UserHandler, nodeH *handler.NodeHandler, configH *handler.ConfigHandler) {
	// ---- User (admin) ----
	user := admin.Group("/api/v1/user")
	{
		user.POST("/create", userH.Create)
		user.POST("/list", userH.List)
		user.POST("/update", userH.Update)
		user.POST("/delete", userH.Delete)
		user.POST("/reset", userH.Reset)
	}

	// ---- Node ----
	node := admin.Group("/api/v1/node")
	{
		node.POST("/create", nodeH.Create)
		node.POST("/list", nodeH.List)
		node.POST("/update", nodeH.Update)
		node.POST("/delete", nodeH.Delete)
		node.POST("/install", nodeH.Install)
	}

	// ---- Config (admin write) ----
	cfgAdmin := admin.Group("/api/v1/config")
	{
		cfgAdmin.POST("/update", configH.Update)
		cfgAdmin.POST("/update-single", configH.UpdateSingle)
	}

	// TODO: Phase 3 - Tunnel, Forward, UserTunnel, SpeedLimit routes
}
