package model

// model 定義所有資料庫模型，對應現有 MySQL schema

// User 用戶表
type User struct {
	ID            int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	User          string `json:"user" gorm:"column:user"`
	Pwd           string `json:"-" gorm:"column:pwd"`
	RoleID        int    `json:"role_id" gorm:"column:role_id"`
	ExpTime       int64  `json:"exp_time" gorm:"column:exp_time"`
	Flow          int64  `json:"flow" gorm:"column:flow"`
	InFlow        int64  `json:"in_flow" gorm:"column:in_flow"`
	OutFlow       int64  `json:"out_flow" gorm:"column:out_flow"`
	FlowResetTime int64  `json:"flow_reset_time" gorm:"column:flow_reset_time"`
	Num           int    `json:"num" gorm:"column:num"`
	CreatedTime   int64  `json:"created_time" gorm:"column:created_time"`
	UpdatedTime   int64  `json:"updated_time" gorm:"column:updated_time"`
	Status        int    `json:"status" gorm:"column:status"`
}

func (User) TableName() string { return "user" }

// Node 節點表
type Node struct {
	ID          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string `json:"name" gorm:"column:name"`
	Secret      string `json:"secret" gorm:"column:secret"`
	IP          string `json:"ip" gorm:"column:ip"`
	ServerIP    string `json:"server_ip" gorm:"column:server_ip"`
	PortSta     int    `json:"port_sta" gorm:"column:port_sta"`
	PortEnd     int    `json:"port_end" gorm:"column:port_end"`
	Version     string `json:"version" gorm:"column:version"`
	HTTP        int    `json:"http" gorm:"column:http"`
	TLS         int    `json:"tls" gorm:"column:tls"`
	Socks       int    `json:"socks" gorm:"column:socks"`
	CreatedTime int64  `json:"created_time" gorm:"column:created_time"`
	UpdatedTime int64  `json:"updated_time" gorm:"column:updated_time"`
	Status      int    `json:"status" gorm:"column:status"`
}

func (Node) TableName() string { return "node" }

// Tunnel 隧道表
type Tunnel struct {
	ID           int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name         string     `json:"name" gorm:"column:name"`
	TrafficRatio float64    `json:"traffic_ratio" gorm:"column:traffic_ratio;type:decimal(10,1)"` 
	InNodeID     int64      `json:"in_node_id" gorm:"column:in_node_id"`
	InIP         string     `json:"in_ip" gorm:"column:in_ip"`
	OutNodeID    int64      `json:"out_node_id" gorm:"column:out_node_id"`
	OutIP        string     `json:"out_ip" gorm:"column:out_ip"`
	Type         int        `json:"type" gorm:"column:type"`
	Protocol     string     `json:"protocol" gorm:"column:protocol"`
	Flow         int        `json:"flow" gorm:"column:flow"`
	TcpListenAddr string   `json:"tcp_listen_addr" gorm:"column:tcp_listen_addr"`
	UdpListenAddr string   `json:"udp_listen_addr" gorm:"column:udp_listen_addr"`
	InterfaceName string   `json:"interface_name" gorm:"column:interface_name"`
	CreatedTime  int64      `json:"created_time" gorm:"column:created_time"`
	UpdatedTime  int64      `json:"updated_time" gorm:"column:updated_time"`
	Status       int        `json:"status" gorm:"column:status"`
}

func (Tunnel) TableName() string { return "tunnel" }

// Forward 轉發表
type Forward struct {
	ID            int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        int    `json:"user_id" gorm:"column:user_id"`
	UserName      string `json:"user_name" gorm:"column:user_name"`
	Name          string `json:"name" gorm:"column:name"`
	TunnelID      int    `json:"tunnel_id" gorm:"column:tunnel_id"`
	InPort        int    `json:"in_port" gorm:"column:in_port"`
	OutPort       *int   `json:"out_port" gorm:"column:out_port"`
	RemoteAddr    string `json:"remote_addr" gorm:"column:remote_addr"`
	Strategy      string `json:"strategy" gorm:"column:strategy"`
	InterfaceName string `json:"interface_name" gorm:"column:interface_name"`
	InFlow        int64  `json:"in_flow" gorm:"column:in_flow"`
	OutFlow       int64  `json:"out_flow" gorm:"column:out_flow"`
	CreatedTime   int64  `json:"created_time" gorm:"column:created_time"`
	UpdatedTime   int64  `json:"updated_time" gorm:"column:updated_time"`
	Status        int    `json:"status" gorm:"column:status"`
	Inx           int    `json:"inx" gorm:"column:inx"`
}

func (Forward) TableName() string { return "forward" }

// UserTunnel 用戶隧道關聯表
type UserTunnel struct {
	ID            int   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        int   `json:"user_id" gorm:"column:user_id"`
	TunnelID      int   `json:"tunnel_id" gorm:"column:tunnel_id"`
	SpeedID       *int  `json:"speed_id" gorm:"column:speed_id"`
	Num           int   `json:"num" gorm:"column:num"`
	Flow          int64 `json:"flow" gorm:"column:flow"`
	InFlow        int64 `json:"in_flow" gorm:"column:in_flow"`
	OutFlow       int64 `json:"out_flow" gorm:"column:out_flow"`
	FlowResetTime int64 `json:"flow_reset_time" gorm:"column:flow_reset_time"`
	ExpTime       int64 `json:"exp_time" gorm:"column:exp_time"`
	Status        int   `json:"status" gorm:"column:status"`
}

func (UserTunnel) TableName() string { return "user_tunnel" }

// SpeedLimit 限速規則表
type SpeedLimit struct {
	ID          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string `json:"name" gorm:"column:name"`
	Speed       int    `json:"speed" gorm:"column:speed"`
	TunnelID    int64  `json:"tunnel_id" gorm:"column:tunnel_id"`
	TunnelName  string `json:"tunnel_name" gorm:"column:tunnel_name"`
	CreatedTime int64  `json:"created_time" gorm:"column:created_time"`
	UpdatedTime int64  `json:"updated_time" gorm:"column:updated_time"`
	Status      int    `json:"status" gorm:"column:status"`
}

func (SpeedLimit) TableName() string { return "speed_limit" }

// StatisticsFlow 流量統計表
type StatisticsFlow struct {
	ID          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      int64  `json:"user_id" gorm:"column:user_id"`
	Flow        int64  `json:"flow" gorm:"column:flow"`
	TotalFlow   int64  `json:"total_flow" gorm:"column:total_flow"`
	Time        string `json:"time" gorm:"column:time"`
	CreatedTime int64  `json:"created_time" gorm:"column:created_time"`
}

func (StatisticsFlow) TableName() string { return "statistics_flow" }

// ViteConfig 前端配置表
type ViteConfig struct {
	ID    int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Name  string `json:"name" gorm:"column:name;uniqueIndex"`
	Value string `json:"value" gorm:"column:value"`
	Time  int64  `json:"time" gorm:"column:time"`
}

func (ViteConfig) TableName() string { return "vite_config" }
