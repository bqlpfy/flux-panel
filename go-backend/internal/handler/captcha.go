package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/lijt/flux-panel/internal/model"
	"github.com/lijt/flux-panel/internal/pkg"
	"gorm.io/gorm"
)

type CaptchaHandler struct {
	DB *gorm.DB
}

// Check 檢查驗證碼是否啟用（公開）
func (h *CaptchaHandler) Check(c *gin.Context) {
	var config model.ViteConfig
	if err := h.DB.Where("name = ?", "captcha_enabled").First(&config).Error; err != nil {
		c.JSON(200, pkg.Ok(0))
		return
	}
	if config.Value != "true" {
		c.JSON(200, pkg.Ok(0))
		return
	}
	c.JSON(200, pkg.Ok(1))
}

// Generate 生成驗證碼
// TODO: Phase 4 中整合完整的驗證碼生成邏輯
// 目前使用簡化實作，前端可根據 captcha_enabled=false 跳過驗證碼
func (h *CaptchaHandler) Generate(c *gin.Context) {
	// 簡化回應，待整合驗證碼庫
	c.JSON(200, map[string]interface{}{
		"id":   "",
		"type": "SLIDER",
	})
}

// Verify 驗證驗證碼
// TODO: Phase 4 中整合完整的驗證碼驗證邏輯
func (h *CaptchaHandler) Verify(c *gin.Context) {
	var req struct {
		ID   string      `json:"id"`
		Data interface{} `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, map[string]interface{}{
			"success": false,
			"msg":     "参数错误",
		})
		return
	}

	// 簡化：直接返回成功
	c.JSON(200, map[string]interface{}{
		"success": true,
		"data":    map[string]string{"validToken": req.ID},
	})
}
