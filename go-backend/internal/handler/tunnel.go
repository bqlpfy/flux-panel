package handler

import (
	"strconv"

	"github.com/lijt/flux-panel/internal/middleware"
	"github.com/lijt/flux-panel/internal/pkg"
	"github.com/lijt/flux-panel/internal/service"

	"github.com/gin-gonic/gin"
)

type TunnelHandler struct {
	Svc *service.TunnelService
}

// Create POST /tunnel/add
func (h *TunnelHandler) Create(c *gin.Context) {
	var req service.CreateTunnelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.Create(req))
}

// List GET /tunnel/list
func (h *TunnelHandler) List(c *gin.Context) {
	roleID, _ := middleware.GetCurrentRoleID(c)
	userID, _ := middleware.GetCurrentUserID(c)
	c.JSON(200, h.Svc.List(roleID, userID))
}

// Update PUT /tunnel/update
func (h *TunnelHandler) Update(c *gin.Context) {
	var req service.UpdateTunnelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.Update(req))
}

// Delete DELETE /tunnel/delete/:id
func (h *TunnelHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.Delete(id))
}

// GetByID GET /tunnel/:id
func (h *TunnelHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.GetByID(id))
}
