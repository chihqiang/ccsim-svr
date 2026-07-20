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

// AuthHandler 认证处理器
type AuthHandler struct {
	hub            *ws.Hub
	visitorService *service.VisitorService
	agentService   *service.AgentService
	messageService *service.MessageService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(hub *ws.Hub, visitorService *service.VisitorService, agentService *service.AgentService, messageService *service.MessageService) *AuthHandler {
	return &AuthHandler{
		hub:            hub,
		visitorService: visitorService,
		agentService:   agentService,
		messageService: messageService,
	}
}

// Handle 处理认证消息
func (h *AuthHandler) Handle(ctx context.Context, conn ws.Connection, data []byte) error {
	var msg protocol.AuthMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		logger.ErrorfCtx(ctx, "解析认证消息失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	if err := msg.Validate(); err != nil {
		logger.WarnfCtx(ctx, "认证消息校验失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	logger.InfofCtx(ctx, "收到认证消息, 连接ID: %s, 租户: %s, 角色: %s", conn.GetID(), msg.TenantNo, msg.Role)

	conn.SetTenantNo(ctx, msg.TenantNo)
	conn.SetRole(ctx, protocol.Role(msg.Role))

	var authErr error
	switch protocol.Role(msg.Role) {
	case protocol.RoleVisitor:
		authErr = h.handleVisitorAuth(ctx, conn, &msg)
	case protocol.RoleAgent:
		authErr = h.handleAgentAuth(ctx, conn, &msg)
	default:
		logger.ErrorfCtx(ctx, "未知角色, 连接ID: %s, 角色: %s", conn.GetID(), msg.Role)
		return ws.ErrInvalidRole
	}

	// 认证完成后清除密码字段，防止后续日志泄露
	msg.AgentPassword = ""
	return authErr
}

// handleVisitorAuth 处理访客认证
func (h *AuthHandler) handleVisitorAuth(ctx context.Context, conn ws.Connection, msg *protocol.AuthMessage) error {
	visitor, err := h.visitorService.Authenticate(ctx, msg.TenantNo, msg.ExternalVisitorID, msg.Nickname, msg.Phone, msg.Platform)
	if err != nil {
		logger.ErrorfCtx(ctx, "访客认证失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	conn.SetUserID(ctx, visitor.ID)
	conn.SetNickname(ctx, visitor.Nickname)
	h.hub.BindUser(ctx, visitor.ID, protocol.RoleVisitor, conn)

	resp := protocol.AuthOKMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgAuthOK},
		ConnID:        conn.GetID(),
		VisitorID:     visitor.ID,
	}
	if err := sendJSON(ctx, conn, resp); err != nil {
		return err
	}

	h.sendOfflinePush(ctx, conn, protocol.RoleVisitor, visitor.ID)
	return nil
}

// handleAgentAuth 处理客服认证
func (h *AuthHandler) handleAgentAuth(ctx context.Context, conn ws.Connection, msg *protocol.AuthMessage) error {
	agent, err := h.agentService.Authenticate(ctx, msg.TenantNo, msg.AgentAccount, msg.AgentPassword)
	if err != nil {
		logger.ErrorfCtx(ctx, "客服认证失败, 连接ID: %s, 错误: %v", conn.GetID(), err)
		return err
	}

	conn.SetUserID(ctx, agent.ID)
	conn.SetNickname(ctx, agent.Nickname)
	h.hub.BindUser(ctx, agent.ID, protocol.RoleAgent, conn)

	resp := protocol.AuthOKMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgAuthOK},
		ConnID:        conn.GetID(),
		AgentID:       agent.ID,
	}
	if err := sendJSON(ctx, conn, resp); err != nil {
		return err
	}

	h.sendOfflinePush(ctx, conn, protocol.RoleAgent, agent.ID)
	return nil
}

// sendOfflinePush 推送离线未读消息（分页推送，避免单条消息过大）
func (h *AuthHandler) sendOfflinePush(ctx context.Context, conn ws.Connection, role protocol.Role, userID int64) {
	const offlinePageSize = 50

	var dbMessages []*model.Message

	switch role {
	case protocol.RoleVisitor:
		msgs, err := h.messageService.GetOfflineMessagesForVisitor(ctx, userID)
		if err != nil {
			logger.ErrorfCtx(ctx, "获取访客离线消息失败, 用户ID: %d, 错误: %v", userID, err)
			return
		}
		dbMessages = msgs
	case protocol.RoleAgent:
		msgs, err := h.messageService.GetOfflineMessagesForAgent(ctx, userID)
		if err != nil {
			logger.ErrorfCtx(ctx, "获取客服离线消息失败, 用户ID: %d, 错误: %v", userID, err)
			return
		}
		dbMessages = msgs
	}

	if len(dbMessages) == 0 {
		return
	}

	for i := 0; i < len(dbMessages); i += offlinePageSize {
		end := i + offlinePageSize
		if end > len(dbMessages) {
			end = len(dbMessages)
		}
		batch := dbMessages[i:end]

		messages := make([]protocol.ChatPushMessage, 0, len(batch))
		for _, m := range batch {
			messages = append(messages, toChatPushMessage(m))
		}

		push := protocol.OfflinePushMessage{
			ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgOfflinePush},
			Messages:      messages,
		}
		if err := sendJSON(ctx, conn, push); err != nil {
			logger.ErrorfCtx(ctx, "推送离线消息失败, 用户ID: %d, 错误: %v", userID, err)
			return
		}
	}

	logger.InfofCtx(ctx, "推送离线消息成功, 用户ID: %d, 数量: %d", userID, len(dbMessages))
}
