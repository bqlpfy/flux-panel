package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lijt/flux-panel/internal/middleware"
	"github.com/lijt/flux-panel/internal/pkg"
	"github.com/lijt/flux-panel/internal/service"
)

type UserHandler struct {
	Svc *service.UserService
}

func (h *UserHandler) Login(c *gin.Context) {
	var req service.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.Login(req))
}

func (h *UserHandler) Create(c *gin.Context) {
	var req service.CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.CreateUser(req))
}

func (h *UserHandler) List(c *gin.Context) {
	c.JSON(200, h.Svc.GetAllUsers())
}

func (h *UserHandler) Update(c *gin.Context) {
	var req service.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.UpdateUser(req))
}

func (h *UserHandler) Delete(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	idVal, ok := params["id"]
	if !ok {
		pkg.ResErr(c, "参数错误")
		return
	}
	id, _ := toInt64(idVal)
	c.JSON(200, h.Svc.DeleteUser(id))
}

func (h *UserHandler) Package(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		pkg.ResErr(c, "用户未登录")
		return
	}
	c.JSON(200, h.Svc.GetUserPackageInfo(userID))
}

func (h *UserHandler) UpdatePassword(c *gin.Context) {
	var req service.ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		pkg.ResErr(c, "用户未登录")
		return
	}
	c.JSON(200, h.Svc.UpdatePassword(userID, req))
}

func (h *UserHandler) Reset(c *gin.Context) {
	var req service.ResetFlowReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ResErr(c, "参数错误")
		return
	}
	c.JSON(200, h.Svc.ResetFlow(req))
}

// toInt64 通用類型轉換輔助
func toInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case float64:
		return int64(val), true
	case string:
		i, err := strconv.ParseInt(val, 10, 64)
		return i, err == nil
	case int64:
		return val, true
	case int:
		return int64(val), true
	default:
		return 0, false
	}
}
