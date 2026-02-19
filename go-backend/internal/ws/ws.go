package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lijt/flux-panel/internal/model"
	"github.com/lijt/flux-panel/internal/pkg"
	"gorm.io/gorm"
)

// ──────────────────── 常量 ────────────────────

const (
	sendMsgTimeout = 10 * time.Second // 等待節點回應超時
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ──────────────────── NodeConn 節點連線 ────────────────────

// NodeConn 代表一個節點 WebSocket 連線
type NodeConn struct {
	NodeID int64
	Secret string
	Conn   *websocket.Conn
	mu     sync.Mutex // 保護寫操作
}

// writeJSON 線程安全地寫 JSON
func (nc *NodeConn) writeJSON(v interface{}) error {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	return nc.Conn.WriteMessage(websocket.TextMessage, mustMarshal(v))
}

// writeText 線程安全地寫文字訊息（已序列化/加密的）
func (nc *NodeConn) writeText(msg string) error {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	return nc.Conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

// ──────────────────── AdminConn 管理員連線 ────────────────────

// AdminConn 代表一個管理員 WebSocket 連線
type AdminConn struct {
	UserID string
	Conn   *websocket.Conn
	mu     sync.Mutex
}

func (ac *AdminConn) writeText(msg string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	return ac.Conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

// ──────────────────── Hub WebSocket 管理器 ────────────────────

// Hub 管理所有 WebSocket 連線
type Hub struct {
	DB            *gorm.DB
	JWTSecret     string
	nodeSessions  map[int64]*NodeConn
	adminSessions map[string]*AdminConn
	mu            sync.RWMutex
	// pendingRequests: requestID → chan GostResult
	pendingRequests sync.Map
}

// NewHub 建立新的 Hub
func NewHub(db *gorm.DB, jwtSecret string) *Hub {
	return &Hub{
		DB:            db,
		JWTSecret:     jwtSecret,
		nodeSessions:  make(map[int64]*NodeConn),
		adminSessions: make(map[string]*AdminConn),
	}
}

// ──────────────────── HTTP → WebSocket 升級 ────────────────────

// HandleWS 處理 WebSocket 升級請求（掛載到 gin 路由）
func (h *Hub) HandleWS(c *gin.Context) {
	connType := c.Query("type")
	secret := c.Query("secret")

	if connType == "1" {
		// 節點連線：以 secret 查 node
		h.handleNodeConnect(c, secret)
	} else {
		// 管理員連線：以 secret 作為 JWT token 驗證
		h.handleAdminConnect(c, secret)
	}
}

// ──────────────────── 節點連線 ────────────────────

func (h *Hub) handleNodeConnect(c *gin.Context, secret string) {
	if secret == "" {
		c.JSON(http.StatusUnauthorized, pkg.ErrMsg("缺少 secret"))
		return
	}

	// 以 secret 查節點
	var node model.Node
	if err := h.DB.Where("secret = ?", secret).First(&node).Error; err != nil {
		slog.Warn("節點驗證失敗：未找到匹配的 secret")
		c.JSON(http.StatusUnauthorized, pkg.ErrMsg("節點驗證失敗"))
		return
	}

	version := c.Query("version")
	httpPort := c.Query("http")
	tlsPort := c.Query("tls")
	socksPort := c.Query("socks")

	// 升級 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("WebSocket 升級失敗", "nodeID", node.ID, "error", err)
		return
	}

	nc := &NodeConn{
		NodeID: node.ID,
		Secret: secret,
		Conn:   conn,
	}

	// 註冊到 hub，關閉舊連線
	h.mu.Lock()
	if old, exists := h.nodeSessions[node.ID]; exists && old.Conn != nil {
		slog.Info("節點已有連線存在，關閉舊連線", "nodeID", node.ID)
		old.Conn.Close()
	}
	h.nodeSessions[node.ID] = nc
	h.mu.Unlock()

	// 更新節點狀態
	updates := map[string]interface{}{"status": 1}
	if version != "" {
		updates["version"] = version
	}
	if httpPort != "" {
		if v, err := strconv.Atoi(httpPort); err == nil {
			updates["http"] = v
		}
	}
	if tlsPort != "" {
		if v, err := strconv.Atoi(tlsPort); err == nil {
			updates["tls"] = v
		}
	}
	if socksPort != "" {
		if v, err := strconv.Atoi(socksPort); err == nil {
			updates["socks"] = v
		}
	}
	h.DB.Model(&model.Node{}).Where("id = ?", node.ID).Updates(updates)
	slog.Info("節點連線建立", "nodeID", node.ID, "version", version)

	// 廣播上線
	h.broadcastStatus(node.ID, 1)

	// 啟動讀取迴圈
	go h.readPumpNode(nc)
}

// ──────────────────── 管理員連線 ────────────────────

func (h *Hub) handleAdminConnect(c *gin.Context, token string) {
	if token == "" {
		c.JSON(http.StatusUnauthorized, pkg.ErrMsg("缺少 token"))
		return
	}

	claims, err := pkg.ValidateToken(token, h.JWTSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, pkg.ErrMsg("token 無效"))
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("WebSocket 升級失敗（管理員）", "error", err)
		return
	}

	ac := &AdminConn{
		UserID: claims.Sub,
		Conn:   conn,
	}

	sessionID := uuid.NewString()

	h.mu.Lock()
	h.adminSessions[sessionID] = ac
	h.mu.Unlock()

	slog.Info("管理員 WebSocket 連線建立", "userID", claims.Sub)

	go h.readPumpAdmin(ac, sessionID)
}

// ──────────────────── 讀取迴圈 ────────────────────

func (h *Hub) readPumpNode(nc *NodeConn) {
	defer func() {
		h.onNodeDisconnect(nc)
		nc.Conn.Close()
	}()

	nc.Conn.SetReadDeadline(time.Now().Add(pongWait))
	nc.Conn.SetPongHandler(func(string) error {
		nc.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	crypto := pkg.GetOrCreateCrypto(nc.Secret)

	for {
		_, rawMsg, err := nc.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Warn("節點 WebSocket 讀取錯誤", "nodeID", nc.NodeID, "error", err)
			}
			return
		}

		payload := pkg.UnwrapDecrypt(crypto, string(rawMsg))
		h.handleNodeMessage(nc, payload)
	}
}

func (h *Hub) readPumpAdmin(ac *AdminConn, sessionID string) {
	defer func() {
		h.mu.Lock()
		delete(h.adminSessions, sessionID)
		h.mu.Unlock()
		ac.Conn.Close()
		slog.Info("管理員 WebSocket 連線關閉", "userID", ac.UserID)
	}()

	for {
		_, _, err := ac.Conn.ReadMessage()
		if err != nil {
			return
		}
		// 管理員訊息不需要特別處理
	}
}

// ──────────────────── 處理節點訊息 ────────────────────

func (h *Hub) handleNodeMessage(nc *NodeConn, payload string) {
	if payload == "" {
		return
	}

	// 心跳（memory_usage）
	if containsField(payload, "memory_usage") {
		crypto := pkg.GetOrCreateCrypto(nc.Secret)
		ack := `{"type":"call"}`
		outMsg, _ := pkg.WrapEncrypt(crypto, ack)
		nc.writeText(outMsg)
		return
	}

	// 帶 requestId 的回應
	if containsField(payload, "requestId") {
		slog.Info("收到節點回應", "nodeID", nc.NodeID, "payload", payload)

		var resp struct {
			RequestID string          `json:"requestId"`
			Message   string          `json:"message"`
			Type      string          `json:"type"`
			Data      json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal([]byte(payload), &resp); err != nil {
			slog.Warn("解析回應失敗", "error", err)
			return
		}

		if resp.RequestID == "" {
			return
		}

		if ch, ok := h.pendingRequests.LoadAndDelete(resp.RequestID); ok {
			result := pkg.GostResult{Msg: resp.Message}
			if resp.Message == "" {
				result.Msg = "OK"
			}
			if len(resp.Data) > 0 {
				var data interface{}
				json.Unmarshal(resp.Data, &data)
				result.Data = data
			}
			// 非阻塞送入 channel
			select {
			case ch.(chan pkg.GostResult) <- result:
			default:
			}
		}
		return
	}

	// 其他訊息：廣播給管理員
	broadcastData := map[string]interface{}{
		"id":   nc.NodeID,
		"type": "info",
		"data": payload,
	}
	b, _ := json.Marshal(broadcastData)
	h.broadcastToAdmins(string(b))
}

// ──────────────────── 節點斷線 ────────────────────

func (h *Hub) onNodeDisconnect(nc *NodeConn) {
	h.mu.Lock()
	// 只有當前 session 才清理
	if current, ok := h.nodeSessions[nc.NodeID]; ok && current == nc {
		delete(h.nodeSessions, nc.NodeID)
		h.mu.Unlock()

		// 更新為離線
		h.DB.Model(&model.Node{}).Where("id = ?", nc.NodeID).Update("status", 0)
		slog.Info("節點離線", "nodeID", nc.NodeID)

		h.broadcastStatus(nc.NodeID, 0)
	} else {
		h.mu.Unlock()
		slog.Info("節點舊連線關閉，跳過狀態更新", "nodeID", nc.NodeID)
	}
}

// ──────────────────── SendMsg 核心發送 ────────────────────

// SendMsg 發送命令到節點並等待回應（替換 gost.go 的 stub）
func (h *Hub) SendMsg(nodeID int64, data interface{}, action string) pkg.GostResult {
	h.mu.RLock()
	nc, ok := h.nodeSessions[nodeID]
	h.mu.RUnlock()

	if !ok || nc == nil {
		slog.Warn("發送失敗：節點不在線", "nodeID", nodeID)
		return pkg.GostResult{Code: 1, Msg: "節點不在線"}
	}

	// 生成 requestID
	requestID := uuid.NewString()

	// 建立等待 channel
	ch := make(chan pkg.GostResult, 1)
	h.pendingRequests.Store(requestID, ch)

	// 構建訊息
	msg := map[string]interface{}{
		"type":      action,
		"data":      data,
		"requestId": requestID,
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		h.pendingRequests.Delete(requestID)
		return pkg.GostResult{Code: 1, Msg: "序列化失敗"}
	}

	// 加密
	crypto := pkg.GetOrCreateCrypto(nc.Secret)
	outMsg, _ := pkg.WrapEncrypt(crypto, string(msgJSON))

	// 發送
	if err := nc.writeText(outMsg); err != nil {
		h.pendingRequests.Delete(requestID)
		slog.Error("發送 WebSocket 訊息失敗", "nodeID", nodeID, "error", err)
		return pkg.GostResult{Code: 1, Msg: "發送失敗"}
	}

	// 等待回應
	select {
	case result := <-ch:
		slog.Info("節點回應成功", "nodeID", nodeID, "action", action)
		return result
	case <-time.After(sendMsgTimeout):
		h.pendingRequests.Delete(requestID)
		slog.Warn("等待節點回應超時", "nodeID", nodeID, "action", action)
		return pkg.GostResult{Code: 1, Msg: "等待回應超時"}
	}
}

// ──────────────────── 廣播 ────────────────────

func (h *Hub) broadcastStatus(nodeID int64, status int) {
	data := map[string]interface{}{
		"id":   nodeID,
		"type": "status",
		"data": status,
	}
	b, _ := json.Marshal(data)
	h.broadcastToAdmins(string(b))
}

func (h *Hub) broadcastToAdmins(msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ac := range h.adminSessions {
		if err := ac.writeText(msg); err != nil {
			slog.Warn("廣播給管理員失敗", "userID", ac.UserID, "error", err)
		}
	}
}

// ──────────────────── 工具函數 ────────────────────

func containsField(payload, field string) bool {
	return len(payload) > 0 && json.Valid([]byte(payload)) && containsKey(payload, field)
}

func containsKey(jsonStr, key string) bool {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}
