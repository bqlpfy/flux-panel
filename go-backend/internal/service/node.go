package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lijt/flux-panel/internal/model"
	"github.com/lijt/flux-panel/internal/pkg"
	"gorm.io/gorm"
)

// NodeService 節點業務邏輯
type NodeService struct {
	DB *gorm.DB
}

type CreateNodeReq struct {
	Name     string `json:"name" binding:"required"`
	IP       string `json:"ip" binding:"required"`
	ServerIP string `json:"server_ip" binding:"required"`
	PortSta  int    `json:"port_sta" binding:"required"`
	PortEnd  int    `json:"port_end" binding:"required"`
}

type UpdateNodeReq struct {
	ID       int64  `json:"id" binding:"required"`
	Name     string `json:"name" binding:"required"`
	IP       string `json:"ip" binding:"required"`
	ServerIP string `json:"server_ip" binding:"required"`
	PortSta  int    `json:"port_sta" binding:"required"`
	PortEnd  int    `json:"port_end" binding:"required"`
	HTTP     *int   `json:"http"`
	TLS      *int   `json:"tls"`
	Socks    *int   `json:"socks"`
}

func (s *NodeService) CreateNode(req CreateNodeReq) pkg.R {
	if err := validatePortRange(req.PortSta, req.PortEnd); err != nil {
		return pkg.ErrMsg(err.Error())
	}

	now := time.Now().UnixMilli()
	node := model.Node{
		Name:        req.Name,
		IP:          req.IP,
		ServerIP:    req.ServerIP,
		Secret:      generateUUID(),
		PortSta:     req.PortSta,
		PortEnd:     req.PortEnd,
		Status:      0, // 初始離線
		CreatedTime: now,
		UpdatedTime: now,
	}

	if err := s.DB.Create(&node).Error; err != nil {
		slog.Error("創建節點失敗", "error", err)
		return pkg.ErrMsg("节点创建失败")
	}
	return pkg.Ok("节点创建成功")
}

func (s *NodeService) GetAllNodes() pkg.R {
	var nodes []model.Node
	s.DB.Find(&nodes)
	// 隱藏 secret
	for i := range nodes {
		nodes[i].Secret = ""
	}
	return pkg.Ok(nodes)
}

func (s *NodeService) UpdateNode(req UpdateNodeReq) pkg.R {
	var node model.Node
	if err := s.DB.First(&node, req.ID).Error; err != nil {
		return pkg.ErrMsg("节点不存在")
	}

	if err := validatePortRange(req.PortSta, req.PortEnd); err != nil {
		return pkg.ErrMsg(err.Error())
	}

	// TODO: Phase 4 中實作節點在線時通過 WebSocket 通知協議更新

	updates := map[string]interface{}{
		"name":         req.Name,
		"ip":           req.IP,
		"server_ip":    req.ServerIP,
		"port_sta":     req.PortSta,
		"port_end":     req.PortEnd,
		"updated_time": time.Now().UnixMilli(),
	}
	if req.HTTP != nil {
		updates["http"] = *req.HTTP
	}
	if req.TLS != nil {
		updates["tls"] = *req.TLS
	}
	if req.Socks != nil {
		updates["socks"] = *req.Socks
	}

	s.DB.Model(&model.Node{}).Where("id = ?", req.ID).Updates(updates)

	// 更新關聯隧道的入口 IP
	s.DB.Model(&model.Tunnel{}).Where("in_node_id = ?", req.ID).Update("in_ip", req.IP)
	// 更新關聯隧道的出口 IP
	s.DB.Model(&model.Tunnel{}).Where("out_node_id = ?", req.ID).Update("out_ip", req.ServerIP)

	return pkg.Ok("节点更新成功")
}

func (s *NodeService) DeleteNode(id int64) pkg.R {
	var node model.Node
	if err := s.DB.First(&node, id).Error; err != nil {
		return pkg.ErrMsg("节点不存在")
	}

	// 檢查入口節點使用
	var inCount int64
	s.DB.Model(&model.Tunnel{}).Where("in_node_id = ?", id).Count(&inCount)
	if inCount > 0 {
		return pkg.ErrMsg(fmt.Sprintf("该节点还有 %d 个隧道作为入口节点在使用，请先删除相关隧道", inCount))
	}

	// 檢查出口節點使用
	var outCount int64
	s.DB.Model(&model.Tunnel{}).Where("out_node_id = ?", id).Count(&outCount)
	if outCount > 0 {
		return pkg.ErrMsg(fmt.Sprintf("该节点还有 %d 个隧道作为出口节点在使用，请先删除相关隧道", outCount))
	}

	if err := s.DB.Delete(&model.Node{}, id).Error; err != nil {
		return pkg.ErrMsg("节点删除失败")
	}
	return pkg.Ok("节点删除成功")
}

func (s *NodeService) GetInstallCommand(id int64) pkg.R {
	var node model.Node
	if err := s.DB.First(&node, id).Error; err != nil {
		return pkg.ErrMsg("节点不存在")
	}

	var ipConfig model.ViteConfig
	if err := s.DB.Where("name = ?", "ip").First(&ipConfig).Error; err != nil {
		return pkg.ErrMsg("请先前往网站配置中设置ip")
	}

	serverAddr := processServerAddress(ipConfig.Value)
	cmd := fmt.Sprintf("curl -L https://github.com/bqlpfy/flux-panel/releases/download/1.4.3/install.sh -o ./install.sh && chmod +x ./install.sh && ./install.sh -a %s -s %s",
		serverAddr, node.Secret)

	return pkg.Ok(cmd)
}

func (s *NodeService) GetNodeByID(id int64) (*model.Node, error) {
	var node model.Node
	if err := s.DB.First(&node, id).Error; err != nil {
		return nil, fmt.Errorf("节点不存在")
	}
	return &node, nil
}

// --- 輔助函數 ---

func validatePortRange(sta, end int) error {
	if sta < 1 || sta > 65535 || end < 1 || end > 65535 {
		return fmt.Errorf("端口必须在1-65535范围内")
	}
	if end < sta {
		return fmt.Errorf("结束端口不能小于起始端口")
	}
	return nil
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func processServerAddress(addr string) string {
	if addr == "" || strings.HasPrefix(addr, "[") {
		return addr
	}
	lastColon := strings.LastIndex(addr, ":")
	if lastColon == -1 {
		if isIPv6(addr) {
			return "[" + addr + "]"
		}
		return addr
	}
	host := addr[:lastColon]
	port := addr[lastColon:]
	if isIPv6(host) {
		return "[" + host + "]" + port
	}
	return addr
}

func isIPv6(addr string) bool {
	return strings.Count(addr, ":") >= 2
}
