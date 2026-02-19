package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/lijt/flux-panel/internal/model"
	"github.com/lijt/flux-panel/internal/pkg"
	"gorm.io/gorm"
)

type OpenAPIHandler struct {
	DB *gorm.DB
}

// SubStore 子訂閱查詢（公開，GET 方式）
func (h *OpenAPIHandler) SubStore(c *gin.Context) {
	user := c.Query("user")
	pwd := c.Query("pwd")
	tunnel := c.DefaultQuery("tunnel", "-1")

	if user == "" {
		c.JSON(200, pkg.ErrMsg("用户不能为空"))
		return
	}
	if pwd == "" {
		c.JSON(200, pkg.ErrMsg("密码不能为空"))
		return
	}

	// 驗證用戶身份
	var userInfo model.User
	if err := h.DB.Where("user = ?", user).First(&userInfo).Error; err != nil {
		c.JSON(200, pkg.ErrMsg("鉴权失败"))
		return
	}

	pwdMD5 := pkg.MD5(pwd)
	if pwdMD5 != userInfo.Pwd {
		c.JSON(200, pkg.ErrMsg("鉴权失败"))
		return
	}

	const GIGA = 1024 * 1024 * 1024

	var headerValue string
	if tunnel == "-1" {
		headerValue = fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%d",
			userInfo.OutFlow, userInfo.InFlow, userInfo.Flow*GIGA, userInfo.ExpTime/1000)
	} else {
		var tunnelInfo model.UserTunnel
		if err := h.DB.Where("id = ?", tunnel).First(&tunnelInfo).Error; err != nil {
			c.JSON(200, pkg.ErrMsg("隧道不存在"))
			return
		}
		if fmt.Sprintf("%d", tunnelInfo.UserID) != fmt.Sprintf("%d", userInfo.ID) {
			c.JSON(200, pkg.ErrMsg("隧道不存在"))
			return
		}
		headerValue = fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%d",
			tunnelInfo.OutFlow, tunnelInfo.InFlow, tunnelInfo.Flow*GIGA, tunnelInfo.ExpTime/1000)
	}

	c.Header("subscription-userinfo", headerValue)
	c.String(200, headerValue)
}
