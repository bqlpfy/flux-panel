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

type CreateTunnelReq struct {
	Name          string  `json:"name" binding:"required"`
	InNodeID      int64   `json:"in_node_id" binding:"required"`
	OutNodeID     *int64  `json:"out_node_id"`
	Type          int     `json:"type" binding:"required"`
	Flow          int     `json:"flow" binding:"required"`
	TrafficRatio  float64 `json:"traffic_ratio"`
	InterfaceName string  `json:"interface_name"`
	Protocol      string  `json:"protocol"`
	TcpListenAddr string  `json:"tcp_listen_addr"`
	UdpListenAddr string  `json:"udp_listen_addr"`
}

type UpdateTunnelReq struct {
	ID            int64   `json:"id" binding:"required"`
	Name          string  `json:"name" binding:"required"`
	Flow          int     `json:"flow" binding:"required"`
	TrafficRatio  float64 `json:"traffic_ratio"`
	Protocol      string  `json:"protocol" binding:"required"`
	TcpListenAddr string  `json:"tcp_listen_addr" binding:"required"`
	UdpListenAddr string  `json:"udp_listen_addr" binding:"required"`
	InterfaceName string  `json:"interface_name"`
}

// TunnelListItem 隧道列表展示項
type TunnelListItem struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	IP            string `json:"ip"`
	InNodePortSta int    `json:"inNodePortSta"`
	InNodePortEnd int    `json:"inNodePortEnd"`
	Type          int    `json:"type"`
	Protocol      string `json:"protocol"`
}

// ──────────────────────── Service ────────────────────────

type TunnelService struct {
	DB *gorm.DB
}

// Create 新增隧道（對應 Java TunnelServiceImpl.add）
func (s *TunnelService) Create(req CreateTunnelReq) pkg.R {
	// 1. 校驗入口節點
	var inNode model.Node
	if err := s.DB.First(&inNode, req.InNodeID).Error; err != nil {
		return pkg.ErrMsg("入口节点不存在")
	}

	// 2. 確定出口節點（type=1 端口轉發 -> 出口=入口）
	outNodeID := req.InNodeID
	outIP := inNode.ServerIP
	if req.Type != 1 && req.OutNodeID != nil {
		var outNode model.Node
		if err := s.DB.First(&outNode, *req.OutNodeID).Error; err != nil {
			return pkg.ErrMsg("出口节点不存在")
		}
		outNodeID = *req.OutNodeID
		outIP = outNode.ServerIP
	}

	// 3. 預設值
	if req.TrafficRatio <= 0 {
		req.TrafficRatio = 1.0
	}
	if req.Protocol == "" {
		req.Protocol = "tls"
	}
	if req.TcpListenAddr == "" {
		req.TcpListenAddr = "0.0.0.0"
	}
	if req.UdpListenAddr == "" {
		req.UdpListenAddr = "0.0.0.0"
	}

	now := time.Now().UnixMilli()
	tunnel := model.Tunnel{
		Name:          req.Name,
		TrafficRatio:  req.TrafficRatio,
		InNodeID:      req.InNodeID,
		InIP:          inNode.ServerIP,
		OutNodeID:     outNodeID,
		OutIP:         outIP,
		Type:          req.Type,
		Protocol:      req.Protocol,
		Flow:          req.Flow,
		TcpListenAddr: req.TcpListenAddr,
		UdpListenAddr: req.UdpListenAddr,
		InterfaceName: req.InterfaceName,
		CreatedTime:   now,
		UpdatedTime:   now,
		Status:        1,
	}

	if err := s.DB.Create(&tunnel).Error; err != nil {
		slog.Error("創建隧道失敗", "error", err)
		return pkg.ErrMsg("创建隧道失败")
	}

	return pkg.Ok("隧道创建成功")
}

// List 隧道列表（管理員看全部，普通用戶只看已分配的）
func (s *TunnelService) List(roleID int, userID int64) pkg.R {
	if roleID == 0 {
		return s.adminList()
	}
	return s.userList(userID)
}

func (s *TunnelService) adminList() pkg.R {
	var tunnels []model.Tunnel
	s.DB.Find(&tunnels)

	var items []TunnelListItem
	for _, t := range tunnels {
		var inNode model.Node
		s.DB.First(&inNode, t.InNodeID)
		items = append(items, TunnelListItem{
			ID:            t.ID,
			Name:          t.Name,
			IP:            inNode.ServerIP,
			InNodePortSta: inNode.PortSta,
			InNodePortEnd: inNode.PortEnd,
			Type:          t.Type,
			Protocol:      t.Protocol,
		})
	}
	return pkg.Ok(items)
}

func (s *TunnelService) userList(userID int64) pkg.R {
	var userTunnels []model.UserTunnel
	s.DB.Where("user_id = ? AND status = 1", userID).Find(&userTunnels)

	var items []TunnelListItem
	for _, ut := range userTunnels {
		var t model.Tunnel
		if err := s.DB.First(&t, ut.TunnelID).Error; err != nil {
			continue
		}
		var inNode model.Node
		s.DB.First(&inNode, t.InNodeID)
		items = append(items, TunnelListItem{
			ID:            t.ID,
			Name:          t.Name,
			IP:            inNode.ServerIP,
			InNodePortSta: inNode.PortSta,
			InNodePortEnd: inNode.PortEnd,
			Type:          t.Type,
			Protocol:      t.Protocol,
		})
	}
	return pkg.Ok(items)
}

// Update 更新隧道
func (s *TunnelService) Update(req UpdateTunnelReq) pkg.R {
	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, req.ID).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}

	oldProtocol := tunnel.Protocol

	tunnel.Name = req.Name
	tunnel.Flow = req.Flow
	if req.TrafficRatio > 0 {
		tunnel.TrafficRatio = req.TrafficRatio
	}
	tunnel.Protocol = req.Protocol
	tunnel.TcpListenAddr = req.TcpListenAddr
	tunnel.UdpListenAddr = req.UdpListenAddr
	tunnel.InterfaceName = req.InterfaceName
	tunnel.UpdatedTime = time.Now().UnixMilli()

	if err := s.DB.Save(&tunnel).Error; err != nil {
		slog.Error("更新隧道失敗", "error", err)
		return pkg.ErrMsg("更新隧道失败")
	}

	// 若協議改變，需要更新所有關聯轉發的 Gost 配置（隧道轉發類型）
	if tunnel.Type != 1 && oldProtocol != req.Protocol {
		s.updateRelatedForwards(tunnel)
	}

	return pkg.Ok("更新成功")
}

// updateRelatedForwards 協議變更時重建相關轉發的 chains
func (s *TunnelService) updateRelatedForwards(tunnel model.Tunnel) {
	var forwards []model.Forward
	s.DB.Where("tunnel_id = ? AND status = 1", tunnel.ID).Find(&forwards)

	for _, fw := range forwards {
		name := fmt.Sprintf("fow_%d", fw.ID)
		outAddr := fmt.Sprintf("%s:%d", tunnel.OutIP, *fw.OutPort)

		// 更新 chain
		pkg.UpdateChains(tunnel.InNodeID, name, outAddr, tunnel.Protocol, tunnel.InterfaceName)
		// 更新 remote service
		pkg.UpdateRemoteService(tunnel.OutNodeID, name, *fw.OutPort, fw.RemoteAddr, tunnel.Protocol, fw.Strategy, fw.InterfaceName)
	}
}

// Delete 刪除隧道（級聯刪除轉發、用戶隧道、限速）
func (s *TunnelService) Delete(id int64) pkg.R {
	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, id).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}

	// 1. 刪除所有關聯轉發的 Gost 配置
	var forwards []model.Forward
	s.DB.Where("tunnel_id = ?", id).Find(&forwards)
	for _, fw := range forwards {
		s.deleteForwardGost(fw, tunnel)
	}

	// 2. 刪除關聯限速的 Gost 配置
	var speedLimits []model.SpeedLimit
	s.DB.Where("tunnel_id = ?", id).Find(&speedLimits)
	for _, sl := range speedLimits {
		pkg.DeleteLimiters(tunnel.InNodeID, sl.ID)
	}

	// 3. 級聯刪除資料庫記錄
	tx := s.DB.Begin()
	tx.Where("tunnel_id = ?", id).Delete(&model.Forward{})
	tx.Where("tunnel_id = ?", id).Delete(&model.UserTunnel{})
	tx.Where("tunnel_id = ?", id).Delete(&model.SpeedLimit{})
	tx.Delete(&model.Tunnel{}, id)

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		slog.Error("刪除隧道失敗", "error", err)
		return pkg.ErrMsg("删除隧道失败")
	}

	return pkg.Ok("删除成功")
}

// deleteForwardGost 清理單條轉發的 Gost 配置
func (s *TunnelService) deleteForwardGost(fw model.Forward, tunnel model.Tunnel) {
	name := fmt.Sprintf("fow_%d", fw.ID)

	if tunnel.Type == 1 {
		// 端口轉發
		pkg.DeleteService(tunnel.InNodeID, name)
	} else {
		// 隧道轉發
		pkg.DeleteService(tunnel.InNodeID, name)
		pkg.DeleteChains(tunnel.InNodeID, name)
		if fw.OutPort != nil {
			pkg.DeleteRemoteService(tunnel.OutNodeID, name)
		}
	}
}

// GetByID 取得單一隧道（含節點 IP）
func (s *TunnelService) GetByID(id int64) pkg.R {
	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, id).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}
	return pkg.Ok(tunnel)
}
