package handler

import (
	"strconv"

	"github.com/lijt/flux-panel/internal/pkg"
	"github.com/lijt/flux-panel/internal/service"

	"github.com/gin-gonic/gin"
)

type UserTunnelHandler struct {
	Svc *service.UserTunnelService
}

// Create POST /user_tunnel/add
func (h *UserTunnelHandler) Create(c *gin.Context) {
	var req service.CreateUserTunnelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.Create(req))
}

// List POST /user_tunnel/list
func (h *UserTunnelHandler) List(c *gin.Context) {
	var req service.UserTunnelQueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.List(req.UserID))
}

// Update PUT /user_tunnel/update
func (h *UserTunnelHandler) Update(c *gin.Context) {
	var req service.UpdateUserTunnelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.Update(req))
}

// Delete DELETE /user_tunnel/delete/:id
func (h *UserTunnelHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.Delete(id))
}

// ResetFlow PUT /user_tunnel/reset_flow/:id
func (h *UserTunnelHandler) ResetFlow(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.ResetFlow(id))
}
