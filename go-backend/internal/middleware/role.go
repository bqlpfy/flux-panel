package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lijt/flux-panel/internal/pkg"
)

// AdminOnly 管理員權限中間件（role_id == 0 為管理員）
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleID, exists := c.Get("roleID")
		if !exists {
			c.JSON(http.StatusForbidden, pkg.Err(403, "无权限"))
			c.Abort()
			return
		}

		if roleID.(int) != 0 {
			c.JSON(http.StatusForbidden, pkg.Err(403, "无权限，需要管理员权限"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCurrentUserID 從 context 取當前用戶 ID
func GetCurrentUserID(c *gin.Context) (int64, bool) {
	sub, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	id, err := strconv.ParseInt(sub.(string), 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// GetCurrentRoleID 從 context 取當前角色 ID
func GetCurrentRoleID(c *gin.Context) (int, bool) {
	roleID, exists := c.Get("roleID")
	if !exists {
		return 0, false
	}
	return roleID.(int), true
}

// GetCurrentUserName 從 context 取當前用戶名
func GetCurrentUserName(c *gin.Context) (string, bool) {
	name, exists := c.Get("userName")
	if !exists {
		return "", false
	}
	return name.(string), true
}
