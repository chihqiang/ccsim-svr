package service

import (
	"context"
	"time"

	"chihqiang/ccsim-svr/model"
	"chihqiang/ccsim-svr/repo"

	"github.com/chihqiang/infra-go/logger"
)

// MessageService 消息服务
type MessageService struct {
	messageStore *repo.MessageRepo
	sessionStore *repo.SessionRepo
}

// NewMessageService 创建消息服务
func NewMessageService(messageStore *repo.MessageRepo, sessionStore *repo.SessionRepo) *MessageService {
	return &MessageService{
		messageStore: messageStore,
		sessionStore: sessionStore,
	}
}

// SendMessage 发送消息（事务保护：创建消息 + 更新会话最后消息 + 递增序号原子化）
func (s *MessageService) SendMessage(ctx context.Context, sessionID int64, senderRole model.SenderRole, senderID int64, nickname, content, msgType, tempID string) (*model.Message, error) {
	var message *model.Message

	err := repo.TxDo(ctx, s.sessionStore.DB(), func(txCtx context.Context) error {
		seqNum, err := s.messageStore.GetNextSeqNum(txCtx, sessionID)
		if err != nil {
			return err
		}

		message = &model.Message{
			SessionID:  sessionID,
			SenderRole: senderRole,
			SenderID:   senderID,
			Nickname:   nickname,
			Content:    content,
			MsgType:    model.MsgType(msgType),
			SeqNum:     seqNum,
			IsRead:     model.ReadStatusUnread,
		}

		if err := s.messageStore.Create(txCtx, message); err != nil {
			return err
		}

		if err := s.sessionStore.UpdateLastMessage(txCtx, sessionID, message.ID); err != nil {
			logger.ErrorfCtx(txCtx, "更新会话最后消息失败, 会话ID: %d, 错误: %v", sessionID, err)
		}

		return nil
	})

	if err != nil {
		logger.ErrorfCtx(ctx, "保存消息失败, 会话ID: %d, 错误: %v", sessionID, err)
		return nil, err
	}

	logger.InfofCtx(ctx, "消息发送成功, ID: %d, 会话ID: %d, 序列号: %d", message.ID, sessionID, message.SeqNum)
	return message, nil
}

// SendSystemMessage 发送系统消息
func (s *MessageService) SendSystemMessage(ctx context.Context, sessionID int64, content string) (*model.Message, error) {
	return s.SendMessage(ctx, sessionID, model.SenderRoleSystem, 0, "系统", content, "text", "")
}

// GetHistoryMessages 获取历史消息
func (s *MessageService) GetHistoryMessages(ctx context.Context, sessionID int64, beforeSeq, limit int) ([]*model.Message, error) {
	return s.messageStore.FindBySessionID(ctx, sessionID, beforeSeq, limit)
}

// GetMessagesByIDs 根据ID列表批量获取消息
func (s *MessageService) GetMessagesByIDs(ctx context.Context, ids []int64) (map[int64]*model.Message, error) {
	return s.messageStore.FindByIDs(ctx, ids)
}

// GetUnreadMessages 获取未读消息
func (s *MessageService) GetUnreadMessages(ctx context.Context, sessionID int64) ([]*model.Message, error) {
	return s.messageStore.FindUnreadBySession(ctx, sessionID)
}

// MarkAsRead 标记已读
func (s *MessageService) MarkAsRead(ctx context.Context, sessionID, msgID int64) error {
	return s.messageStore.UpdateReadStatus(ctx, sessionID, msgID)
}

// GetMessage 获取消息
func (s *MessageService) GetMessage(ctx context.Context, id int64) (*model.Message, error) {
	return s.messageStore.FindByID(ctx, id)
}

// GetMessageCount 获取消息数量（使用 SQL COUNT）
func (s *MessageService) GetMessageCount(ctx context.Context, sessionID int64) (int, error) {
	count, err := s.messageStore.CountBySession(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// GetLastMessage 获取最后一条消息
func (s *MessageService) GetLastMessage(ctx context.Context, sessionID int64) (*model.Message, error) {
	messages, err := s.messageStore.FindBySessionID(ctx, sessionID, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	return messages[0], nil
}

// BatchMarkAsRead 批量标记已读
func (s *MessageService) BatchMarkAsRead(ctx context.Context, sessionID int64, msgIDs []int64) error {
	return s.messageStore.BatchMarkAsRead(ctx, sessionID, msgIDs)
}

// SearchMessages 搜索消息（使用 SQL LIKE）
func (s *MessageService) SearchMessages(ctx context.Context, sessionID int64, keyword string) ([]*model.Message, error) {
	return s.messageStore.SearchByContent(ctx, sessionID, keyword)
}

// GetOfflineMessagesForVisitor 获取访客离线未读消息
func (s *MessageService) GetOfflineMessagesForVisitor(ctx context.Context, visitorID int64) ([]*model.Message, error) {
	return s.messageStore.FindUnreadForVisitor(ctx, visitorID)
}

// GetOfflineMessagesForAgent 获取客服离线未读消息
func (s *MessageService) GetOfflineMessagesForAgent(ctx context.Context, agentID int64) ([]*model.Message, error) {
	return s.messageStore.FindUnreadForAgent(ctx, agentID)
}

// GetMessageStats 获取消息统计（使用 SQL 聚合）
func (s *MessageService) GetMessageStats(ctx context.Context, sessionID int64) (*MessageStats, error) {
	totalCount, err := s.messageStore.CountBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	visitorCount, err := s.messageStore.CountBySessionAndRole(ctx, sessionID, model.SenderRoleVisitor)
	if err != nil {
		return nil, err
	}

	agentCount, err := s.messageStore.CountBySessionAndRole(ctx, sessionID, model.SenderRoleAgent)
	if err != nil {
		return nil, err
	}

	systemCount, err := s.messageStore.CountBySessionAndRole(ctx, sessionID, model.SenderRoleSystem)
	if err != nil {
		return nil, err
	}

	avgResponseTime, err := s.messageStore.AvgResponseTime(ctx, sessionID)
	if err != nil {
		logger.WarnfCtx(ctx, "计算平均响应时间失败, 会话ID: %d, 错误: %v", sessionID, err)
		avgResponseTime = 0
	}

	stats := &MessageStats{
		TotalCount:      int(totalCount),
		VisitorCount:    int(visitorCount),
		AgentCount:      int(agentCount),
		SystemCount:     int(systemCount),
		AvgResponseTime: avgResponseTime,
	}

	return stats, nil
}

// MessageStats 消息统计
type MessageStats struct {
	TotalCount      int           `json:"totalCount"`
	VisitorCount    int           `json:"visitorCount"`
	AgentCount      int           `json:"agentCount"`
	SystemCount     int           `json:"systemCount"`
	AvgResponseTime time.Duration `json:"avgResponseTime"`
}
