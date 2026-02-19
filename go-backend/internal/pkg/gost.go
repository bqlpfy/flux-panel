package pkg

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/lijt/flux-panel/internal/model"
)

// GostResult 表示 Gost 操作回應
type GostResult struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// GostOk 返回成功結果
func GostOk() GostResult {
	return GostResult{Code: 0, Msg: "success"}
}

// GostErr 返回錯誤結果
func GostErr(msg string) GostResult {
	return GostResult{Code: 1, Msg: msg}
}

// ──────────────────────── WebSocket Hub 介面注入 ────────────────────────

// WSHub WebSocket 發送介面（由 ws.Hub 實作）
type WSHub interface {
	SendMsg(nodeID int64, data interface{}, action string) GostResult
}

var wsHub WSHub

// SetWSHub 注入 WebSocket Hub 實例（由 main.go 啟動時呼叫）
func SetWSHub(h WSHub) {
	wsHub = h
}

// sendMsg 透過 WebSocket Hub 發送命令到節點
func sendMsg(nodeID int64, data interface{}, action string) GostResult {
	if wsHub == nil {
		slog.Warn("GostUtil.sendMsg: WebSocket Hub 尚未初始化", "nodeID", nodeID, "action", action)
		return GostOk() // 靜默成功，避免阻塞啟動流程
	}
	return wsHub.SendMsg(nodeID, data, action)
}

// ──────────────────────── Limiters ────────────────────────

func AddLimiters(nodeID int64, name int64, speed string) GostResult {
	data := createLimiterData(name, speed)
	return sendMsg(nodeID, data, "AddLimiters")
}

func UpdateLimiters(nodeID int64, name int64, speed string) GostResult {
	data := createLimiterData(name, speed)
	req := map[string]interface{}{
		"limiter": fmt.Sprintf("%d", name),
		"data":    data,
	}
	return sendMsg(nodeID, req, "UpdateLimiters")
}

func DeleteLimiters(nodeID int64, name int64) GostResult {
	req := map[string]interface{}{
		"limiter": fmt.Sprintf("%d", name),
	}
	return sendMsg(nodeID, req, "DeleteLimiters")
}

// ──────────────────────── Service ────────────────────────

func AddService(nodeID int64, name string, inPort int, limiter *int, remoteAddr string, fowType int, tunnel model.Tunnel, strategy string, interfaceName string) GostResult {
	services := buildServicePair(name, inPort, limiter, remoteAddr, fowType, tunnel, strategy, interfaceName)
	return sendMsg(nodeID, services, "AddService")
}

func UpdateService(nodeID int64, name string, inPort int, limiter *int, remoteAddr string, fowType int, tunnel model.Tunnel, strategy string, interfaceName string) GostResult {
	services := buildServicePair(name, inPort, limiter, remoteAddr, fowType, tunnel, strategy, interfaceName)
	return sendMsg(nodeID, services, "UpdateService")
}

func DeleteService(nodeID int64, name string) GostResult {
	data := map[string]interface{}{
		"services": []string{name + "_tcp", name + "_udp"},
	}
	return sendMsg(nodeID, data, "DeleteService")
}

func PauseService(nodeID int64, name string) GostResult {
	data := map[string]interface{}{
		"services": []string{name + "_tcp", name + "_udp"},
	}
	return sendMsg(nodeID, data, "PauseService")
}

func ResumeService(nodeID int64, name string) GostResult {
	data := map[string]interface{}{
		"services": []string{name + "_tcp", name + "_udp"},
	}
	return sendMsg(nodeID, data, "ResumeService")
}

// ──────────────────────── Remote Service (TLS) ────────────────────────

func AddRemoteService(nodeID int64, name string, outPort int, remoteAddr string, protocol string, strategy string, interfaceName string) GostResult {
	data := buildRemoteServiceData(name, outPort, remoteAddr, protocol, strategy, interfaceName)
	return sendMsg(nodeID, []interface{}{data}, "AddService")
}

func UpdateRemoteService(nodeID int64, name string, outPort int, remoteAddr string, protocol string, strategy string, interfaceName string) GostResult {
	data := buildRemoteServiceData(name, outPort, remoteAddr, protocol, strategy, interfaceName)
	return sendMsg(nodeID, []interface{}{data}, "UpdateService")
}

func DeleteRemoteService(nodeID int64, name string) GostResult {
	req := map[string]interface{}{
		"services": []string{name + "_tls"},
	}
	return sendMsg(nodeID, req, "DeleteService")
}

func PauseRemoteService(nodeID int64, name string) GostResult {
	data := map[string]interface{}{
		"services": []string{name + "_tls"},
	}
	return sendMsg(nodeID, data, "PauseService")
}

func ResumeRemoteService(nodeID int64, name string) GostResult {
	data := map[string]interface{}{
		"services": []string{name + "_tls"},
	}
	return sendMsg(nodeID, data, "ResumeService")
}

// ──────────────────────── Chains ────────────────────────

func AddChains(nodeID int64, name string, remoteAddr string, protocol string, interfaceName string) GostResult {
	data := buildChainData(name, remoteAddr, protocol, interfaceName)
	return sendMsg(nodeID, data, "AddChains")
}

func UpdateChains(nodeID int64, name string, remoteAddr string, protocol string, interfaceName string) GostResult {
	data := buildChainData(name, remoteAddr, protocol, interfaceName)
	req := map[string]interface{}{
		"chain": name + "_chains",
		"data":  data,
	}
	return sendMsg(nodeID, req, "UpdateChains")
}

func DeleteChains(nodeID int64, name string) GostResult {
	data := map[string]interface{}{
		"chain": name + "_chains",
	}
	return sendMsg(nodeID, data, "DeleteChains")
}

// ──────────────────────── 內部構建函數 ────────────────────────

func createLimiterData(name int64, speed string) map[string]interface{} {
	return map[string]interface{}{
		"name":   fmt.Sprintf("%d", name),
		"limits": []string{fmt.Sprintf("$ %sMB %sMB", speed, speed)},
	}
}

func buildServicePair(name string, inPort int, limiter *int, remoteAddr string, fowType int, tunnel model.Tunnel, strategy string, interfaceName string) []interface{} {
	var services []interface{}
	for _, protocol := range []string{"tcp", "udp"} {
		svc := createServiceConfig(name, inPort, limiter, remoteAddr, protocol, fowType, tunnel, strategy, interfaceName)
		services = append(services, svc)
	}
	return services
}

func createServiceConfig(name string, inPort int, limiter *int, remoteAddr string, protocol string, fowType int, tunnel model.Tunnel, strategy string, interfaceName string) map[string]interface{} {
	svc := map[string]interface{}{
		"name": name + "_" + protocol,
	}

	// 地址: tcp 用 tcpListenAddr, udp 用 udpListenAddr
	if protocol == "tcp" {
		svc["addr"] = fmt.Sprintf("%s:%d", tunnel.TcpListenAddr, inPort)
	} else {
		svc["addr"] = fmt.Sprintf("%s:%d", tunnel.UdpListenAddr, inPort)
	}

	// 接口
	if interfaceName != "" {
		svc["metadata"] = map[string]interface{}{"interface": interfaceName}
	}

	// 限流器
	if limiter != nil {
		svc["limiter"] = fmt.Sprintf("%d", *limiter)
	}

	// handler
	handler := map[string]interface{}{"type": protocol}
	if isTunnelForwarding(fowType) {
		handler["chain"] = name + "_chains"
	}
	svc["handler"] = handler

	// listener
	listener := map[string]interface{}{"type": protocol}
	if protocol == "udp" {
		listener["metadata"] = map[string]interface{}{"keepAlive": true}
	}
	svc["listener"] = listener

	// forwarder（端口轉發才需要）
	if isPortForwarding(fowType) {
		svc["forwarder"] = createForwarder(remoteAddr, strategy)
	}

	return svc
}

func buildRemoteServiceData(name string, outPort int, remoteAddr string, protocol string, strategy string, interfaceName string) map[string]interface{} {
	data := map[string]interface{}{
		"name": name + "_tls",
		"addr": fmt.Sprintf(":%d", outPort),
		"handler":  map[string]interface{}{"type": "relay"},
		"listener": map[string]interface{}{"type": protocol},
	}

	if interfaceName != "" {
		data["metadata"] = map[string]interface{}{"interface": interfaceName}
	}

	data["forwarder"] = createForwarder(remoteAddr, strategy)
	return data
}

func buildChainData(name string, remoteAddr string, protocol string, interfaceName string) map[string]interface{} {
	dialer := map[string]interface{}{"type": protocol}
	if protocol == "quic" {
		dialer["metadata"] = map[string]interface{}{"keepAlive": true, "ttl": "10s"}
	}

	connector := map[string]interface{}{"type": "relay"}

	node := map[string]interface{}{
		"name":      "node-" + name,
		"addr":      remoteAddr,
		"connector": connector,
		"dialer":    dialer,
	}
	if interfaceName != "" {
		node["interface"] = interfaceName
	}

	hop := map[string]interface{}{
		"name":  "hop-" + name,
		"nodes": []interface{}{node},
	}

	return map[string]interface{}{
		"name": name + "_chains",
		"hops": []interface{}{hop},
	}
}

func createForwarder(remoteAddr string, strategy string) map[string]interface{} {
	addrs := strings.Split(remoteAddr, ",")
	var nodes []interface{}
	for i, addr := range addrs {
		nodes = append(nodes, map[string]interface{}{
			"name": fmt.Sprintf("node_%d", i+1),
			"addr": strings.TrimSpace(addr),
		})
	}
	if strategy == "" {
		strategy = "fifo"
	}
	return map[string]interface{}{
		"nodes": nodes,
		"selector": map[string]interface{}{
			"strategy":    strategy,
			"maxFails":    1,
			"failTimeout": "600s",
		},
	}
}

func isPortForwarding(fowType int) bool {
	return fowType == 1
}

func isTunnelForwarding(fowType int) bool {
	return fowType != 1
}
