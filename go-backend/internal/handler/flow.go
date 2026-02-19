package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lijt/flux-panel/internal/model"
	"github.com/lijt/flux-panel/internal/pkg"
	"gorm.io/gorm"
)

const (
	defaultUserTunnelID = "0"
	bytesToGB           = int64(1024 * 1024 * 1024)
)

// FlowHandler 流量上報 handler
type FlowHandler struct {
	DB *gorm.DB
}

// ──────────────────── 並發鎖 ────────────────────

var (
	userLocks    sync.Map
	tunnelLocks  sync.Map
	forwardLocks sync.Map
)

func getLock(m *sync.Map, key string) *sync.Mutex {
	v, _ := m.LoadOrStore(key, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// ──────────────────── FlowDto ────────────────────

// FlowDto 節點上報的流量資料
type FlowDto struct {
	N string `json:"n"` // 轉發名稱：forwardId_userId_userTunnelId_type
	U int64  `json:"u"` // 上傳流量 (bytes)
	D int64  `json:"d"` // 下載流量 (bytes)
}

// ──────────────────── API 端點 ────────────────────

// Upload POST /flow/upload — 節點流量上報
func (h *FlowHandler) Upload(c *gin.Context) {
	secret := c.Query("secret")

	// 1. 驗證節點
	var count int64
	h.DB.Model(&model.Node{}).Where("secret = ?", secret).Count(&count)
	if count == 0 {
		c.String(http.StatusOK, "ok")
		return
	}

	// 2. 讀取並解密
	rawData, _ := c.GetRawData()
	decrypted := pkg.UnwrapDecrypt(pkg.GetOrCreateCrypto(secret), string(rawData))

	// 3. 解析 FlowDto
	var flow FlowDto
	if err := json.Unmarshal([]byte(decrypted), &flow); err != nil {
		c.String(http.StatusOK, "ok")
		return
	}

	// 過濾 web_api
	if flow.N == "web_api" {
		c.String(http.StatusOK, "ok")
		return
	}

	slog.Info("節點上報流量", "n", flow.N, "u", flow.U, "d", flow.D)

	// 4. 處理流量資料
	h.processFlowData(flow)
	c.String(http.StatusOK, "ok")
}

// Config POST /flow/config — 節點配置上報（觸發清理）
func (h *FlowHandler) Config(c *gin.Context) {
	secret := c.Query("secret")

	var node model.Node
	if err := h.DB.Where("secret = ?", secret).First(&node).Error; err != nil {
		c.String(http.StatusOK, "ok")
		return
	}

	rawData, _ := c.GetRawData()
	decrypted := pkg.UnwrapDecrypt(pkg.GetOrCreateCrypto(secret), string(rawData))

	// 異步清理孤立配置
	go h.cleanNodeConfigs(node.ID, decrypted)

	c.String(http.StatusOK, "ok")
}

// ──────────────────── 核心邏輯 ────────────────────

func (h *FlowHandler) processFlowData(flow FlowDto) {
	parts := strings.Split(flow.N, "_")
	if len(parts) < 3 {
		return
	}
	forwardID := parts[0]
	userID := parts[1]
	userTunnelID := parts[2]

	// 取得 Forward
	var forward model.Forward
	h.DB.First(&forward, forwardID)

	// 流量倍率 + 單/雙向
	flowType := h.getFlowType(&forward)
	h.applyTrafficRatio(&flow, &forward, flowType)

	// 原子更新流量
	h.updateForwardFlow(forwardID, flow)
	h.updateUserFlow(userID, flow)
	h.updateUserTunnelFlow(userTunnelID, flow)

	// 限流檢查（非管理員轉發）
	name := forwardID + "_" + userID + "_" + userTunnelID
	if userTunnelID != defaultUserTunnelID {
		h.checkUserLimits(userID, name)
		h.checkUserTunnelLimits(userTunnelID, name, userID)
	}
}

func (h *FlowHandler) getFlowType(forward *model.Forward) int {
	if forward.ID == 0 {
		return 2
	}
	var tunnel model.Tunnel
	if err := h.DB.First(&tunnel, forward.TunnelID).Error; err != nil {
		return 2
	}
	return tunnel.Flow
}

func (h *FlowHandler) applyTrafficRatio(flow *FlowDto, forward *model.Forward, flowType int) {
	if forward.ID == 0 {
		return
	}
	var tunnel model.Tunnel
	if err := h.DB.First(&tunnel, forward.TunnelID).Error; err != nil {
		return
	}
	ratio := tunnel.TrafficRatio
	if ratio <= 0 {
		ratio = 1
	}
	flow.D = int64(float64(flow.D)*ratio) * int64(flowType)
	flow.U = int64(float64(flow.U)*ratio) * int64(flowType)
}

// ──────────────────── 原子流量更新 ────────────────────

func (h *FlowHandler) updateForwardFlow(id string, flow FlowDto) {
	mu := getLock(&forwardLocks, id)
	mu.Lock()
	defer mu.Unlock()
	h.DB.Model(&model.Forward{}).Where("id = ?", id).
		UpdateColumn("in_flow", gorm.Expr("in_flow + ?", flow.D)).
		UpdateColumn("out_flow", gorm.Expr("out_flow + ?", flow.U))
}

func (h *FlowHandler) updateUserFlow(id string, flow FlowDto) {
	mu := getLock(&userLocks, id)
	mu.Lock()
	defer mu.Unlock()
	h.DB.Model(&model.User{}).Where("id = ?", id).
		UpdateColumn("in_flow", gorm.Expr("in_flow + ?", flow.D)).
		UpdateColumn("out_flow", gorm.Expr("out_flow + ?", flow.U))
}

func (h *FlowHandler) updateUserTunnelFlow(id string, flow FlowDto) {
	if id == defaultUserTunnelID {
		return
	}
	mu := getLock(&tunnelLocks, id)
	mu.Lock()
	defer mu.Unlock()
	h.DB.Model(&model.UserTunnel{}).Where("id = ?", id).
		UpdateColumn("in_flow", gorm.Expr("in_flow + ?", flow.D)).
		UpdateColumn("out_flow", gorm.Expr("out_flow + ?", flow.U))
}

// ──────────────────── 限流檢查 ────────────────────

func (h *FlowHandler) checkUserLimits(userID, name string) {
	var user model.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		return
	}

	now := time.Now().UnixMilli()

	// 流量超限
	if user.Flow > 0 {
		userFlowLimit := user.Flow * bytesToGB
		userCurrentFlow := user.InFlow + user.OutFlow
		if userFlowLimit < userCurrentFlow {
			h.pauseAllUserServices(userID, name)
			return
		}
	}

	// 到期
	if user.ExpTime > 0 && user.ExpTime <= now {
		h.pauseAllUserServices(userID, name)
		return
	}

	// 狀態異常
	if user.Status != 1 {
		h.pauseAllUserServices(userID, name)
	}
}

func (h *FlowHandler) checkUserTunnelLimits(utID, name, userID string) {
	var ut model.UserTunnel
	if err := h.DB.First(&ut, utID).Error; err != nil {
		return
	}

	now := time.Now().UnixMilli()

	// 隧道流量超限
	if ut.Flow > 0 {
		flow := ut.InFlow + ut.OutFlow
		if flow >= ut.Flow*bytesToGB {
			h.pauseSpecificForward(int64(ut.TunnelID), name, userID)
			return
		}
	}

	// 到期
	if ut.ExpTime > 0 && ut.ExpTime <= now {
		h.pauseSpecificForward(int64(ut.TunnelID), name, userID)
		return
	}

	// 狀態異常
	if ut.Status != 1 {
		h.pauseSpecificForward(int64(ut.TunnelID), name, userID)
	}
}

func (h *FlowHandler) pauseAllUserServices(userID, name string) {
	var forwards []model.Forward
	h.DB.Where("user_id = ?", userID).Find(&forwards)
	h.pauseForwards(forwards, name)
}

func (h *FlowHandler) pauseSpecificForward(tunnelID int64, name, userID string) {
	var forwards []model.Forward
	h.DB.Where("tunnel_id = ? AND user_id = ?", tunnelID, userID).Find(&forwards)
	h.pauseForwards(forwards, name)
}

func (h *FlowHandler) pauseForwards(forwards []model.Forward, name string) {
	for _, fwd := range forwards {
		var tunnel model.Tunnel
		if err := h.DB.First(&tunnel, fwd.TunnelID).Error; err != nil {
			continue
		}
		pkg.PauseService(tunnel.InNodeID, name)
		if tunnel.Type == 2 {
			pkg.PauseRemoteService(tunnel.OutNodeID, name)
		}
		h.DB.Model(&model.Forward{}).Where("id = ?", fwd.ID).Update("status", 0)
	}
}

// ──────────────────── 配置清理（CheckGostConfigAsync）────────────────────

type gostConfigDto struct {
	Services []configItem `json:"services"`
	Chains   []configItem `json:"chains"`
	Limiters []configItem `json:"limiters"`
}

type configItem struct {
	Name string `json:"name"`
}

func (h *FlowHandler) cleanNodeConfigs(nodeID int64, configJSON string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("清理配置 panic", "error", r)
		}
	}()

	var cfg gostConfigDto
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return
	}

	// 清理孤立 Service
	for _, svc := range cfg.Services {
		if svc.Name == "web_api" {
			continue
		}
		parts := strings.Split(svc.Name, "_")
		if len(parts) >= 4 {
			forwardID := parts[0]
			svcType := parts[3]
			var count int64
			h.DB.Model(&model.Forward{}).Where("id = ?", forwardID).Count(&count)
			if count == 0 {
				name := parts[0] + "_" + parts[1] + "_" + parts[2]
				if svcType == "tcp" {
					slog.Info("刪除孤立服務", "name", svc.Name, "nodeID", nodeID)
					pkg.DeleteService(nodeID, name)
				}
				if svcType == "tls" {
					slog.Info("刪除孤立遠端服務", "name", svc.Name, "nodeID", nodeID)
					pkg.DeleteRemoteService(nodeID, name)
				}
			}
		}
	}

	// 清理孤立 Chain
	for _, chain := range cfg.Chains {
		parts := strings.Split(chain.Name, "_")
		if len(parts) >= 4 && parts[3] == "chains" {
			forwardID := parts[0]
			var count int64
			h.DB.Model(&model.Forward{}).Where("id = ?", forwardID).Count(&count)
			if count == 0 {
				name := parts[0] + "_" + parts[1] + "_" + parts[2]
				slog.Info("刪除孤立鏈", "name", chain.Name, "nodeID", nodeID)
				pkg.DeleteChains(nodeID, name)
			}
		}
	}

	// 清理孤立 Limiter
	for _, limiter := range cfg.Limiters {
		var count int64
		h.DB.Model(&model.SpeedLimit{}).Where("id = ?", limiter.Name).Count(&count)
		if count == 0 {
			slog.Info("刪除孤立限流器", "name", limiter.Name, "nodeID", nodeID)
			// limiter name is numeric ID
			var limiterID int64
			fmt.Sscanf(limiter.Name, "%d", &limiterID)
			if limiterID > 0 {
				pkg.DeleteLimiters(nodeID, limiterID)
			}
		}
	}
}
