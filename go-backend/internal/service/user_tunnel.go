package service

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/lijt/flux-panel/internal/model"
	"github.com/lijt/flux-panel/internal/pkg"

	"gorm.io/gorm"
)

// ──────────────────────── Request DTOs ────────────────────────

type CreateUserTunnelReq struct {
	UserID        int   `json:"user_id" binding:"required"`
	TunnelID      int   `json:"tunnel_id" binding:"required"`
	Flow          int64 `json:"flow" binding:"required"`
	Num           int   `json:"num" binding:"required"`
	FlowResetTime int64 `json:"flow_reset_time" binding:"required"`
	ExpTime       int64 `json:"exp_time" binding:"required"`
	SpeedID       *int  `json:"speed_id"`
}

type UpdateUserTunnelReq struct {
	ID            int   `json:"id" binding:"required"`
	Flow          int64 `json:"flow" binding:"required"`
	Num           int   `json:"num" binding:"required"`
	FlowResetTime int64 `json:"flow_reset_time" binding:"required"`
	ExpTime       int64 `json:"exp_time" binding:"required"`
	Status        int   `json:"status" binding:"required"`
	SpeedID       *int  `json:"speed_id"`
}

type UserTunnelQueryReq struct {
	UserID int `json:"user_id" binding:"required"`
}

// UserTunnelDetail 用戶隧道權限展示（附帶隧道名+用戶名）
type UserTunnelDetail struct {
	model.UserTunnel
	TunnelName string `json:"tunnel_name"`
	UserName   string `json:"user_name"`
	SpeedName  string `json:"speed_name,omitempty"`
}

// ──────────────────────── Service ────────────────────────

type UserTunnelService struct {
	DB *gorm.DB
}

// Create 分配用戶隧道權限
func (s *UserTunnelService) Create(req CreateUserTunnelReq) pkg.R {
	// 校驗用戶
	var user model.User
	if err := s.DB.First(&user, req.UserID).Error; err != nil {
		return pkg.ErrMsg("用户不存在")
	}

	// 校驗隧道
	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, req.TunnelID).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}

	// 檢查是否已分配
	var count int64
	s.DB.Model(&model.UserTunnel{}).Where("user_id = ? AND tunnel_id = ?", req.UserID, req.TunnelID).Count(&count)
	if count > 0 {
		return pkg.ErrMsg("该用户已拥有此隧道权限")
	}

	ut := model.UserTunnel{
		UserID:        req.UserID,
		TunnelID:      req.TunnelID,
		SpeedID:       req.SpeedID,
		Num:           req.Num,
		Flow:          req.Flow,
		FlowResetTime: req.FlowResetTime,
		ExpTime:       req.ExpTime,
		Status:        1,
	}

	if err := s.DB.Create(&ut).Error; err != nil {
		slog.Error("分配用戶隧道權限失敗", "error", err)
		return pkg.ErrMsg("分配权限失败")
	}

	return pkg.Ok("权限分配成功")
}

// List 查詢用戶的隧道權限列表
func (s *UserTunnelService) List(userID int) pkg.R {
	var uts []model.UserTunnel
	s.DB.Where("user_id = ?", userID).Find(&uts)

	var result []UserTunnelDetail
	for _, ut := range uts {
		detail := UserTunnelDetail{UserTunnel: ut}

		var tunnel model.Tunnel
		if s.DB.First(&tunnel, ut.TunnelID).Error == nil {
			detail.TunnelName = tunnel.Name
		}

		var user model.User
		if s.DB.First(&user, ut.UserID).Error == nil {
			detail.UserName = user.User
		}

		if ut.SpeedID != nil && *ut.SpeedID > 0 {
			var sl model.SpeedLimit
			if s.DB.First(&sl, *ut.SpeedID).Error == nil {
				detail.SpeedName = sl.Name
			}
		}

		result = append(result, detail)
	}
	return pkg.Ok(result)
}

// Update 更新用戶隧道權限
func (s *UserTunnelService) Update(req UpdateUserTunnelReq) pkg.R {
	var ut model.UserTunnel
	if err := s.DB.First(&ut, req.ID).Error; err != nil {
		return pkg.ErrMsg("用户隧道权限不存在")
	}

	oldSpeedID := ut.SpeedID
	oldStatus := ut.Status

	ut.Flow = req.Flow
	ut.Num = req.Num
	ut.FlowResetTime = req.FlowResetTime
	ut.ExpTime = req.ExpTime
	ut.Status = req.Status
	ut.SpeedID = req.SpeedID

	if err := s.DB.Save(&ut).Error; err != nil {
		slog.Error("更新用戶隧道權限失敗", "error", err)
		return pkg.ErrMsg("更新权限失败")
	}

	// 限速改變時需要更新關聯轉發的 Gost limiter
	if !intPtrEqual(oldSpeedID, req.SpeedID) {
		s.updateForwardLimiters(ut)
	}

	// 狀態變化：禁用時暫停所有轉發，啟用時恢復
	if oldStatus != req.Status {
		s.toggleForwards(ut, req.Status == 1)
	}

	return pkg.Ok("更新成功")
}

// Delete 移除用戶隧道權限（同時刪除關聯轉發）
func (s *UserTunnelService) Delete(id int) pkg.R {
	var ut model.UserTunnel
	if err := s.DB.First(&ut, id).Error; err != nil {
		return pkg.ErrMsg("用户隧道权限不存在")
	}

	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, ut.TunnelID).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}

	// 刪除該用戶在此隧道下的所有轉發及其 Gost 配置
	var forwards []model.Forward
	s.DB.Where("user_id = ? AND tunnel_id = ?", ut.UserID, ut.TunnelID).Find(&forwards)
	for _, fw := range forwards {
		name := fmt.Sprintf("fow_%d", fw.ID)
		if tunnel.Type == 1 {
			pkg.DeleteService(tunnel.InNodeID, name)
		} else {
			pkg.DeleteService(tunnel.InNodeID, name)
			pkg.DeleteChains(tunnel.InNodeID, name)
			if fw.OutPort != nil {
				pkg.DeleteRemoteService(tunnel.OutNodeID, name)
			}
		}
	}
	s.DB.Where("user_id = ? AND tunnel_id = ?", ut.UserID, ut.TunnelID).Delete(&model.Forward{})

	// 刪除用戶隧道權限
	if err := s.DB.Delete(&model.UserTunnel{}, id).Error; err != nil {
		slog.Error("刪除用戶隧道權限失敗", "error", err)
		return pkg.ErrMsg("删除权限失败")
	}

	return pkg.Ok("删除成功")
}

// ResetFlow 重置用戶隧道流量
func (s *UserTunnelService) ResetFlow(id int) pkg.R {
	result := s.DB.Model(&model.UserTunnel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"in_flow":  0,
		"out_flow": 0,
	})
	if result.Error != nil {
		return pkg.ErrMsg("重置流量失败")
	}
	return pkg.Ok("流量重置成功")
}

// ──────────────────────── 內部方法 ────────────────────────

func (s *UserTunnelService) updateForwardLimiters(ut model.UserTunnel) {
	var tunnel model.Tunnel
	if s.DB.First(&tunnel, ut.TunnelID).Error != nil {
		return
	}

	var forwards []model.Forward
	s.DB.Where("user_id = ? AND tunnel_id = ? AND status = 1", ut.UserID, ut.TunnelID).Find(&forwards)

	for _, fw := range forwards {
		name := fmt.Sprintf("fow_%d", fw.ID)

		// 先刪除舊配置再重新添加（含新的 limiter）
		if tunnel.Type == 1 {
			pkg.DeleteService(tunnel.InNodeID, name)
		} else {
			pkg.DeleteService(tunnel.InNodeID, name)
			pkg.DeleteChains(tunnel.InNodeID, name)
			if fw.OutPort != nil {
				pkg.DeleteRemoteService(tunnel.OutNodeID, name)
			}
		}

		// 重新添加
		var limiter *int
		if ut.SpeedID != nil && *ut.SpeedID > 0 {
			limiter = ut.SpeedID
		}

		if tunnel.Type == 1 {
			pkg.AddService(tunnel.InNodeID, name, fw.InPort, limiter, fw.RemoteAddr, tunnel.Type, tunnel, fw.Strategy, fw.InterfaceName)
		} else {
			outAddr := fmt.Sprintf("%s:%d", tunnel.OutIP, *fw.OutPort)
			pkg.AddChains(tunnel.InNodeID, name, outAddr, tunnel.Protocol, tunnel.InterfaceName)
			pkg.AddService(tunnel.InNodeID, name, fw.InPort, limiter, outAddr, tunnel.Type, tunnel, fw.Strategy, fw.InterfaceName)
			pkg.AddRemoteService(tunnel.OutNodeID, name, *fw.OutPort, fw.RemoteAddr, tunnel.Protocol, fw.Strategy, fw.InterfaceName)
		}
	}
}

func (s *UserTunnelService) toggleForwards(ut model.UserTunnel, enable bool) {
	var tunnel model.Tunnel
	if s.DB.First(&tunnel, ut.TunnelID).Error != nil {
		return
	}

	var forwards []model.Forward
	s.DB.Where("user_id = ? AND tunnel_id = ?", ut.UserID, ut.TunnelID).Find(&forwards)

	newStatus := 0
	if enable {
		newStatus = 1
	}

	for _, fw := range forwards {
		name := fmt.Sprintf("fow_%d", fw.ID)
		if enable {
			pkg.ResumeService(tunnel.InNodeID, name)
			if tunnel.Type != 1 {
				pkg.ResumeRemoteService(tunnel.OutNodeID, name)
			}
		} else {
			pkg.PauseService(tunnel.InNodeID, name)
			if tunnel.Type != 1 {
				pkg.PauseRemoteService(tunnel.OutNodeID, name)
			}
		}
		s.DB.Model(&fw).Updates(map[string]interface{}{
			"status":       newStatus,
			"updated_time": time.Now().UnixMilli(),
		})
	}
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
