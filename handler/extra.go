package handler

import (
	"context"
	"encoding/json"

	"chihqiang/ccsim-svr/bizctx"
	"chihqiang/ccsim-svr/model"
	"chihqiang/ccsim-svr/protocol"
	"chihqiang/ccsim-svr/service"
	"chihqiang/ccsim-svr/ws"

	"github.com/chihqiang/infra-go/logger"
)

// ExtraHandler 扩展消息处理器（满意度评价、访客信息更新）
type ExtraHandler struct {
	hub                 *ws.Hub
	sessionService      *service.SessionService
	visitorService      *service.VisitorService
	satisfactionService *service.SatisfactionService
}

// NewExtraHandler 创建扩展处理器
func NewExtraHandler(hub *ws.Hub, sessionService *service.SessionService, visitorService *service.VisitorService, satisfactionService *service.SatisfactionService) *ExtraHandler {
	return &ExtraHandler{
		hub:                 hub,
		sessionService:      sessionService,
		visitorService:      visitorService,
		satisfactionService: satisfactionService,
	}
}

// Handle 处理消息
func (h *ExtraHandler) Handle(ctx context.Context, conn ws.Connection, data []byte) error {
	msgType := bizctx.MsgTypeFrom(ctx)

	switch protocol.ClientMessageType(msgType) {
	case protocol.ClientMsgSatisfactionRate:
		return h.handleSatisfactionRate(ctx, conn, data)
	case protocol.ClientMsgVisitorUpdate:
		return h.handleVisitorUpdate(ctx, conn, data)
	default:
		return ws.ErrUnknownMessageType
	}
}

// handleSatisfactionRate 处理满意度评价
func (h *ExtraHandler) handleSatisfactionRate(ctx context.Context, conn ws.Connection, data []byte) error {
	var msg protocol.SatisfactionRateMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	if err := msg.Validate(); err != nil {
		return err
	}

	logger.InfofCtx(ctx, "收到满意度评价, 连接ID: %s, 会话ID: %d, 评分: %d", conn.GetID(), msg.SessionID, msg.Rating)

	session, err := h.sessionService.GetSession(ctx, msg.SessionID)
	if err != nil {
		return err
	}

	rating := &model.SatisfactionRating{
		SessionID: msg.SessionID,
		VisitorID: conn.GetUserID(),
		AgentID:   session.AgentID,
		Rating:    int8(msg.Rating),
	}
	if err := h.satisfactionService.SubmitRating(ctx, rating); err != nil {
		logger.ErrorfCtx(ctx, "保存满意度评价失败, 会话ID: %d, 错误: %v", msg.SessionID, err)
		return err
	}

	resp := protocol.SatisfactionRateResponseMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgSatisfactionRate},
		SessionID:     msg.SessionID,
		Status:        "ok",
	}
	return sendJSON(ctx, conn, resp)
}

// handleVisitorUpdate 处理访客信息更新
func (h *ExtraHandler) handleVisitorUpdate(ctx context.Context, conn ws.Connection, data []byte) error {
	var msg protocol.VisitorUpdateMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	logger.InfofCtx(ctx, "更新访客信息, 连接ID: %s, 用户ID: %d", conn.GetID(), conn.GetUserID())

	if err := h.visitorService.UpdateVisitor(ctx, conn.GetUserID(), msg.Nickname, msg.Phone, msg.Avatar); err != nil {
		logger.ErrorfCtx(ctx, "更新访客信息失败, 用户ID: %d, 错误: %v", conn.GetUserID(), err)
		return err
	}

	if msg.Nickname != "" {
		conn.SetNickname(ctx, msg.Nickname)
	}

	resp := protocol.ServerMessage{Type: protocol.ServerMsgVisitorUpdateOK}
	if err := sendJSON(ctx, conn, resp); err != nil {
		return err
	}

	var sessionID int64
	activeSession, err := h.sessionService.GetActiveSessionByVisitor(ctx, conn.GetTenantNo(), conn.GetUserID())
	if err == nil && activeSession != nil {
		sessionID = activeSession.ID
	}

	updated := protocol.VisitorInfoUpdatedMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgVisitorInfoUpdated},
		SessionID:     sessionID,
		VisitorID:     conn.GetUserID(),
		Nickname:      msg.Nickname,
		Phone:         msg.Phone,
		Avatar:        msg.Avatar,
	}
	if data, err := mustMarshal(updated); err == nil {
		h.hub.BroadcastToTenant(ctx, conn.GetTenantNo(), data)
	}

	return nil
}
