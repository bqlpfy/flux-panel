package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中間件
func CORS(allowedOrigins string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if allowedOrigins == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			origins := strings.Split(allowedOrigins, ",")
			for _, o := range origins {
				if strings.TrimSpace(o) == origin {
					c.Header("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "3600")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
