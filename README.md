# ccsim-svr

ccsim-ui 配套 IM 服务端，基于 Go 语言开发，提供 WebSocket 长连接管理、会话分配、消息路由、离线推送等能力，支持单机与 Redis 集群部署。

## 特性

- **实时通信** — 基于 `infra-go/websocket`（gws）的高性能 WebSocket 服务
- **访客/客服双角色** — 认证鉴权、角色权限隔离、消息路由
- **会话管理** — 创建、分配、关闭、历史消息、排队
- **分布式部署** — Redis Pub/Sub 实现跨节点消息分发，节点自动注册与心跳
- **智能分配** — 最少活跃会话优先分配客服
- **限流保护** — 每连接令牌桶限流（100/s，桶容量 200）
- **离线消息** — 重连后自动推送离线消息
- **满意度评价** — 会话结束后访客可评分
- **数据库可切换** — GORM 模型，支持 SQLite / MySQL / PostgreSQL

## 快速开始

### 本地运行

```bash
git clone https://github.com/chihqiang/ccsim-svr.git
cd ccsim-svr
go run .
```

默认加载项目根目录 `config.yaml`，启动后监听 `0.0.0.0:8080`。

### Docker

```bash
docker build -t ccsim-svr .
docker run -p 8080:8080 ccsim-svr
```

## 配置

通过 `-config` 参数或 `CONFIG_PATH` 环境变量指定配置文件路径，默认 `config.yaml`。

```yaml
env: development

server:
  host: 0.0.0.0
  port: 8080
  wsPath: /ws
  healthPath: /health

database:
  driver: sqlite          # sqlite / mysql / postgres
  database: ccsim_im.db   # sqlite 文件路径；mysql/postgres 为 DSN

redis:
  enabled: false          # 开启后启用分布式模式
  addr: 127.0.0.1:6379
  password: ""
  db: 0
  registryTTL: 90         # 节点注册 TTL（秒）

websocket:
  PingInterval: 30s       # 心跳间隔
  PingTimeout: 120s       # 心跳超时

trace:
  name: ccsim-svr
  disabled: true          # 生产环境建议开启
  endpoint: logs/ccsim-trace.log
  sampler: 1.0
  batcher: file

log:
  appName: ccsim-svr
  Output:
    - stdout
    - ./logs/app.log
```

## 项目结构

```
ccsim-svr/
├── main.go              # 入口，配置加载、依赖注入、优雅退出
├── app/
│   ├── app.go           # App 主体，DI 容器
│   └── server.go        # HTTP/WebSocket 服务器
├── handler/             # 消息处理器
│   ├── auth.go          # 认证
│   ├── heartbeat.go     # 心跳
│   ├── chat.go          # 聊天消息
│   ├── session.go       # 会话操作（接受/关闭/历史/排队）
│   ├── session_list.go  # 会话列表
│   ├── agent.go         # 客服上下线/输入状态/已读
│   └── extra.go         # 满意度评价/访客信息更新
├── model/               # GORM 模型
│   ├── tenant.go
│   ├── visitor.go
│   ├── agent.go
│   ├── session.go
│   ├── message.go
│   └── satisfaction.go
├── repo/                # 数据访问层
├── service/             # 业务逻辑层
├── ws/                  # WebSocket 核心
│   ├── conn.go          # 连接封装
│   ├── hub.go           # 连接管理、消息分发
│   ├── distributor.go   # Redis 分布式分发
│   └── conn_handler.go  # 连接生命周期回调
├── protocol/            # 消息协议定义
├── bizctx/              # 上下文工具
├── config/
│   ├── config.go        # 配置结构体
│   └── config.yaml      # 默认配置
├── Dockerfile           # 多阶段构建
└── .dockerignore
```

## 部署模式

### 单机模式

```yaml
redis:
  enabled: false
```

适用于本地开发和测试，消息在进程内直接路由。

### 集群模式

```yaml
redis:
  enabled: true
  addr: 127.0.0.1:6379
```

多个实例共享 Redis，通过 Pub/Sub 实现跨节点消息分发：

- 每个节点启动时自动注册（UUID 前 8 位作为 nodeID）
- 心跳续约，默认 TTL 90 秒
- 客服在线状态通过 Redis Set 维护（`ccsim:online:<tenantNo>`）
- 用户-连接映射通过 Redis Hash 维护（`ccsim:conn:<tenantNo>`）

## WebSocket 端点

| 路径 | 说明 |
|------|------|
| `/ws` | WebSocket 连接端点（可通过 `server.wsPath` 配置） |
| `/health` | 健康检查，返回 200 |

## 协议概览

客户端发送 JSON 消息，格式：

```json
{
  "type": "消息类型",
  "其他字段": "..."
}
```

### 支持的消息类型

| 类型 | 方向 | 说明 |
|------|------|------|
| `auth` | C→S | 认证（访客/客服） |
| `heartbeat` | C→S | 心跳 |
| `chat_send` | C→S | 发送聊天消息 |
| `session_accept` | C→S | 客服接待会话 |
| `session_close` | C→S | 关闭会话 |
| `session_list` | C→S | 请求会话列表 |
| `session_history` | C→S | 请求历史消息 |
| `waiting_session_list` | C→S | 请求排队列表 |
| `agent_online` | C→S | 客服上线 |
| `agent_offline` | C→S | 客服下线 |
| `typing` | C→S | 正在输入 |
| `message_read` | C→S | 消息已读 |
| `satisfaction_rate` | C→S | 满意度评价 |
| `visitor_update` | C→S | 更新访客信息 |
| `auth_ok` | S→C | 认证成功 |
| `heartbeat_ack` | S→C | 心跳响应 |
| `chat_ack` | S→C | 消息确认 |
| `chat_push` | S→C | 消息推送 |
| `offline_push` | S→C | 离线消息推送 |
| `history_batch` | S→C | 历史消息 |
| `session_created` | S→C | 会话创建 |
| `session_assigned` | S→C | 会话分配 |
| `session_closed` | S→C | 会话关闭 |
| `session_list_res` | S→C | 会话列表 |
| `waiting_session_list_res` | S→C | 排队列表 |
| `new_session` | S→C | 新会话通知 |
| `agent_status` | S→C | 在线客服状态 |
| `typing_push` | S→C | 正在输入推送 |
| `message_read_push` | S→C | 已读回执 |
| `satisfaction_rate` | S→C | 评价确认 |
| `visitor_update_ok` | S→C | 访客信息更新确认 |
| `visitor_info_updated` | S→C | 访客信息变更通知 |
| `error` | S→C | 错误 |

完整协议文档参见 [ccsim-ui/README.md](https://github.com/chihqiang/ccsim-ui)。

## 技术栈

- **Go 1.25+**
- **infra-go** — 日志、链路追踪、ORM、WebSocket、限流、Redis 工具库
- **GORM** — ORM 框架，支持 SQLite / MySQL / PostgreSQL
- **gws** — 基于 `infra-go/websocket` 的高性能 WebSocket 库
- **go-redis** — Redis 客户端（集群模式）
