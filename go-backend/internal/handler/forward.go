package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lijt/flux-panel/internal/middleware"
	"github.com/lijt/flux-panel/internal/pkg"
	"github.com/lijt/flux-panel/internal/service"
)

type ForwardHandler struct {
	Svc *service.ForwardService
}

// Create POST /forward/add
func (h *ForwardHandler) Create(c *gin.Context) {
	var req service.CreateForwardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	userID, _ := middleware.GetCurrentUserID(c)
	roleID, _ := middleware.GetCurrentRoleID(c)
	userName, _ := middleware.GetCurrentUserName(c)
	c.JSON(200, h.Svc.Create(req, userID, roleID, userName))
}

// List GET /forward/list
func (h *ForwardHandler) List(c *gin.Context) {
	userID, _ := middleware.GetCurrentUserID(c)
	roleID, _ := middleware.GetCurrentRoleID(c)
	c.JSON(200, h.Svc.List(userID, roleID))
}

// Update PUT /forward/update
func (h *ForwardHandler) Update(c *gin.Context) {
	var req service.UpdateForwardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	roleID, _ := middleware.GetCurrentRoleID(c)
	userID, _ := middleware.GetCurrentUserID(c)
	c.JSON(200, h.Svc.Update(req, roleID, userID))
}

// Delete DELETE /forward/delete/:id
func (h *ForwardHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	roleID, _ := middleware.GetCurrentRoleID(c)
	userID, _ := middleware.GetCurrentUserID(c)
	c.JSON(200, h.Svc.Delete(id, roleID, userID))
}

// Pause PUT /forward/pause/:id
func (h *ForwardHandler) Pause(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	roleID, _ := middleware.GetCurrentRoleID(c)
	userID, _ := middleware.GetCurrentUserID(c)
	c.JSON(200, h.Svc.Pause(id, roleID, userID))
}

// Resume PUT /forward/resume/:id
func (h *ForwardHandler) Resume(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	roleID, _ := middleware.GetCurrentRoleID(c)
	userID, _ := middleware.GetCurrentUserID(c)
	c.JSON(200, h.Svc.Resume(id, roleID, userID))
}

// Diagnose GET /forward/diagnose/:id
func (h *ForwardHandler) Diagnose(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.Diagnose(id))
}

// UpdateOrder PUT /forward/updateOrder
func (h *ForwardHandler) UpdateOrder(c *gin.Context) {
	var ids []int64
	if err := c.ShouldBindJSON(&ids); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.UpdateOrder(ids))
}
