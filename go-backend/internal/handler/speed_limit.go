package handler

import (
	"strconv"

	"github.com/lijt/flux-panel/internal/pkg"
	"github.com/lijt/flux-panel/internal/service"

	"github.com/gin-gonic/gin"
)

type SpeedLimitHandler struct {
	Svc *service.SpeedLimitService
}

// Create POST /speed_limit/add
func (h *SpeedLimitHandler) Create(c *gin.Context) {
	var req service.CreateSpeedLimitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.Create(req))
}

// List GET /speed_limit/list
func (h *SpeedLimitHandler) List(c *gin.Context) {
	c.JSON(200, h.Svc.List())
}

// Update PUT /speed_limit/update
func (h *SpeedLimitHandler) Update(c *gin.Context) {
	var req service.UpdateSpeedLimitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.Update(req))
}

// Delete DELETE /speed_limit/delete/:id
func (h *SpeedLimitHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.Delete(id))
}

// GetTunnelSpeedLimits GET /speed_limit/tunnel/:tunnelId
func (h *SpeedLimitHandler) GetTunnelSpeedLimits(c *gin.Context) {
	tunnelID, err := strconv.ParseInt(c.Param("tunnelId"), 10, 64)
	if err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.GetTunnelSpeedLimits(tunnelID))
}
