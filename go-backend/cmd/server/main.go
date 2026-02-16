package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/lijt/flux-panel/internal/config"
	"github.com/lijt/flux-panel/internal/middleware"
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
	setupPublicRoutes(r, db, cfg)

	// ===== 認證路由 =====
	auth := r.Group("")
	auth.Use(middleware.JWTAuth(cfg.JWTSecret))
	setupAuthRoutes(auth, db, cfg)

	// ===== 管理員路由 =====
	admin := r.Group("")
	admin.Use(middleware.JWTAuth(cfg.JWTSecret))
	admin.Use(middleware.AdminOnly())
	setupAdminRoutes(admin, db, cfg)

	// 啟動伺服器
	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	slog.Info("伺服器啟動", "addr", addr)
	if err := r.Run(addr); err != nil {
		slog.Error("伺服器啟動失敗", "error", err)
		os.Exit(1)
	}
}

// setupPublicRoutes 公開路由（無需認證）
func setupPublicRoutes(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	// POST /api/v1/user/login
	// POST /api/v1/captcha/get
	// POST /api/v1/captcha/check
	// GET  /api/v1/captcha/get
	// POST /api/v1/open_api/sub
	// POST /flow/report
	// GET  /api/v1/config/get  (公開)

	// TODO: Phase 2 中實作各 handler
}

// setupAuthRoutes 需登入的路由
func setupAuthRoutes(auth *gin.RouterGroup, db *gorm.DB, cfg *config.Config) {
	// GET  /api/v1/user/info
	// POST /api/v1/user/updatePwd
	// GET  /api/v1/user/package

	// TODO: Phase 2 中實作
}

// setupAdminRoutes 管理員路由
func setupAdminRoutes(admin *gin.RouterGroup, db *gorm.DB, cfg *config.Config) {
	// ---- User (admin) ----
	// GET    /api/v1/user/list
	// POST   /api/v1/user/add
	// POST   /api/v1/user/update
	// DELETE /api/v1/user/delete/:id
	// POST   /api/v1/user/updateStatus

	// ---- Node ----
	// GET    /api/v1/node/list
	// POST   /api/v1/node/add
	// POST   /api/v1/node/update
	// DELETE /api/v1/node/delete/:id
	// GET    /api/v1/node/install_script

	// ---- Tunnel ----
	// GET    /api/v1/tunnel/list
	// POST   /api/v1/tunnel/add
	// POST   /api/v1/tunnel/update
	// DELETE /api/v1/tunnel/delete/:id
	// ... 更多隧道端點

	// ---- Forward ----
	// GET    /api/v1/forward/list
	// POST   /api/v1/forward/add
	// POST   /api/v1/forward/update
	// DELETE /api/v1/forward/delete/:id
	// ... 更多轉發端點

	// ---- SpeedLimit ----
	// GET    /api/v1/speed-limit/list
	// POST   /api/v1/speed-limit/add
	// POST   /api/v1/speed-limit/update
	// DELETE /api/v1/speed-limit/delete/:id

	// ---- Config ----
	// POST   /api/v1/config/add
	// POST   /api/v1/config/update
	// DELETE /api/v1/config/delete/:id

	// TODO: Phase 2-3 中實作
}
