package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lijt/flux-panel/internal/model"
	"github.com/lijt/flux-panel/internal/pkg"
	"gorm.io/gorm"
)

type ConfigHandler struct {
	DB *gorm.DB
}

// List 取得所有配置（公開）
func (h *ConfigHandler) List(c *gin.Context) {
	var configs []model.ViteConfig
	h.DB.Find(&configs)

	configMap := make(map[string]string)
	for _, cfg := range configs {
		configMap[cfg.Name] = cfg.Value
	}
	c.JSON(200, pkg.Ok(configMap))
}

// Get 取得單一配置（公開）
func (h *ConfigHandler) Get(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	name, _ := params["name"].(string)
	if name == "" {
		pkg.ResErr(c, "配置名称不能为空")
		return
	}

	var config model.ViteConfig
	if err := h.DB.Where("name = ?", name).First(&config).Error; err != nil {
		pkg.ResErr(c, "配置不存在")
		return
	}
	c.JSON(200, pkg.Ok(config))
}

// Update 批量更新配置（管理員）
func (h *ConfigHandler) Update(c *gin.Context) {
	var configMap map[string]string
	if err := c.ShouldBindJSON(&configMap); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	if len(configMap) == 0 {
		pkg.ResErr(c, "配置数据不能为空")
		return
	}

	for name, value := range configMap {
		if name == "" {
			continue
		}
		h.upsertConfig(name, value)
	}
	c.JSON(200, pkg.Ok("配置更新成功"))
}

// UpdateSingle 更新單一配置（管理員）
func (h *ConfigHandler) UpdateSingle(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	name, _ := params["name"].(string)
	value, _ := params["value"].(string)
	if name == "" {
		pkg.ResErr(c, "配置名称不能为空")
		return
	}
	if value == "" {
		pkg.ResErr(c, "配置值不能为空")
		return
	}

	h.upsertConfig(name, value)
	c.JSON(200, pkg.Ok("配置更新成功"))
}

func (h *ConfigHandler) upsertConfig(name, value string) {
	var existing model.ViteConfig
	now := time.Now().UnixMilli()

	if err := h.DB.Where("name = ?", name).First(&existing).Error; err == nil {
		// 更新
		h.DB.Model(&existing).Updates(map[string]interface{}{
			"value": value,
			"time":  now,
		})
	} else {
		// 創建
		h.DB.Create(&model.ViteConfig{
			Name:  name,
			Value: value,
			Time:  now,
		})
	}
}
