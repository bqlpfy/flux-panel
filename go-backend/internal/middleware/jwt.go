package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lijt/flux-panel/internal/pkg"
)

// JWTAuth JWT 認證中間件
func JWTAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, pkg.Err(401, "未登录"))
			c.Abort()
			return
		}

		// 移除可能的 "Bearer " 前綴
		token = strings.TrimPrefix(token, "Bearer ")

		claims, err := pkg.ValidateToken(token, jwtSecret)
		if err != nil {
			slog.Debug("JWT 驗證失敗", "error", err)
			c.JSON(http.StatusUnauthorized, pkg.Err(401, "登录已过期，请重新登录"))
			c.Abort()
			return
		}

		// 把用戶資訊存入 context
		c.Set("userID", claims.Sub)
		c.Set("userName", claims.Name)
		c.Set("roleID", claims.RoleID)
		c.Set("token", token)

		c.Next()
	}
}
