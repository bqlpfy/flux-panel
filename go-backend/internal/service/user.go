package service

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/lijt/flux-panel/internal/config"
	"github.com/lijt/flux-panel/internal/model"
	"github.com/lijt/flux-panel/internal/pkg"
	"gorm.io/gorm"
)

// UserService 用戶業務邏輯
type UserService struct {
	DB  *gorm.DB
	Cfg *config.Config
}

// --- 請求 DTO ---

type LoginReq struct {
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	CaptchaID string `json:"captchaId"`
}

type CreateUserReq struct {
	User          string `json:"user" binding:"required"`
	Pwd           string `json:"pwd" binding:"required"`
	Flow          int64  `json:"flow" binding:"required"`
	Num           int    `json:"num" binding:"required"`
	ExpTime       int64  `json:"exp_time" binding:"required"`
	FlowResetTime int64  `json:"flow_reset_time" binding:"required"`
	Status        *int   `json:"status"`
}

type UpdateUserReq struct {
	ID            int64  `json:"id" binding:"required"`
	User          string `json:"user" binding:"required"`
	Pwd           string `json:"pwd"`
	Flow          int64  `json:"flow" binding:"required"`
	Num           int    `json:"num" binding:"required"`
	ExpTime       int64  `json:"exp_time" binding:"required"`
	FlowResetTime int64  `json:"flow_reset_time" binding:"required"`
	Status        *int   `json:"status"`
}

type ChangePasswordReq struct {
	NewUsername     string `json:"newUsername" binding:"required"`
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
}

type ResetFlowReq struct {
	ID   int `json:"id" binding:"required"`
	Type int `json:"type" binding:"required"`
}

// --- 業務方法 ---

func (s *UserService) Login(req LoginReq) pkg.R {
	// 1. 驗證驗證碼（若啟用）
	var captchaConfig model.ViteConfig
	if err := s.DB.Where("name = ?", "captcha_enabled").First(&captchaConfig).Error; err == nil {
		if captchaConfig.Value == "true" && req.CaptchaID == "" {
			return pkg.ErrMsg("验证码校验失败")
		}
		// TODO: Phase 4 中實作驗證碼二次校驗
	}

	// 2. 查找用戶
	var user model.User
	if err := s.DB.Where("user = ?", req.Username).First(&user).Error; err != nil {
		return pkg.ErrMsg("账号或密码错误")
	}

	// 3. 驗證密碼（使用原始 MD5，非 salt 版本）
	if user.Pwd != pkg.MD5(req.Password) {
		return pkg.ErrMsg("账号或密码错误")
	}

	// 4. 檢查狀態
	if user.Status == 0 {
		return pkg.ErrMsg("账户停用")
	}

	// 5. 生成 JWT
	token, err := pkg.GenerateToken(user.ID, user.User, user.RoleID, s.Cfg.JWTSecret, s.Cfg.JWTExpireHours)
	if err != nil {
		slog.Error("生成 token 失敗", "error", err)
		return pkg.ErrMsg("登录失败")
	}

	// 6. 檢查是否使用默認憑據
	requireChange := req.Username == "admin_user" || req.Password == "admin_user"

	return pkg.Ok(map[string]interface{}{
		"token":                 token,
		"name":                  user.User,
		"role_id":               user.RoleID,
		"requirePasswordChange": requireChange,
	})
}

func (s *UserService) CreateUser(req CreateUserReq) pkg.R {
	// 核查用戶名唯一性
	var count int64
	s.DB.Model(&model.User{}).Where("user = ?", req.User).Count(&count)
	if count > 0 {
		return pkg.ErrMsg("用户名已存在")
	}

	now := time.Now().UnixMilli()
	status := 1
	if req.Status != nil {
		status = *req.Status
	}

	user := model.User{
		User:          req.User,
		Pwd:           pkg.MD5(req.Pwd),
		RoleID:        1, // 普通用戶
		Flow:          req.Flow,
		Num:           req.Num,
		ExpTime:       req.ExpTime,
		FlowResetTime: req.FlowResetTime,
		Status:        status,
		CreatedTime:   now,
		UpdatedTime:   now,
	}

	if err := s.DB.Create(&user).Error; err != nil {
		slog.Error("創建用戶失敗", "error", err)
		return pkg.ErrMsg("用户创建失败")
	}

	return pkg.Ok("用户创建成功")
}

func (s *UserService) GetAllUsers() pkg.R {
	var users []model.User
	s.DB.Where("role_id != ?", 0).Find(&users)
	return pkg.Ok(users)
}

func (s *UserService) UpdateUser(req UpdateUserReq) pkg.R {
	var user model.User
	if err := s.DB.First(&user, req.ID).Error; err != nil {
		return pkg.ErrMsg("用户不存在")
	}
	if user.RoleID == 0 {
		return pkg.ErrMsg("不能修改管理员用户信息")
	}

	// 驗證用戶名唯一性
	var count int64
	s.DB.Model(&model.User{}).Where("user = ? AND id != ?", req.User, req.ID).Count(&count)
	if count > 0 {
		return pkg.ErrMsg("用户名已被其他用户使用")
	}

	updates := map[string]interface{}{
		"user":            req.User,
		"flow":            req.Flow,
		"num":             req.Num,
		"exp_time":        req.ExpTime,
		"flow_reset_time": req.FlowResetTime,
		"updated_time":    time.Now().UnixMilli(),
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Pwd != "" {
		updates["pwd"] = pkg.MD5(req.Pwd)
	}

	if err := s.DB.Model(&model.User{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		return pkg.ErrMsg("用户更新失败")
	}

	return pkg.Ok("用户更新成功")
}

func (s *UserService) DeleteUser(id int64) pkg.R {
	var user model.User
	if err := s.DB.First(&user, id).Error; err != nil {
		return pkg.ErrMsg("用户不存在")
	}
	if user.RoleID == 0 {
		return pkg.ErrMsg("不能删除管理员用户")
	}

	// 級聯刪除：轉發、用戶隧道、流量統計
	tx := s.DB.Begin()
	tx.Where("user_id = ?", id).Delete(&model.Forward{})
	tx.Where("user_id = ?", id).Delete(&model.UserTunnel{})
	tx.Where("user_id = ?", id).Delete(&model.StatisticsFlow{})
	tx.Delete(&model.User{}, id)

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		slog.Error("刪除用戶失敗", "error", err, "userID", id)
		return pkg.ErrMsg("删除用户时发生错误")
	}

	return pkg.Ok("用户及关联数据删除成功")
}

func (s *UserService) GetUserPackageInfo(userID int64) pkg.R {
	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return pkg.ErrMsg("用户不存在")
	}

	// 取用戶隧道列表
	var userTunnels []model.UserTunnel
	s.DB.Where("user_id = ?", userID).Find(&userTunnels)

	type TunnelDetail struct {
		model.UserTunnel
		TunnelName string `json:"tunnel_name"`
		SpeedName  string `json:"speed_name"`
	}

	var details []TunnelDetail
	for _, ut := range userTunnels {
		d := TunnelDetail{UserTunnel: ut}
		var tunnel model.Tunnel
		if s.DB.First(&tunnel, ut.TunnelID).Error == nil {
			d.TunnelName = tunnel.Name
		}
		if ut.SpeedID != nil {
			var speed model.SpeedLimit
			if s.DB.First(&speed, *ut.SpeedID).Error == nil {
				d.SpeedName = speed.Name
			}
		}
		details = append(details, d)
	}

	return pkg.Ok(map[string]interface{}{
		"user":    user,
		"tunnels": details,
	})
}

func (s *UserService) UpdatePassword(userID int64, req ChangePasswordReq) pkg.R {
	if req.NewPassword != req.ConfirmPassword {
		return pkg.ErrMsg("新密码和确认密码不匹配")
	}

	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return pkg.ErrMsg("用户不存在")
	}

	if user.Pwd != pkg.MD5(req.CurrentPassword) {
		return pkg.ErrMsg("当前密码错误")
	}

	// 驗證新用戶名唯一性
	if user.User != req.NewUsername {
		var count int64
		s.DB.Model(&model.User{}).Where("user = ? AND id != ?", req.NewUsername, user.ID).Count(&count)
		if count > 0 {
			return pkg.ErrMsg("用户名已被其他用户使用")
		}
	}

	s.DB.Model(&user).Updates(map[string]interface{}{
		"user":         req.NewUsername,
		"pwd":          pkg.MD5(req.NewPassword),
		"updated_time": time.Now().UnixMilli(),
	})

	return pkg.Ok("账号密码修改成功")
}

func (s *UserService) ResetFlow(req ResetFlowReq) pkg.R {
	if req.Type == 1 { // 重置用戶流量
		result := s.DB.Model(&model.User{}).Where("id = ?", req.ID).Updates(map[string]interface{}{
			"in_flow":  0,
			"out_flow": 0,
		})
		if result.RowsAffected == 0 {
			return pkg.ErrMsg("用户不存在")
		}
	} else { // 重置隧道流量
		result := s.DB.Model(&model.UserTunnel{}).Where("id = ?", req.ID).Updates(map[string]interface{}{
			"in_flow":  0,
			"out_flow": 0,
		})
		if result.RowsAffected == 0 {
			return pkg.ErrMsg("隧道不存在")
		}
	}
	return pkg.OkMsg()
}

// GetUserByID 取得用戶（供其他 service 使用）
func (s *UserService) GetUserByID(id int64) (*model.User, error) {
	var user model.User
	if err := s.DB.First(&user, id).Error; err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}
	return &user, nil
}

// GetUserIDFromToken 從 token 取 userID（helper for handler）
func GetUserIDFromToken(tokenStr string) (int64, error) {
	return pkg.GetUserIDFromToken(tokenStr)
}

func GetUserIDStr(sub string) int64 {
	id, _ := strconv.ParseInt(sub, 10, 64)
	return id
}
