package handler

import (
	"context"
	"encoding/json"

	"chihqiang/ccsim-svr/bizctx"
	"chihqiang/ccsim-svr/protocol"
	"chihqiang/ccsim-svr/service"
	"chihqiang/ccsim-svr/ws"

	"github.com/chihqiang/infra-go/logger"
)

// AgentHandler 客服状态处理器
type AgentHandler struct {
	hub            *ws.Hub
	agentService   *service.AgentService
	sessionService *service.SessionService
	messageService *service.MessageService
}

// NewAgentHandler 创建客服处理器
func NewAgentHandler(hub *ws.Hub, agentService *service.AgentService, sessionService *service.SessionService, messageService *service.MessageService) *AgentHandler {
	return &AgentHandler{
		hub:            hub,
		agentService:   agentService,
		sessionService: sessionService,
		messageService: messageService,
	}
}

// Handle 处理客服消息
func (h *AgentHandler) Handle(ctx context.Context, conn ws.Connection, data []byte) error {
	msgType := bizctx.MsgTypeFrom(ctx)

	switch protocol.ClientMessageType(msgType) {
	case protocol.ClientMsgAgentOnline:
		return h.handleOnline(ctx, conn)
	case protocol.ClientMsgAgentOffline:
		return h.handleOffline(ctx, conn)
	case protocol.ClientMsgTyping:
		return h.handleTyping(ctx, conn, data)
	case protocol.ClientMsgMessageRead:
		return h.handleRead(ctx, conn, data)
	default:
		return ws.ErrUnknownMessageType
	}
}

// broadcastAgentStatus 广播客服在线状态给同租户所有客服
func (h *AgentHandler) broadcastAgentStatus(ctx context.Context, tenantNo string) {
	count, err := h.agentService.GetOnlineCount(ctx, tenantNo)
	if err != nil {
		logger.ErrorfCtx(ctx, "获取在线客服数失败, 租户: %s, 错误: %v", tenantNo, err)
		return
	}

	msg := protocol.AgentStatusMessage{
		ServerMessage:  protocol.ServerMessage{Type: protocol.ServerMsgAgentStatus},
		OnlineCount:    int(count),
		HasOnlineAgent: count > 0,
	}
	data, _ := json.Marshal(msg)
	h.hub.BroadcastToTenant(ctx, tenantNo, data)
}

// handleOnline 处理客服上线
func (h *AgentHandler) handleOnline(ctx context.Context, conn ws.Connection) error {
	logger.InfofCtx(ctx, "客服上线, 连接ID: %s, 用户ID: %d", conn.GetID(), conn.GetUserID())

	if err := h.agentService.SetOnline(ctx, conn.GetUserID(), true, conn.GetTenantNo()); err != nil {
		logger.ErrorfCtx(ctx, "设置客服上线失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	ack := protocol.AgentOnlineAckMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgAgentOnlineAck},
		AgentID:       conn.GetUserID(),
		Status:        string(protocol.AgentOnlineStatusOnline),
	}
	if err := sendJSON(ctx, conn, ack); err != nil {
		return err
	}

	h.broadcastAgentStatus(ctx, conn.GetTenantNo())
	return nil
}

// handleOffline 处理客服下线
func (h *AgentHandler) handleOffline(ctx context.Context, conn ws.Connection) error {
	logger.InfofCtx(ctx, "客服下线, 连接ID: %s, 用户ID: %d", conn.GetID(), conn.GetUserID())

	if err := h.agentService.SetOnline(ctx, conn.GetUserID(), false, conn.GetTenantNo()); err != nil {
		logger.ErrorfCtx(ctx, "设置客服下线失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	ack := protocol.AgentOfflineAckMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgAgentOfflineAck},
		AgentID:       conn.GetUserID(),
		Status:        string(protocol.AgentOnlineStatusOffline),
	}
	if err := sendJSON(ctx, conn, ack); err != nil {
		return err
	}

	h.broadcastAgentStatus(ctx, conn.GetTenantNo())
	return nil
}

// handleTyping 处理正在输入（使用缓存获取对方用户ID，避免DB查询）
func (h *AgentHandler) handleTyping(ctx context.Context, conn ws.Connection, data []byte) error {
	var msg protocol.TypingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	if err := msg.Validate(); err != nil {
		return err
	}

	logger.DebugfCtx(ctx, "正在输入, 连接ID: %s, 会话ID: %d", conn.GetID(), msg.SessionID)

	targetUserID, targetRole := h.sessionService.GetPeerIDs(ctx, msg.SessionID, conn.GetUserID())

	push := protocol.TypingPushMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgTypingPush},
		SessionID:     msg.SessionID,
		SenderRole:    string(conn.GetRole()),
		SenderID:      conn.GetUserID(),
	}

	if targetUserID > 0 {
		if data, err := mustMarshal(push); err == nil {
			if err := h.hub.SendToUser(ctx, targetUserID, targetRole, data); err != nil {
				logger.ErrorfCtx(ctx, "推送输入状态失败, 目标用户ID: %d, 错误: %v", targetUserID, err)
			}
		}
	}

	return nil
}

// handleRead 处理消息已读（使用缓存获取对方用户ID，避免DB查询）
func (h *AgentHandler) handleRead(ctx context.Context, conn ws.Connection, data []byte) error {
	var msg protocol.MessageReadMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	if err := msg.Validate(); err != nil {
		return err
	}

	logger.DebugfCtx(ctx, "消息已读, 连接ID: %s, 会话ID: %d, 消息ID: %d", conn.GetID(), msg.SessionID, msg.MsgID)

	if err := h.messageService.MarkAsRead(ctx, msg.SessionID, msg.MsgID); err != nil {
		logger.ErrorfCtx(ctx, "标记消息已读失败, 会话ID: %d, 错误: %v", msg.SessionID, err)
	}

	targetUserID, targetRole := h.sessionService.GetPeerIDs(ctx, msg.SessionID, conn.GetUserID())

	push := protocol.MessageReadPushMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgMessageReadPush},
		SessionID:     msg.SessionID,
		ReaderRole:    string(conn.GetRole()),
		ReaderID:      conn.GetUserID(),
		MsgID:         msg.MsgID,
		SeqNum:        msg.SeqNum,
	}

	if targetUserID > 0 {
		if data, err := mustMarshal(push); err == nil {
			if err := h.hub.SendToUser(ctx, targetUserID, targetRole, data); err != nil {
				logger.ErrorfCtx(ctx, "推送已读状态失败, 目标用户ID: %d, 错误: %v", targetUserID, err)
			}
		}
	}

	return nil
}
