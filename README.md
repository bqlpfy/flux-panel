# flux-panel 转发面板

本项目基于 [go-gost/gost](https://github.com/go-gost/gost) 和 [go-gost/x](https://github.com/go-gost/x) 两个开源库，实现了转发面板。

# 赞助商
<p align="center">
  <a href="https://vps.town" style="margin: 0 20px; text-align:center;">
    <img src="./doc/vpstown.png" width="300">
  </a>

  <a href="https://whmcs.as211392.com" style="margin: 0 20px; text-align:center;">
    <img src="./doc/as211392.png" width="300">
  </a>
</p>

---

## 特性

- 支持按 **隧道账号级别** 管理流量转发数量，可用于用户/隧道配额控制
- 支持 **TCP** 和 **UDP** 协议的转发
- 支持两种转发模式：**端口转发** 与 **隧道转发**
- 可针对 **指定用户的指定隧道进行限速** 设置
- 支持配置 **单向或双向流量计费方式**，灵活适配不同计费模型
- 提供灵活的转发策略配置，适用于多种网络场景

---

## 技术栈

| 组件     | 技术                     | 说明                  |
|----------|--------------------------|----------------------|
| 后端     | **Go 1.23** + Gin        | 高性能，单二进制部署   |
| 前端     | Vue 3 + Vite             | 现代化 SPA            |
| 数据库   | MySQL 5.7+               | utf8mb4              |
| 通信     | WebSocket + AES-256-GCM  | 节点加密通信          |
| 部署     | Docker Compose           | 一键部署              |

### 项目结构

```
flux-panel/
├── go-backend/              # Go 后端
│   ├── cmd/server/          # 主入口
│   ├── internal/
│   │   ├── config/          # 环境变量配置
│   │   ├── handler/         # HTTP 处理器
│   │   ├── middleware/      # JWT / CORS / Recovery
│   │   ├── model/           # 数据模型
│   │   ├── pkg/             # 工具 (AES / Gost / MD5)
│   │   ├── service/         # 业务逻辑
│   │   ├── task/            # 排程任务
│   │   └── ws/              # WebSocket
│   └── Dockerfile
├── vite-frontend/           # Vue 前端
├── docker-compose-go.yml    # Go 版部署
├── gost.sql                 # 数据库初始化
└── install.sh               # 节点安装脚本
```

---

## 部署流程

### 环境要求

- Docker 20.10+
- Docker Compose v2+
- 开放端口：面板端口（默认 6365）、节点通信端口

### 一、Docker Compose 部署（推荐）

#### 1. 克隆项目

```bash
git clone https://github.com/bqlpfy/flux-panel.git
cd flux-panel
```

#### 2. 配置环境变量

```bash
cp .env.example .env
```

编辑 `.env` 文件：

```env
# 数据库
DB_NAME=gost
DB_USER=gost
DB_PASSWORD=your_secure_password

# JWT 密钥（请修改为随机字符串）
JWT_SECRET=your_jwt_secret_here

# 端口
BACKEND_PORT=6365
FRONTEND_PORT=80
```

#### 3. 启动服务

```bash
docker compose -f docker-compose-go.yml --env-file .env up -d
```

#### 4. 查看状态

```bash
docker compose -f docker-compose-go.yml ps
```

服务启动后：
- **前端**：`http://your-server-ip:80`
- **后端 API**：`http://your-server-ip:6365`

#### 5. 默认管理员账号

| 项目 | 值 |
|------|-----|
| 账号 | `admin_user` |
| 密码 | `admin_user` |

> ⚠️ **首次登录后请立即修改默认密码！**

---

### 二、节点安装

#### 稳定版

```bash
curl -L https://raw.githubusercontent.com/bqlpfy/flux-panel/refs/heads/main/install.sh -o install.sh && chmod +x install.sh && ./install.sh
```

#### 带参数安装（非交互）

```bash
curl -L https://raw.githubusercontent.com/bqlpfy/flux-panel/refs/heads/main/install.sh -o install.sh && chmod +x install.sh && ./install.sh -a <面板地址:端口> -s <节点密钥>
```

---

### 三、更新与维护

#### 更新面板

```bash
cd flux-panel
git pull
docker compose -f docker-compose-go.yml --env-file .env up -d --build
```

#### 查看日志

```bash
# 后端日志
docker logs -f go-backend

# 全部日志
docker compose -f docker-compose-go.yml logs -f
```

#### 停止服务

```bash
docker compose -f docker-compose-go.yml down
```

#### 停止并清除数据（⚠️ 不可恢复）

```bash
docker compose -f docker-compose-go.yml down -v
```

---

## 配置参考

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DB_HOST` | `mysql` | 数据库地址（容器内使用 `mysql`） |
| `DB_PORT` | `3306` | 数据库端口 |
| `DB_NAME` | `gost` | 数据库名 |
| `DB_USER` | `root` | 数据库用户 |
| `DB_PASSWORD` | — | 数据库密码（**必填**） |
| `JWT_SECRET` | — | JWT 签名密钥（**必填**） |
| `JWT_EXPIRE_HOURS` | `168` | Token 过期时间（小时） |
| `SERVER_PORT` | `8080` | 后端监听端口（容器内部） |
| `BACKEND_PORT` | `6365` | 后端对外映射端口 |
| `FRONTEND_PORT` | `80` | 前端对外映射端口 |
| `CORS_ORIGINS` | `*` | CORS 允许来源 |

---

## 免责声明

本项目仅供个人学习与研究使用，基于开源项目进行二次开发。  

使用本项目所带来的任何风险均由使用者自行承担，包括但不限于：  

- 配置不当或使用错误导致的服务异常或不可用；  
- 使用本项目引发的网络攻击、封禁、滥用等行为；  
- 服务器因使用本项目被入侵、渗透、滥用导致的数据泄露、资源消耗或损失；  
- 因违反当地法律法规所产生的任何法律责任。  

本项目为开源的流量转发工具，仅限合法、合规用途。  
使用者必须确保其使用行为符合所在国家或地区的法律法规。  

**作者不对因使用本项目导致的任何法律责任、经济损失或其他后果承担责任。**  
**禁止将本项目用于任何违法或未经授权的行为，包括但不限于网络攻击、数据窃取、非法访问等。**  

如不同意上述条款，请立即停止使用本项目。  

---

## ⭐ 喝杯咖啡！（USDT）

| 网络       | 地址                                                                 |
|------------|----------------------------------------------------------------------|
| BNB(BEP20) | `0x755492c03728851bbf855daa28a1e089f9aca4d1`                          |
| TRC20      | `TYh2L3xxXpuJhAcBWnt3yiiADiCSJLgUm7`                                  |
| Aptos      | `0xf2f9fb14749457748506a8281628d556e8540d1eb586d202cd8b02b99d369ef8`  |

[![Star History Chart](https://api.star-history.com/svg?repos=bqlpfy/flux-panel&type=Date)](https://www.star-history.com/#bqlpfy/flux-panel&Date)
