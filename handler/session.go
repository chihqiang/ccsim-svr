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

// SessionHandler 会话处理器
type SessionHandler struct {
	hub            *ws.Hub
	sessionService *service.SessionService
	messageService *service.MessageService
	agentService   *service.AgentService
	visitorService *service.VisitorService
}

// NewSessionHandler 创建会话处理器
func NewSessionHandler(hub *ws.Hub, sessionService *service.SessionService, messageService *service.MessageService, agentService *service.AgentService, visitorService *service.VisitorService) *SessionHandler {
	return &SessionHandler{
		hub:            hub,
		sessionService: sessionService,
		messageService: messageService,
		agentService:   agentService,
		visitorService: visitorService,
	}
}

// Handle 处理会话消息
func (h *SessionHandler) Handle(ctx context.Context, conn ws.Connection, data []byte) error {
	msgType := bizctx.MsgTypeFrom(ctx)

	switch protocol.ClientMessageType(msgType) {
	case protocol.ClientMsgSessionAccept:
		return h.handleAccept(ctx, conn, data)
	case protocol.ClientMsgSessionClose:
		return h.handleClose(ctx, conn, data)
	case protocol.ClientMsgSessionList:
		return h.handleList(ctx, conn, data)
	case protocol.ClientMsgSessionHistory:
		return h.handleHistory(ctx, conn, data)
	case protocol.ClientMsgWaitingList:
		return h.handleWaitingList(ctx, conn, data)
	default:
		return ws.ErrUnknownMessageType
	}
}

// handleAccept 处理会话接受
func (h *SessionHandler) handleAccept(ctx context.Context, conn ws.Connection, data []byte) error {
	var msg protocol.SessionAcceptMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	if err := msg.Validate(); err != nil {
		return err
	}

	logger.InfofCtx(ctx, "客服接受会话, 连接ID: %s, 会话ID: %d", conn.GetID(), msg.SessionID)

	session, err := h.sessionService.AcceptSession(ctx, msg.SessionID, conn.GetUserID())
	if err != nil {
		logger.ErrorfCtx(ctx, "接受会话失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	agentNickname := session.AgentNickname

	// 发送系统消息通知访客
	h.messageService.SendSystemMessage(ctx, session.ID, "客服已接入会话")

	assigned := protocol.SessionAssignedMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgSessionAssigned},
		SessionID:     session.ID,
		AgentID:       conn.GetUserID(),
		AgentNickname: agentNickname,
		AgentAvatar:   "",
	}
	dataAssigned, err := mustMarshal(assigned)
	if err != nil {
		return err
	}

	// 1. 通知访客 session_assigned
	h.hub.SendToUser(ctx, session.VisitorID, protocol.RoleVisitor, dataAssigned)

	// 2. 广播 session_assigned 给同租户所有其他客服（排除操作者本人）
	h.hub.BroadcastToTenantExcept(ctx, conn.GetTenantNo(), dataAssigned, conn.GetID())

	// 3. 返回 session_accept 确认给操作者
	ack := protocol.SessionAcceptAckMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgSessionAcceptACK},
		SessionID:     session.ID,
		Status:        string(session.Status),
	}
	return sendJSON(ctx, conn, ack)
}

// handleClose 处理会话关闭
func (h *SessionHandler) handleClose(ctx context.Context, conn ws.Connection, data []byte) error {
	var msg protocol.SessionCloseMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	if err := msg.Validate(); err != nil {
		return err
	}

	logger.InfofCtx(ctx, "关闭会话, 连接ID: %s, 会话ID: %d", conn.GetID(), msg.SessionID)

	closeReason := protocol.CloseReasonAgent
	if conn.GetRole() == protocol.RoleVisitor {
		closeReason = protocol.CloseReasonVisitor
	}

	session, err := h.sessionService.CloseSession(ctx, msg.SessionID, string(closeReason))
	if err != nil {
		logger.ErrorfCtx(ctx, "关闭会话失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	// 发送系统消息
	h.messageService.SendSystemMessage(ctx, session.ID, "会话已关闭")

	closed := protocol.SessionClosedMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgSessionClosed},
		SessionID:     session.ID,
		CloseReason:   string(closeReason),
	}
	closedBytes, err := mustMarshal(closed)
	if err != nil {
		return err
	}

	// 1. 广播 session_closed 给同租户所有客服
	h.hub.BroadcastToTenant(ctx, conn.GetTenantNo(), closedBytes)

	// 2. 通知访客
	h.hub.SendToUser(ctx, session.VisitorID, protocol.RoleVisitor, closedBytes)

	// 3. 返回 session_close 确认给操作者
	ack := protocol.SessionCloseAckMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgSessionCloseACK},
		SessionID:     session.ID,
		CloseReason:   string(closeReason),
	}
	return sendJSON(ctx, conn, ack)
}

// handleList 处理会话列表
func (h *SessionHandler) handleList(ctx context.Context, conn ws.Connection, data []byte) error {
	var msg protocol.SessionListMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	logger.InfofCtx(ctx, "请求会话列表, 连接ID: %s, 页码: %d, 状态: %s", conn.GetID(), msg.Page, msg.Status)

	sessions, total, err := h.sessionService.ListSessions(ctx, conn.GetTenantNo(), msg.Status, msg.Page, msg.Limit)
	if err != nil {
		logger.ErrorfCtx(ctx, "获取会话列表失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	lastMsgMap, visitorMap := h.fetchSessionExtras(ctx, sessions)
	items := buildSessionListItems(sessions, lastMsgMap, visitorMap)

	resp := protocol.SessionListResMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgSessionListRes},
		Items:         items,
		Total:         int(total),
		Page:          msg.Page,
		Limit:         msg.Limit,
	}
	return sendJSON(ctx, conn, resp)
}

// handleHistory 处理历史消息
func (h *SessionHandler) handleHistory(ctx context.Context, conn ws.Connection, data []byte) error {
	var msg protocol.SessionHistoryMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	if err := msg.Validate(); err != nil {
		return err
	}

	logger.InfofCtx(ctx, "请求历史消息, 连接ID: %s, 会话ID: %d", conn.GetID(), msg.SessionID)

	limit := msg.Limit
	if limit <= 0 {
		limit = 20
	}

	messages, err := h.messageService.GetHistoryMessages(ctx, msg.SessionID, msg.BeforeSeq, limit)
	if err != nil {
		logger.ErrorfCtx(ctx, "获取历史消息失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	items := make([]protocol.ChatPushMessage, 0, len(messages))
	for _, m := range messages {
		items = append(items, toChatPushMessage(m))
	}

	resp := protocol.HistoryBatchMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgHistoryBatch},
		SessionID:     msg.SessionID,
		Count:         len(items),
		Data:          items,
	}
	return sendJSON(ctx, conn, resp)
}

// handleWaitingList 处理等待列表
func (h *SessionHandler) handleWaitingList(ctx context.Context, conn ws.Connection, data []byte) error {
	logger.InfofCtx(ctx, "请求等待列表, 连接ID: %s", conn.GetID())

	sessions, err := h.sessionService.GetWaitingSessions(ctx, conn.GetTenantNo())
	if err != nil {
		logger.ErrorfCtx(ctx, "获取等待列表失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	lastMsgMap, visitorMap := h.fetchSessionExtras(ctx, sessions)
	items := buildWaitingListItems(sessions, lastMsgMap, visitorMap)

	resp := protocol.WaitingSessionListResMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgWaitingListRes},
		Items:         items,
	}
	return sendJSON(ctx, conn, resp)
}
