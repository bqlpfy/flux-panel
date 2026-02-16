package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/lijt/flux-panel/internal/pkg"
)

// Recovery 全局 panic 恢復中間件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", err,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"stack", string(debug.Stack()),
				)
				// 不洩漏內部錯誤（與 Java GlobalExceptionHandler 一致）
				c.JSON(http.StatusInternalServerError, pkg.Err(500, "服务器内部错误"))
				c.Abort()
			}
		}()
		c.Next()
	}
}
