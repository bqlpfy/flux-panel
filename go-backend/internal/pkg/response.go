package pkg

import (
	"time"

	"github.com/gin-gonic/gin"
)

// R 統一回應格式，與 Java 版 R 類完全一致
type R struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Ts   int64       `json:"ts"`
	Data interface{} `json:"data"`
}

// Ok 成功回應（帶資料）
func Ok(data interface{}) R {
	return R{
		Code: 0,
		Msg:  "操作成功",
		Ts:   time.Now().UnixMilli(),
		Data: data,
	}
}

// OkMsg 成功回應（無資料）
func OkMsg() R {
	return R{
		Code: 0,
		Msg:  "操作成功",
		Ts:   time.Now().UnixMilli(),
	}
}

// Err 錯誤回應（帶自定義錯誤碼）
func Err(code int, msg string) R {
	return R{
		Code: code,
		Msg:  msg,
		Ts:   time.Now().UnixMilli(),
	}
}

// ErrMsg 錯誤回應（預設錯誤碼 -1）
func ErrMsg(msg string) R {
	return R{
		Code: -1,
		Msg:  msg,
		Ts:   time.Now().UnixMilli(),
	}
}

// ErrDefault 預設錯誤回應
func ErrDefault() R {
	return R{
		Code: -1,
		Msg:  "请求失败",
		Ts:   time.Now().UnixMilli(),
	}
}

// JSON 快捷方法：寫入 gin.Context
func ResOk(c *gin.Context, data interface{}) {
	c.JSON(200, Ok(data))
}

func ResOkMsg(c *gin.Context) {
	c.JSON(200, OkMsg())
}

func ResErr(c *gin.Context, msg string) {
	c.JSON(200, ErrMsg(msg))
}

func ResErrCode(c *gin.Context, code int, msg string) {
	c.JSON(200, Err(code, msg))
}
