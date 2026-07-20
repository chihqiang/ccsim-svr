package ws

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"chihqiang/ccsim-svr/bizctx"
	"chihqiang/ccsim-svr/protocol"

	"github.com/chihqiang/infra-go/logger"
	gws "github.com/chihqiang/infra-go/websocket"
)

// broadcastMsg Redis广播消息体
type broadcastMsg struct {
	NodeID        string        `json:"node_id"`
	TenantNo      string        `json:"tenant_no"`
	UserID        int64         `json:"user_id,omitempty"`
	Role          protocol.Role `json:"role,omitempty"`
	ExcludeConnID string        `json:"exclude_conn_id,omitempty"`
	Data          []byte        `json:"data"`
}

// userKey 生成用户唯一键
func userKey(userID int64, role protocol.Role) string {
	return string(role) + ":" + strconv.FormatInt(userID, 10)
}

// parseUserKey 从 userKey 中解析 role 和 userID
func parseUserKey(key string) (protocol.Role, int64) {
	i := -1
	for idx, c := range key {
		if c == ':' {
			i = idx
			break
		}
	}
	if i < 0 {
		return "", 0
	}
	uid, _ := strconv.ParseInt(key[i+1:], 10, 64)
	return protocol.Role(key[:i]), uid
}

// tenantRoom 租户房间名
func tenantRoom(tenantNo string) string {
	return "tenant:" + tenantNo
}

// Hub 连接管理中心（实现 gws.Handler 接口）
type Hub struct {
	server         *gws.Server
	router         RouterInterface
	connections    map[gws.ConnID]*Conn
	userConns      map[string]map[gws.ConnID]struct{}
	tenantConns    map[string]map[gws.ConnID]struct{}
	mu             sync.RWMutex
	distributor    Distributor
	nodeID         string
	clusterMode    bool
	onAgentOffline func(ctx context.Context, agentID int64)
}

// RouterInterface 路由器接口
type RouterInterface interface {
	Route(ctx context.Context, conn Connection, data []byte) error
	RemoveLimiter(connID string)
}

// NewHub 创建连接管理中心
func NewHub(serverConfig gws.Config, opts ...gws.Option) *Hub {
	h := &Hub{
		connections: make(map[gws.ConnID]*Conn),
		userConns:   make(map[string]map[gws.ConnID]struct{}),
		tenantConns: make(map[string]map[gws.ConnID]struct{}),
	}
	h.server = gws.MustNew(serverConfig, h, opts...)
	return h
}

// SetRouter 设置消息路由器
func (h *Hub) SetRouter(r RouterInterface) {
	h.router = r
}

// Server 获取底层 infra-go Server
func (h *Hub) Server() *gws.Server {
	return h.server
}

// SetDistributor 设置分布式分发器
func (h *Hub) SetDistributor(ctx context.Context, d Distributor, nodeID string) {
	h.distributor = d
	h.nodeID = nodeID
}

// SetClusterMode 设置集群模式
func (h *Hub) SetClusterMode(v bool) {
	h.clusterMode = v
}

// SetOnAgentOffline 设置客服下线回调（仅 standalone 模式使用）
func (h *Hub) SetOnAgentOffline(fn func(ctx context.Context, agentID int64)) {
	h.onAgentOffline = fn
}

// GetDistributor 获取分布式分发器
func (h *Hub) GetDistributor(ctx context.Context) Distributor {
	return h.distributor
}

// HandleOpen 连接建立时调用
func (h *Hub) HandleOpen(conn *gws.Conn) {
	wsConn := NewConn(conn, func(ctx context.Context) {
		h.Remove(ctx, conn.ID())
	})
	h.mu.Lock()
	h.connections[conn.ID()] = wsConn
	h.mu.Unlock()
	logger.InfofCtx(context.Background(), "WebSocket连接已添加, 连接ID: %s", wsConn.GetID())
}

// HandleClose 连接关闭时调用
func (h *Hub) HandleClose(conn *gws.Conn, err error) {
	wsConn := h.removeConn(conn.ID())
	if wsConn != nil && h.router != nil {
		h.router.RemoveLimiter(wsConn.GetID())
	}
}

// HandleMessage 收到消息时调用（解析 event type 并路由）
func (h *Hub) HandleMessage(conn *gws.Conn, messageType int, data []byte) {
	if messageType != gws.TextMessage {
		return
	}

	// 恢复 trace context
	ctx := context.Background()
	if v, ok := conn.Get("traceCtx"); ok {
		if traceCtx, ok := v.(context.Context); ok {
			ctx = traceCtx
		}
	}

	// 获取包装连接
	h.mu.RLock()
	wsConn, ok := h.connections[conn.ID()]
	h.mu.RUnlock()
	if !ok {
		return
	}
	wsConn.UpdateHeartbeat(ctx)

	// 解析 event type 并设置到 context
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil || envelope.Type == "" {
		h.sendError(ctx, wsConn, ErrUnknownMessageType)
		return
	}
	ctx = bizctx.WithMsgType(ctx, envelope.Type)

	// 路由到 handler
	if h.router != nil {
		if err := h.router.Route(ctx, wsConn, data); err != nil {
			h.sendError(ctx, wsConn, err)
		}
	}
}

// HandleError 连接错误时调用
func (h *Hub) HandleError(conn *gws.Conn, err error) {
	logger.WarnfCtx(context.Background(), "WebSocket连接错误, 连接ID: %d, 错误: %v", conn.ID(), err)
}

// sendError 发送错误消息给客户端
func (h *Hub) sendError(ctx context.Context, conn Connection, err error) {
	code := "UNKNOWN_ERROR"
	errMsg := err.Error()
	if wsErr, ok := err.(*WsError); ok {
		code = wsErr.Code
		errMsg = wsErr.Message
	}
	msg := protocol.ErrorMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgError},
		Code:          code,
		ErrMsg:        errMsg,
	}
	data, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		logger.ErrorfCtx(ctx, "序列化错误消息失败: %v", marshalErr)
		return
	}
	if sendErr := conn.Send(ctx, data); sendErr != nil {
		logger.ErrorfCtx(ctx, "发送错误消息失败: %v", sendErr)
	}
}

// removeConn 移除连接并返回（内部方法，HandleClose/Remove 共用）
func (h *Hub) removeConn(connID gws.ConnID) *Conn {
	h.mu.Lock()
	defer h.mu.Unlock()

	wsConn, ok := h.connections[connID]
	if !ok {
		return nil
	}

	userID := wsConn.GetUserID()
	role := wsConn.GetRole()
	tenantNo := wsConn.GetTenantNo()

	// 清理用户索引
	if userID > 0 {
		key := userKey(userID, role)
		if conns, exists := h.userConns[key]; exists {
			delete(conns, connID)
			if len(conns) == 0 {
				delete(h.userConns, key)
				if h.distributor != nil {
					go func(uid int64) {
						if err := h.distributor.UnregisterUser(context.Background(), uid); err != nil {
							logger.ErrorfCtx(context.Background(), "注销用户Redis注册失败, 用户ID: %d, 错误: %v", uid, err)
						}
					}(userID)
				}
				// 客服离线处理
				if role == protocol.RoleAgent {
					if h.clusterMode && h.distributor != nil {
						go func(uid int64, tNo string) {
							h.distributor.UnregisterAgent(context.Background(), tNo, uid)
							count, err := h.distributor.CountOnlineAgents(context.Background(), tNo)
							if err != nil {
								logger.ErrorfCtx(context.Background(), "统计在线客服失败, 租户: %s, 错误: %v", tNo, err)
								return
							}
							if count == 0 && h.onAgentOffline != nil {
								h.onAgentOffline(context.Background(), uid)
							}
						}(userID, tenantNo)
					} else if h.onAgentOffline != nil {
						go h.onAgentOffline(context.Background(), userID)
					}
				}
			}
		}
	}

	// 清理租户索引
	if tenantNo != "" {
		if conns, exists := h.tenantConns[tenantNo]; exists {
			delete(conns, connID)
			if len(conns) == 0 {
				delete(h.tenantConns, tenantNo)
			}
		}
	}

	delete(h.connections, connID)
	return wsConn
}

// Remove 移除连接
func (h *Hub) Remove(ctx context.Context, connID gws.ConnID) {
	wsConn := h.removeConn(connID)
	if wsConn != nil {
		logger.InfofCtx(ctx, "WebSocket连接已移除, 连接ID: %s", wsConn.GetID())
	}
}

// GetByID 根据ID获取连接
func (h *Hub) GetByID(ctx context.Context, connID string) (Connection, bool) {
	id, err := parseConnID(connID)
	if err != nil {
		return nil, false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	conn, ok := h.connections[id]
	return conn, ok
}

// GetByUserID 根据用户ID获取连接
func (h *Hub) GetByUserID(ctx context.Context, userID int64) (Connection, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, role := range []protocol.Role{protocol.RoleVisitor, protocol.RoleAgent} {
		key := userKey(userID, role)
		if conns, ok := h.userConns[key]; ok {
			for connID := range conns {
				if conn, exists := h.connections[connID]; exists {
					return conn, true
				}
			}
		}
	}
	return nil, false
}

// BindUser 绑定用户
func (h *Hub) BindUser(ctx context.Context, userID int64, role protocol.Role, conn Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	id, _ := parseConnID(conn.GetID())
	if _, ok := h.connections[id]; !ok {
		return
	}

	key := userKey(userID, role)
	if h.userConns[key] == nil {
		h.userConns[key] = make(map[gws.ConnID]struct{})
	}
	h.userConns[key][id] = struct{}{}
	logger.InfofCtx(ctx, "用户已绑定连接, 用户ID: %d, 角色: %s, 连接ID: %s", userID, role, conn.GetID())

	// 加入租户房间
	tenantNo := conn.GetTenantNo()
	if tenantNo != "" {
		h.server.Room().Add(id, tenantRoom(tenantNo))
		if h.tenantConns[tenantNo] == nil {
			h.tenantConns[tenantNo] = make(map[gws.ConnID]struct{})
		}
		h.tenantConns[tenantNo][id] = struct{}{}
	}

	// 注册用户到 Redis
	if h.distributor != nil {
		go func(uid int64) {
			if err := h.distributor.RegisterUser(context.Background(), uid); err != nil {
				logger.ErrorfCtx(context.Background(), "注册用户到Redis失败, 用户ID: %d, 错误: %v", uid, err)
			}
		}(userID)
	}
}

// UnbindUser 解绑用户所有连接
func (h *Hub) UnbindUser(ctx context.Context, userID int64, role protocol.Role) {
	h.mu.Lock()
	defer h.mu.Unlock()
	key := userKey(userID, role)
	delete(h.userConns, key)
	logger.InfofCtx(ctx, "用户已解绑所有连接, 用户ID: %d, 角色: %s", userID, role)
}

// SendToUser 发送消息给用户（本地优先，远程 fallback）
func (h *Hub) SendToUser(ctx context.Context, userID int64, role protocol.Role, data []byte) error {
	key := userKey(userID, role)
	h.mu.RLock()
	var connIDs []gws.ConnID
	if conns, ok := h.userConns[key]; ok {
		for id := range conns {
			connIDs = append(connIDs, id)
		}
	}
	h.mu.RUnlock()

	sent := false
	for _, id := range connIDs {
		h.mu.RLock()
		wsConn, ok := h.connections[id]
		h.mu.RUnlock()
		if ok && !wsConn.IsClosed() {
			if err := wsConn.Send(ctx, data); err == nil {
				sent = true
			}
		}
	}
	if sent {
		return nil
	}

	if h.distributor != nil {
		nodeID, err := h.distributor.GetUserNode(ctx, userID)
		if err != nil || nodeID == "" {
			return ErrConnectionNotFound
		}
		msg := broadcastMsg{NodeID: h.nodeID, UserID: userID, Role: role, Data: data}
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			logger.ErrorfCtx(ctx, "序列化远程节点消息失败: %v", err)
			return err
		}
		return h.distributor.PublishToNode(ctx, nodeID, msgBytes)
	}

	return ErrConnectionNotFound
}

// SendToConn 发送消息给连接
func (h *Hub) SendToConn(ctx context.Context, connID string, data []byte) error {
	id, err := parseConnID(connID)
	if err != nil {
		return ErrConnectionNotFound
	}
	h.mu.RLock()
	wsConn, ok := h.connections[id]
	h.mu.RUnlock()
	if !ok {
		return ErrConnectionNotFound
	}
	return wsConn.Send(ctx, data)
}

// BroadcastToTenant 广播消息给租户
func (h *Hub) BroadcastToTenant(ctx context.Context, tenantNo string, data []byte) {
	h.server.To(tenantRoom(tenantNo)).Push(data)

	if h.distributor != nil {
		msg := broadcastMsg{NodeID: h.nodeID, TenantNo: tenantNo, Data: data}
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			logger.ErrorfCtx(ctx, "序列化租户广播消息失败: %v", err)
			return
		}
		if err := h.distributor.PublishToTenant(ctx, tenantNo, msgBytes); err != nil {
			logger.ErrorfCtx(ctx, "Redis租户广播失败, 租户: %s, 错误: %v", tenantNo, err)
		}
	}
}

// BroadcastToTenantExcept 广播消息给租户（排除指定连接）
func (h *Hub) BroadcastToTenantExcept(ctx context.Context, tenantNo string, data []byte, excludeConnID string) {
	excludeID, _ := parseConnID(excludeConnID)

	h.mu.RLock()
	var conns []Connection
	if tc, ok := h.tenantConns[tenantNo]; ok {
		for id := range tc {
			if id == excludeID {
				continue
			}
			if wsConn, exists := h.connections[id]; exists {
				conns = append(conns, wsConn)
			}
		}
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		if err := conn.Send(ctx, data); err != nil {
			logger.ErrorfCtx(ctx, "租户广播消息失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		}
	}

	if h.distributor != nil {
		msg := broadcastMsg{NodeID: h.nodeID, TenantNo: tenantNo, ExcludeConnID: excludeConnID, Data: data}
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			logger.ErrorfCtx(ctx, "序列化租户广播消息失败: %v", err)
			return
		}
		if err := h.distributor.PublishToTenant(ctx, tenantNo, msgBytes); err != nil {
			logger.ErrorfCtx(ctx, "Redis租户广播失败, 租户: %s, 错误: %v", tenantNo, err)
		}
	}
}

// HandleRemoteNodeMessage 处理来自其他节点的消息
func (h *Hub) HandleRemoteNodeMessage(ctx context.Context, data []byte) {
	var msg broadcastMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		logger.ErrorfCtx(ctx, "解析远程节点消息失败: %v", err)
		return
	}
	if msg.UserID > 0 && msg.Role != "" {
		h.mu.RLock()
		var conns []Connection
		key := userKey(msg.UserID, msg.Role)
		if uc, ok := h.userConns[key]; ok {
			for id := range uc {
				if wsConn, exists := h.connections[id]; exists {
					conns = append(conns, wsConn)
				}
			}
		}
		h.mu.RUnlock()
		for _, conn := range conns {
			if err := conn.Send(ctx, msg.Data); err != nil {
				logger.ErrorfCtx(ctx, "远程节点消息投递失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
			}
		}
		return
	}
	logger.WarnfCtx(ctx, "忽略格式不完整的远程节点消息")
}

// HandleRemoteTenantMessage 处理来自其他节点的租户广播
func (h *Hub) HandleRemoteTenantMessage(ctx context.Context, data []byte) {
	var msg broadcastMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		logger.ErrorfCtx(ctx, "解析远程租户广播失败: %v", err)
		return
	}
	if msg.NodeID == h.nodeID {
		return
	}

	excludeID, _ := parseConnID(msg.ExcludeConnID)

	h.mu.RLock()
	var conns []Connection
	if tc, ok := h.tenantConns[msg.TenantNo]; ok {
		for id := range tc {
			if id == excludeID {
				continue
			}
			if wsConn, exists := h.connections[id]; exists {
				conns = append(conns, wsConn)
			}
		}
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		if err := conn.Send(ctx, msg.Data); err != nil {
			logger.ErrorfCtx(ctx, "远程租户广播投递失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		}
	}
}

// RefreshUserTTL 续期所有在线用户的Redis注册
func (h *Hub) RefreshUserTTL(ctx context.Context) {
	if h.distributor == nil {
		return
	}
	h.mu.RLock()
	seen := make(map[int64]struct{}, 64)
	for key := range h.userConns {
		_, uid := parseUserKey(key)
		if uid > 0 {
			seen[uid] = struct{}{}
		}
	}
	h.mu.RUnlock()
	if len(seen) == 0 {
		return
	}
	userIDs := make([]int64, 0, len(seen))
	for uid := range seen {
		userIDs = append(userIDs, uid)
	}
	if err := h.distributor.RefreshUsers(ctx, userIDs); err != nil {
		logger.ErrorfCtx(ctx, "批量续期用户Redis注册失败, 用户数: %d, 错误: %v", len(userIDs), err)
	}
}

// Count 连接数量
func (h *Hub) Count(ctx context.Context) int {
	return h.server.Count()
}

// StartCleanup 启动清理协程
func (h *Hub) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.RefreshUserTTL(ctx)
			}
		}
	}()
}

// Shutdown 优雅关闭
func (h *Hub) Shutdown(ctx context.Context) {
	h.mu.RLock()
	seen := make(map[int64]struct{}, 64)
	var conns []Connection
	for key := range h.userConns {
		_, uid := parseUserKey(key)
		if uid > 0 {
			seen[uid] = struct{}{}
		}
	}
	for _, conn := range h.connections {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		if err := conn.Close(ctx); err != nil {
			logger.ErrorfCtx(ctx, "关闭连接失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		}
	}

	for uid := range seen {
		if h.distributor != nil {
			if err := h.distributor.UnregisterUser(context.Background(), uid); err != nil {
				logger.ErrorfCtx(context.Background(), "关闭时注销用户失败, 用户ID: %d, 错误: %v", uid, err)
			}
		}
	}

	h.mu.Lock()
	h.connections = make(map[gws.ConnID]*Conn)
	h.userConns = make(map[string]map[gws.ConnID]struct{})
	h.tenantConns = make(map[string]map[gws.ConnID]struct{})
	h.mu.Unlock()

	logger.InfofCtx(ctx, "Hub已关闭, 注销用户数: %d, 关闭连接数: %d", len(seen), len(conns))
}

// parseConnID 解析 string ConnID 为 uint64
func parseConnID(s string) (gws.ConnID, error) {
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return gws.ConnID(id), nil
}

// 错误定义
var (
	ErrConnectionNotFound = &WsError{Code: "CONNECTION_NOT_FOUND", Message: "连接不存在"}
)

// 确保 Hub 实现 gws.Handler 接口
var _ gws.Handler = (*Hub)(nil)
