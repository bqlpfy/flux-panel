package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/lijt/flux-panel/internal/pkg"
	"github.com/lijt/flux-panel/internal/service"
)

type NodeHandler struct {
	Svc *service.NodeService
}

func (h *NodeHandler) Create(c *gin.Context) {
	var req service.CreateNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.CreateNode(req))
}

func (h *NodeHandler) List(c *gin.Context) {
	c.JSON(200, h.Svc.GetAllNodes())
}

func (h *NodeHandler) Update(c *gin.Context) {
	var req service.UpdateNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.UpdateNode(req))
}

func (h *NodeHandler) Delete(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	id, ok := toInt64(params["id"])
	if !ok {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.DeleteNode(id))
}

func (h *NodeHandler) Install(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	id, ok := toInt64(params["id"])
	if !ok {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.GetInstallCommand(id))
}
