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

type CreateForwardReq struct {
	Name          string `json:"name" binding:"required"`
	TunnelID      int    `json:"tunnel_id" binding:"required"`
	RemoteAddr    string `json:"remote_addr" binding:"required"`
	Strategy      string `json:"strategy"`
	InPort        *int   `json:"in_port"`
	InterfaceName string `json:"interface_name"`
}

type UpdateForwardReq struct {
	ID            int64  `json:"id" binding:"required"`
	UserID        int    `json:"user_id" binding:"required"`
	Name          string `json:"name" binding:"required"`
	TunnelID      int    `json:"tunnel_id" binding:"required"`
	RemoteAddr    string `json:"remote_addr" binding:"required"`
	Strategy      string `json:"strategy"`
	InPort        *int   `json:"in_port"`
	InterfaceName string `json:"interface_name"`
}

// ForwardWithTunnel 轉發列表展示（含隧道信息）
type ForwardWithTunnel struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	InPort        int    `json:"in_port"`
	RemoteAddr    string `json:"remote_addr"`
	Status        int    `json:"status"`
	CreatedTime   int64  `json:"created_time"`
	UpdatedTime   int64  `json:"updated_time"`
	TunnelName    string `json:"tunnel_name"`
	InIP          string `json:"in_ip"`
	UserName      string `json:"user_name"`
	UserID        int    `json:"user_id"`
	TunnelID      int    `json:"tunnel_id"`
	InFlow        int64  `json:"in_flow"`
	OutFlow       int64  `json:"out_flow"`
	Strategy      string `json:"strategy"`
	Inx           int    `json:"inx"`
	InterfaceName string `json:"interface_name"`
}

// ──────────────────────── Service ────────────────────────

type ForwardService struct {
	DB *gorm.DB
}

// Create 新增轉發（對應 Java ForwardServiceImpl.add）
func (s *ForwardService) Create(req CreateForwardReq, userID int64, roleID int, userName string) pkg.R {
	// 1. 校驗隧道
	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, req.TunnelID).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}

	// 2. 非管理員需校驗用戶隧道權限
	if roleID != 0 {
		if msg := s.checkUserTunnelPermission(userID, req.TunnelID); msg != "" {
			return pkg.ErrMsg(msg)
		}
	}

	// 3. 分配入口端口
	inPort := 0
	if req.InPort != nil {
		inPort = *req.InPort
		// 校驗端口範圍
		inNode := s.getNode(tunnel.InNodeID)
		if inNode == nil {
			return pkg.ErrMsg("入口节点不存在")
		}
		if inPort < inNode.PortSta || inPort > inNode.PortEnd {
			return pkg.ErrMsg(fmt.Sprintf("端口必须在 %d-%d 范围内", inNode.PortSta, inNode.PortEnd))
		}
		// 校驗端口是否已被佔用
		var count int64
		s.DB.Model(&model.Forward{}).Where("tunnel_id = ? AND in_port = ? AND status != -1", req.TunnelID, inPort).Count(&count)
		if count > 0 {
			return pkg.ErrMsg("该端口已被占用")
		}
	} else {
		// 自動分配端口
		allocated := s.allocatePort(tunnel)
		if allocated == 0 {
			return pkg.ErrMsg("没有可用端口")
		}
		inPort = allocated
	}

	// 4. 隧道轉發需要分配出口端口
	var outPort *int
	if tunnel.Type != 1 {
		op := s.allocateOutPort(tunnel)
		if op == 0 {
			return pkg.ErrMsg("出口节点没有可用端口")
		}
		outPort = &op
	}

	now := time.Now().UnixMilli()
	forward := model.Forward{
		UserID:        int(userID),
		UserName:      userName,
		Name:          req.Name,
		TunnelID:      req.TunnelID,
		InPort:        inPort,
		OutPort:       outPort,
		RemoteAddr:    req.RemoteAddr,
		Strategy:      req.Strategy,
		InterfaceName: req.InterfaceName,
		CreatedTime:   now,
		UpdatedTime:   now,
		Status:        1,
	}

	if err := s.DB.Create(&forward).Error; err != nil {
		slog.Error("創建轉發失敗", "error", err)
		return pkg.ErrMsg("创建转发失败")
	}

	// 5. 添加 Gost 配置
	s.addGostConfig(forward, tunnel)

	return pkg.Ok("转发创建成功")
}

// List 轉發列表
func (s *ForwardService) List(userID int64, roleID int) pkg.R {
	var forwards []model.Forward
	if roleID == 0 {
		s.DB.Order("inx ASC, id DESC").Find(&forwards)
	} else {
		s.DB.Where("user_id = ?", userID).Order("inx ASC, id DESC").Find(&forwards)
	}

	var result []ForwardWithTunnel
	for _, fw := range forwards {
		item := ForwardWithTunnel{
			ID:            fw.ID,
			Name:          fw.Name,
			InPort:        fw.InPort,
			RemoteAddr:    fw.RemoteAddr,
			Status:        fw.Status,
			CreatedTime:   fw.CreatedTime,
			UpdatedTime:   fw.UpdatedTime,
			UserName:      fw.UserName,
			UserID:        fw.UserID,
			TunnelID:      fw.TunnelID,
			InFlow:        fw.InFlow,
			OutFlow:       fw.OutFlow,
			Strategy:      fw.Strategy,
			Inx:           fw.Inx,
			InterfaceName: fw.InterfaceName,
		}

		var tunnel model.Tunnel
		if s.DB.First(&tunnel, fw.TunnelID).Error == nil {
			item.TunnelName = tunnel.Name
			item.InIP = tunnel.InIP
		}
		result = append(result, item)
	}
	return pkg.Ok(result)
}

// Update 更新轉發
func (s *ForwardService) Update(req UpdateForwardReq, roleID int, currentUserID int64) pkg.R {
	var forward model.Forward
	if err := s.DB.First(&forward, req.ID).Error; err != nil {
		return pkg.ErrMsg("转发不存在")
	}

	// 權限校驗
	if roleID != 0 && int64(forward.UserID) != currentUserID {
		return pkg.ErrMsg("无权操作")
	}

	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, req.TunnelID).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}

	// 端口變更校驗
	newInPort := forward.InPort
	if req.InPort != nil && *req.InPort != forward.InPort {
		newInPort = *req.InPort
		inNode := s.getNode(tunnel.InNodeID)
		if inNode == nil {
			return pkg.ErrMsg("入口节点不存在")
		}
		if newInPort < inNode.PortSta || newInPort > inNode.PortEnd {
			return pkg.ErrMsg(fmt.Sprintf("端口必须在 %d-%d 范围内", inNode.PortSta, inNode.PortEnd))
		}
		var count int64
		s.DB.Model(&model.Forward{}).Where("tunnel_id = ? AND in_port = ? AND id != ? AND status != -1", req.TunnelID, newInPort, req.ID).Count(&count)
		if count > 0 {
			return pkg.ErrMsg("该端口已被占用")
		}
	}

	// 先刪除舊 Gost 配置
	s.deleteGostConfig(forward, tunnel)

	// 更新資料庫
	forward.Name = req.Name
	forward.InPort = newInPort
	forward.RemoteAddr = req.RemoteAddr
	forward.Strategy = req.Strategy
	forward.InterfaceName = req.InterfaceName
	forward.UpdatedTime = time.Now().UnixMilli()

	if err := s.DB.Save(&forward).Error; err != nil {
		slog.Error("更新轉發失敗", "error", err)
		return pkg.ErrMsg("更新转发失败")
	}

	// 重新添加 Gost 配置
	if forward.Status == 1 {
		s.addGostConfig(forward, tunnel)
	}

	return pkg.Ok("更新成功")
}

// Delete 刪除轉發
func (s *ForwardService) Delete(id int64, roleID int, currentUserID int64) pkg.R {
	var forward model.Forward
	if err := s.DB.First(&forward, id).Error; err != nil {
		return pkg.ErrMsg("转发不存在")
	}

	if roleID != 0 && int64(forward.UserID) != currentUserID {
		return pkg.ErrMsg("无权操作")
	}

	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, forward.TunnelID).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}

	// 刪除 Gost 配置
	s.deleteGostConfig(forward, tunnel)

	// 刪除資料庫記錄
	if err := s.DB.Delete(&model.Forward{}, id).Error; err != nil {
		slog.Error("刪除轉發失敗", "error", err)
		return pkg.ErrMsg("删除转发失败")
	}

	return pkg.Ok("删除成功")
}

// Pause 暫停轉發
func (s *ForwardService) Pause(id int64, roleID int, currentUserID int64) pkg.R {
	var forward model.Forward
	if err := s.DB.First(&forward, id).Error; err != nil {
		return pkg.ErrMsg("转发不存在")
	}
	if roleID != 0 && int64(forward.UserID) != currentUserID {
		return pkg.ErrMsg("无权操作")
	}

	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, forward.TunnelID).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}

	name := fmt.Sprintf("fow_%d", forward.ID)
	pkg.PauseService(tunnel.InNodeID, name)
	if tunnel.Type != 1 {
		pkg.PauseRemoteService(tunnel.OutNodeID, name)
	}

	s.DB.Model(&forward).Updates(map[string]interface{}{
		"status":       0,
		"updated_time": time.Now().UnixMilli(),
	})

	return pkg.Ok("暂停成功")
}

// Resume 恢復轉發
func (s *ForwardService) Resume(id int64, roleID int, currentUserID int64) pkg.R {
	var forward model.Forward
	if err := s.DB.First(&forward, id).Error; err != nil {
		return pkg.ErrMsg("转发不存在")
	}
	if roleID != 0 && int64(forward.UserID) != currentUserID {
		return pkg.ErrMsg("无权操作")
	}

	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, forward.TunnelID).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}

	name := fmt.Sprintf("fow_%d", forward.ID)
	pkg.ResumeService(tunnel.InNodeID, name)
	if tunnel.Type != 1 {
		pkg.ResumeRemoteService(tunnel.OutNodeID, name)
	}

	s.DB.Model(&forward).Updates(map[string]interface{}{
		"status":       1,
		"updated_time": time.Now().UnixMilli(),
	})

	return pkg.Ok("恢复成功")
}

// Diagnose 診斷轉發（檢查 Gost 配置狀態）
func (s *ForwardService) Diagnose(id int64) pkg.R {
	var forward model.Forward
	if err := s.DB.First(&forward, id).Error; err != nil {
		return pkg.ErrMsg("转发不存在")
	}

	var tunnel model.Tunnel
	if err := s.DB.First(&tunnel, forward.TunnelID).Error; err != nil {
		return pkg.ErrMsg("隧道不存在")
	}

	// 重新下發配置
	name := fmt.Sprintf("fow_%d", forward.ID)
	s.deleteGostConfig(forward, tunnel)

	if forward.Status == 1 {
		s.addGostConfig(forward, tunnel)
		return pkg.Ok(fmt.Sprintf("转发 %s 配置已重新下发", name))
	}

	return pkg.Ok(fmt.Sprintf("转发 %s 处于暂停状态，已清理配置", name))
}

// UpdateOrder 更新排序
func (s *ForwardService) UpdateOrder(ids []int64) pkg.R {
	for i, id := range ids {
		s.DB.Model(&model.Forward{}).Where("id = ?", id).Update("inx", i)
	}
	return pkg.Ok("排序更新成功")
}

// ──────────────────────── 內部方法 ────────────────────────

func (s *ForwardService) checkUserTunnelPermission(userID int64, tunnelID int) string {
	var ut model.UserTunnel
	if err := s.DB.Where("user_id = ? AND tunnel_id = ? AND status = 1", userID, tunnelID).First(&ut).Error; err != nil {
		return "没有该隧道的使用权限"
	}

	// 校驗流量
	if ut.Flow > 0 && (ut.InFlow+ut.OutFlow) >= ut.Flow {
		return "隧道流量已用完"
	}

	// 校驗到期時間
	if ut.ExpTime > 0 && ut.ExpTime < time.Now().UnixMilli() {
		return "隧道权限已过期"
	}

	// 校驗轉發數量
	if ut.Num > 0 {
		var count int64
		s.DB.Model(&model.Forward{}).Where("user_id = ? AND tunnel_id = ? AND status != -1", userID, tunnelID).Count(&count)
		if int(count) >= ut.Num {
			return "转发数量已达上限"
		}
	}

	return ""
}

func (s *ForwardService) allocatePort(tunnel model.Tunnel) int {
	inNode := s.getNode(tunnel.InNodeID)
	if inNode == nil {
		return 0
	}

	// 收集已使用的端口
	var usedPorts []int
	s.DB.Model(&model.Forward{}).Where("tunnel_id = ? AND status != -1", tunnel.ID).Pluck("in_port", &usedPorts)
	usedSet := make(map[int]bool)
	for _, p := range usedPorts {
		usedSet[p] = true
	}

	// 尋找可用端口
	for port := inNode.PortSta; port <= inNode.PortEnd; port++ {
		if !usedSet[port] {
			return port
		}
	}
	return 0
}

func (s *ForwardService) allocateOutPort(tunnel model.Tunnel) int {
	outNode := s.getNode(tunnel.OutNodeID)
	if outNode == nil {
		return 0
	}

	// 出口節點上的已用端口（across all tunnels using this node as out）
	var usedPorts []int
	s.DB.Model(&model.Forward{}).
		Joins("JOIN tunnel ON forward.tunnel_id = tunnel.id").
		Where("tunnel.out_node_id = ? AND forward.out_port IS NOT NULL AND forward.status != -1", tunnel.OutNodeID).
		Pluck("forward.out_port", &usedPorts)

	usedSet := make(map[int]bool)
	for _, p := range usedPorts {
		usedSet[p] = true
	}

	for port := outNode.PortSta; port <= outNode.PortEnd; port++ {
		if !usedSet[port] {
			return port
		}
	}
	return 0
}

func (s *ForwardService) getNode(nodeID int64) *model.Node {
	var node model.Node
	if err := s.DB.First(&node, nodeID).Error; err != nil {
		return nil
	}
	return &node
}

func (s *ForwardService) addGostConfig(fw model.Forward, tunnel model.Tunnel) {
	name := fmt.Sprintf("fow_%d", fw.ID)

	// 查找是否有限速
	var limiter *int
	var ut model.UserTunnel
	if s.DB.Where("user_id = ? AND tunnel_id = ?", fw.UserID, fw.TunnelID).First(&ut).Error == nil {
		if ut.SpeedID != nil && *ut.SpeedID > 0 {
			limiter = ut.SpeedID
		}
	}

	if tunnel.Type == 1 {
		// 端口轉發：直接 AddService
		pkg.AddService(tunnel.InNodeID, name, fw.InPort, limiter, fw.RemoteAddr, tunnel.Type, tunnel, fw.Strategy, fw.InterfaceName)
	} else {
		// 隧道轉發：AddChains + AddService + AddRemoteService
		outAddr := fmt.Sprintf("%s:%d", tunnel.OutIP, *fw.OutPort)
		pkg.AddChains(tunnel.InNodeID, name, outAddr, tunnel.Protocol, tunnel.InterfaceName)
		pkg.AddService(tunnel.InNodeID, name, fw.InPort, limiter, outAddr, tunnel.Type, tunnel, fw.Strategy, fw.InterfaceName)
		pkg.AddRemoteService(tunnel.OutNodeID, name, *fw.OutPort, fw.RemoteAddr, tunnel.Protocol, fw.Strategy, fw.InterfaceName)
	}
}

func (s *ForwardService) deleteGostConfig(fw model.Forward, tunnel model.Tunnel) {
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
