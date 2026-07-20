package handler

import (
	"context"
	"encoding/json"

	"chihqiang/ccsim-svr/model"
	"chihqiang/ccsim-svr/protocol"
	"chihqiang/ccsim-svr/service"
	"chihqiang/ccsim-svr/ws"

	"github.com/chihqiang/infra-go/logger"
)

// ChatHandler 聊天消息处理器
type ChatHandler struct {
	hub            *ws.Hub
	sessionService *service.SessionService
	messageService *service.MessageService
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(hub *ws.Hub, sessionService *service.SessionService, messageService *service.MessageService) *ChatHandler {
	return &ChatHandler{
		hub:            hub,
		sessionService: sessionService,
		messageService: messageService,
	}
}

// Handle 处理聊天消息
func (h *ChatHandler) Handle(ctx context.Context, conn ws.Connection, data []byte) error {
	var msg protocol.ChatSendMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		logger.ErrorfCtx(ctx, "解析聊天消息失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	if err := msg.Validate(); err != nil {
		logger.WarnfCtx(ctx, "聊天消息校验失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	if msg.MsgType == "" {
		msg.MsgType = "text"
	}

	logger.InfofCtx(ctx, "收到聊天消息, 连接ID: %s, 会话ID: %d, 内容: %s", conn.GetID(), msg.SessionID, msg.Content)

	if conn.GetUserID() <= 0 {
		logger.ErrorfCtx(ctx, "用户未认证, 连接ID: %s", conn.GetID())
		return ws.ErrUnauthorized
	}

	if conn.GetRole() == protocol.RoleVisitor {
		return h.handleVisitorMessage(ctx, conn, &msg)
	}

	return h.handleAgentMessage(ctx, conn, &msg)
}

// handleVisitorMessage 处理访客消息
func (h *ChatHandler) handleVisitorMessage(ctx context.Context, conn ws.Connection, msg *protocol.ChatSendMessage) error {
	session, err := h.sessionService.CreateOrGetSession(ctx, conn.GetTenantNo(), conn.GetUserID(), conn.GetNickname())
	if err != nil {
		logger.ErrorfCtx(ctx, "创建会话失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	isNewSession := session.Status == model.SessionStatusPending && session.AgentID == 0

	if isNewSession {
		created := protocol.SessionCreatedMessage{
			ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgSessionCreated},
			SessionID:     session.ID,
			Status:        string(session.Status),
			CreatedAt:     session.CreatedAt.UnixMilli(),
		}
		sendJSON(ctx, conn, created)

		newSession := protocol.NewSessionMessage{
			ServerMessage:   protocol.ServerMessage{Type: protocol.ServerMsgNewSession},
			SessionID:       session.ID,
			VisitorID:       conn.GetUserID(),
			VisitorNickname: conn.GetNickname(),
			Source:          session.Source,
			IP:              session.IP,
			Platform:        session.Platform,
			LastMsgContent:  msg.Content,
			CreatedAt:       session.CreatedAt.UnixMilli(),
		}
		if data, err := mustMarshal(newSession); err == nil {
			h.hub.BroadcastToTenant(ctx, conn.GetTenantNo(), data)
		}
	}

	message, err := h.messageService.SendMessage(ctx, session.ID, model.SenderRoleVisitor, conn.GetUserID(), conn.GetNickname(), msg.Content, msg.MsgType, msg.TempID)
	if err != nil {
		logger.ErrorfCtx(ctx, "保存消息失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	// 增加客服侧未读计数
	if session.AgentID > 0 {
		h.sessionService.IncrementUnread(ctx, session.ID)
	}

	ack := protocol.ChatACKMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgChatACK},
		TempID:        msg.TempID,
		MsgID:         message.ID,
		SessionID:     session.ID,
		SeqNum:        message.SeqNum,
		CreatedAt:     message.CreatedAt.UnixMilli(),
	}
	if err := sendJSON(ctx, conn, ack); err != nil {
		return err
	}

	if session.AgentID > 0 {
		push := toChatPushMessage(message)
		push.ServerMessage = protocol.ServerMessage{Type: protocol.ServerMsgChatPush}
		if data, err := mustMarshal(push); err == nil {
			if err := h.hub.SendToUser(ctx, session.AgentID, protocol.RoleAgent, data); err != nil {
				logger.ErrorfCtx(ctx, "推送给客服失败, 客服ID: %d, 错误: %v", session.AgentID, err)
			}
		}
	}

	return nil
}

// handleAgentMessage 处理客服消息
func (h *ChatHandler) handleAgentMessage(ctx context.Context, conn ws.Connection, msg *protocol.ChatSendMessage) error {
	// 使用缓存获取对方用户ID，避免DB查询
	visitorID, _ := h.sessionService.GetPeerIDs(ctx, msg.SessionID, conn.GetUserID())

	message, err := h.messageService.SendMessage(ctx, msg.SessionID, model.SenderRoleAgent, conn.GetUserID(), conn.GetNickname(), msg.Content, msg.MsgType, msg.TempID)
	if err != nil {
		logger.ErrorfCtx(ctx, "保存消息失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	ack := protocol.ChatACKMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgChatACK},
		TempID:        msg.TempID,
		MsgID:         message.ID,
		SessionID:     msg.SessionID,
		SeqNum:        message.SeqNum,
		CreatedAt:     message.CreatedAt.UnixMilli(),
	}
	if err := sendJSON(ctx, conn, ack); err != nil {
		return err
	}

	push := toChatPushMessage(message)
	push.ServerMessage = protocol.ServerMessage{Type: protocol.ServerMsgChatPush}
	if visitorID > 0 {
		if data, err := mustMarshal(push); err == nil {
			if err := h.hub.SendToUser(ctx, visitorID, protocol.RoleVisitor, data); err != nil {
				logger.ErrorfCtx(ctx, "推送给访客失败, 访客ID: %d, 错误: %v", visitorID, err)
			}
		}
	}

	return nil
}
