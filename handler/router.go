package handler

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"

	"chihqiang/ccsim-svr/bizctx"
	"chihqiang/ccsim-svr/protocol"
	"chihqiang/ccsim-svr/ws"

	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/ratelimit"
	"github.com/chihqiang/infra-go/trace"
)

// 需要认证的消息类型（userID 必须 > 0）
var authRequiredTypes = map[protocol.ClientMessageType]bool{
	protocol.ClientMsgChatSend:         true,
	protocol.ClientMsgAgentOnline:      true,
	protocol.ClientMsgAgentOffline:     true,
	protocol.ClientMsgSessionAccept:    true,
	protocol.ClientMsgSessionClose:     true,
	protocol.ClientMsgSessionList:      true,
	protocol.ClientMsgSessionHistory:   true,
	protocol.ClientMsgWaitingList:      true,
	protocol.ClientMsgSatisfactionRate: true,
	protocol.ClientMsgVisitorUpdate:    true,
	protocol.ClientMsgTyping:           true,
	protocol.ClientMsgMessageRead:      true,
}

// 角色-消息类型权限映射
var roleAllowedTypes = map[protocol.Role]map[protocol.ClientMessageType]bool{
	protocol.RoleVisitor: {
		protocol.ClientMsgHeartbeat:        true,
		protocol.ClientMsgChatSend:         true,
		protocol.ClientMsgSessionClose:     true,
		protocol.ClientMsgSessionHistory:   true,
		protocol.ClientMsgSatisfactionRate: true,
		protocol.ClientMsgVisitorUpdate:    true,
		protocol.ClientMsgTyping:           true,
		protocol.ClientMsgMessageRead:      true,
	},
	protocol.RoleAgent: {
		protocol.ClientMsgHeartbeat:      true,
		protocol.ClientMsgChatSend:       true,
		protocol.ClientMsgAgentOnline:    true,
		protocol.ClientMsgAgentOffline:   true,
		protocol.ClientMsgSessionAccept:  true,
		protocol.ClientMsgSessionClose:   true,
		protocol.ClientMsgSessionList:    true,
		protocol.ClientMsgSessionHistory: true,
		protocol.ClientMsgWaitingList:    true,
		protocol.ClientMsgTyping:         true,
		protocol.ClientMsgMessageRead:    true,
	},
}

// Router 消息路由器
type Router struct {
	handlers map[protocol.ClientMessageType]ws.MessageHandler
	limiters map[string]ratelimit.Limiter // connID -> limiter（每连接独立限流）
	mu       sync.RWMutex
}

// NewRouter 创建路由器
func NewRouter() *Router {
	return &Router{
		handlers: make(map[protocol.ClientMessageType]ws.MessageHandler),
		limiters: make(map[string]ratelimit.Limiter),
	}
}

// getLimiter 获取连接的限流器（每连接100/s，桶容量200）
func (r *Router) getLimiter(connID string) ratelimit.Limiter {
	r.mu.RLock()
	limiter, ok := r.limiters[connID]
	r.mu.RUnlock()
	if ok {
		return limiter
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	// double check
	if limiter, ok = r.limiters[connID]; ok {
		return limiter
	}
	limiter = ratelimit.NewTokenBucket(200, 100)
	r.limiters[connID] = limiter
	return limiter
}

// RemoveLimiter 移除连接的限流器（连接关闭时调用）
func (r *Router) RemoveLimiter(connID string) {
	r.mu.Lock()
	delete(r.limiters, connID)
	r.mu.Unlock()
}

// Register 注册消息处理器
func (r *Router) Register(msgType protocol.ClientMessageType, handler ws.MessageHandler) {
	r.handlers[msgType] = handler
}

// Route 路由消息
func (r *Router) Route(ctx context.Context, conn ws.Connection, data []byte) error {
	if !r.getLimiter(conn.GetID()).Allow() {
		return ws.ErrRateLimited
	}

	var base protocol.ClientMessage
	if err := json.Unmarshal(data, &base); err != nil {
		logger.ErrorfCtx(ctx, "解析消息类型失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	ctx = bizctx.WithConnID(ctx, conn.GetID())
	ctx = bizctx.WithTenantNo(ctx, conn.GetTenantNo())
	ctx = bizctx.WithUserID(ctx, conn.GetUserID())
	ctx = bizctx.WithRole(ctx, string(conn.GetRole()))
	ctx = bizctx.WithMsgType(ctx, string(base.Type))
	ctx = bizctx.WithSessionID(ctx, base.SessionID)

	ctx, span := trace.StartSpan(ctx, "ws.handler."+string(base.Type),
		trace.WithAttributes(
			trace.AttrString("msg_type", string(base.Type)),
			trace.AttrString("conn_id", conn.GetID()),
			trace.AttrString("user_id", strconv.FormatInt(conn.GetUserID(), 10)),
			trace.AttrString("role", string(conn.GetRole())),
		),
	)
	defer span.End()

	// 权限校验：auth 和 heartbeat 不需要认证
	if authRequiredTypes[base.Type] && conn.GetUserID() <= 0 {
		logger.WarnfCtx(ctx, "未认证连接发送需要认证的消息, 连接ID: %s, 类型: %s", conn.GetID(), base.Type)
		return ws.ErrUnauthorized
	}

	// 权限校验：角色-消息类型匹配
	if conn.GetUserID() > 0 {
		allowed, exists := roleAllowedTypes[conn.GetRole()]
		if exists && !allowed[base.Type] {
			logger.WarnfCtx(ctx, "角色无权发送该消息类型, 连接ID: %s, 角色: %s, 类型: %s", conn.GetID(), conn.GetRole(), base.Type)
			return ws.ErrInvalidRole
		}
	}

	handler, ok := r.handlers[base.Type]
	if !ok {
		logger.WarnfCtx(ctx, "未找到消息处理器, 连接ID: %s, 类型: %s", conn.GetID(), base.Type)
		return ws.ErrUnknownMessageType
	}

	logger.DebugfCtx(ctx, "路由消息, 连接ID: %s, 类型: %s", conn.GetID(), base.Type)
	return handler.Handle(ctx, conn, data)
}
