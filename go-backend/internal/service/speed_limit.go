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

type CreateSpeedLimitReq struct {
	Name       string `json:"name" binding:"required"`
	Speed      int    `json:"speed" binding:"required,min=1"`
	TunnelID   int64  `json:"tunnel_id" binding:"required"`
	TunnelName string `json:"tunnel_name" binding:"required"`
}

type UpdateSpeedLimitReq struct {
	ID         int64  `json:"id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Speed      int    `json:"speed" binding:"required,min=1"`
	TunnelID   int64  `json:"tunnel_id" binding:"required"`
	TunnelName string `json:"tunnel_name" binding:"required"`
}

// ──────────────────────── Service ────────────────────────

type SpeedLimitService struct {
	DB *gorm.DB
}

// Create 新增限速規則
func (s *SpeedLimitService) Create(req CreateSpeedLimitReq) pkg.R {
	// 校驗隧道存在
	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, req.TunnelID).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}

	now := time.Now().UnixMilli()
	sl := model.SpeedLimit{
		Name:        req.Name,
		Speed:       req.Speed,
		TunnelID:    req.TunnelID,
		TunnelName:  req.TunnelName,
		CreatedTime: now,
		UpdatedTime: now,
		Status:      1,
	}

	if err := s.DB.Create(&sl).Error; err != nil {
		slog.Error("創建限速規則失敗", "error", err)
		return pkg.ErrMsg("创建限速规则失败")
	}

	// 在入口節點上添加 limiter
	pkg.AddLimiters(tunnel.InNodeID, sl.ID, fmt.Sprintf("%d", req.Speed))

	return pkg.Ok("限速规则创建成功")
}

// List 限速規則列表
func (s *SpeedLimitService) List() pkg.R {
	var list []model.SpeedLimit
	s.DB.Order("id DESC").Find(&list)
	return pkg.Ok(list)
}

// Update 更新限速規則
func (s *SpeedLimitService) Update(req UpdateSpeedLimitReq) pkg.R {
	var sl model.SpeedLimit
	if err := s.DB.First(&sl, req.ID).Error; err != nil {
		return pkg.ErrMsg("限速规则不存在")
	}

	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, req.TunnelID).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}

	sl.Name = req.Name
	sl.Speed = req.Speed
	sl.TunnelID = req.TunnelID
	sl.TunnelName = req.TunnelName
	sl.UpdatedTime = time.Now().UnixMilli()

	if err := s.DB.Save(&sl).Error; err != nil {
		slog.Error("更新限速規則失敗", "error", err)
		return pkg.ErrMsg("更新限速规则失败")
	}

	// 更新 Gost limiter
	pkg.UpdateLimiters(tunnel.InNodeID, sl.ID, fmt.Sprintf("%d", req.Speed))

	return pkg.Ok("更新成功")
}

// Delete 刪除限速規則
func (s *SpeedLimitService) Delete(id int64) pkg.R {
	var sl model.SpeedLimit
	if err := s.DB.First(&sl, id).Error; err != nil {
		return pkg.ErrMsg("限速规则不存在")
	}

	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, sl.TunnelID).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}

	// 檢查是否有用戶隧道在使用此限速
	var count int64
	s.DB.Model(&model.UserTunnel{}).Where("speed_id = ?", id).Count(&count)
	if count > 0 {
		// 清除引用
		s.DB.Model(&model.UserTunnel{}).Where("speed_id = ?", id).Update("speed_id", nil)
	}

	// 刪除 Gost limiter
	pkg.DeleteLimiters(tunnel.InNodeID, sl.ID)

	// 刪除資料庫記錄
	if err := s.DB.Delete(&model.SpeedLimit{}, id).Error; err != nil {
		slog.Error("刪除限速規則失敗", "error", err)
		return pkg.ErrMsg("删除限速规则失败")
	}

	return pkg.Ok("删除成功")
}

// GetTunnelSpeedLimits 取得指定隧道的限速規則
func (s *SpeedLimitService) GetTunnelSpeedLimits(tunnelID int64) pkg.R {
	var list []model.SpeedLimit
	s.DB.Where("tunnel_id = ?", tunnelID).Find(&list)
	return pkg.Ok(list)
}
