package repo

import (
	"context"
	"time"

	"chihqiang/ccsim-svr/model"

	"gorm.io/gorm"
)

// MessageRepo 消息存储实现
type MessageRepo struct {
	db *gorm.DB
}

// NewMessageRepo 创建消息存储
func NewMessageRepo(db *gorm.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

// FindByID 根据ID查找
func (s *MessageRepo) FindByID(ctx context.Context, id int64) (*model.Message, error) {
	var message model.Message
	err := UseTx(ctx, s.db).WithContext(ctx).Where("id = ?", id).First(&message).Error
	if err != nil {
		return nil, MapGormError(err)
	}
	return &message, nil
}

// Create 创建消息
func (s *MessageRepo) Create(ctx context.Context, message *model.Message) error {
	return UseTx(ctx, s.db).WithContext(ctx).Create(message).Error
}

// FindBySessionID 根据会话ID查找消息
func (s *MessageRepo) FindBySessionID(ctx context.Context, sessionID int64, beforeSeq, limit int) ([]*model.Message, error) {
	var messages []*model.Message
	query := UseTx(ctx, s.db).WithContext(ctx).Where("session_id = ?", sessionID)
	if beforeSeq > 0 {
		query = query.Where("seq_num < ?", beforeSeq)
	}
	if limit <= 0 {
		limit = 20
	}
	err := query.Order("seq_num DESC").Limit(limit).Find(&messages).Error
	return messages, err
}

// FindUnreadBySession 查找会话未读消息
func (s *MessageRepo) FindUnreadBySession(ctx context.Context, sessionID int64) ([]*model.Message, error) {
	var messages []*model.Message
	err := UseTx(ctx, s.db).WithContext(ctx).Where("session_id = ? AND is_read = ?", sessionID, model.ReadStatusUnread).Order("seq_num ASC").Find(&messages).Error
	return messages, err
}

// UpdateReadStatus 更新已读状态
func (s *MessageRepo) UpdateReadStatus(ctx context.Context, sessionID int64, msgID int64) error {
	return UseTx(ctx, s.db).WithContext(ctx).Model(&model.Message{}).Where("session_id = ? AND id <= ?", sessionID, msgID).Update("is_read", model.ReadStatusRead).Error
}

// BatchMarkAsRead 批量标记已读（单条SQL替代N次UPDATE）
func (s *MessageRepo) BatchMarkAsRead(ctx context.Context, sessionID int64, msgIDs []int64) error {
	if len(msgIDs) == 0 {
		return nil
	}
	return UseTx(ctx, s.db).WithContext(ctx).Model(&model.Message{}).
		Where("session_id = ? AND id IN ?", sessionID, msgIDs).
		Update("is_read", model.ReadStatusRead).Error
}

// GetNextSeqNum 获取下一个序列号（事务保护，防并发重复序号）
func (s *MessageRepo) GetNextSeqNum(ctx context.Context, sessionID int64) (int, error) {
	var maxSeq int
	db := UseTx(ctx, s.db).WithContext(ctx)
	err := db.Model(&model.Message{}).Where("session_id = ?", sessionID).Select("COALESCE(MAX(seq_num), 0)").Scan(&maxSeq).Error
	if err != nil {
		return 0, err
	}
	return maxSeq + 1, nil
}

// FindByIDs 根据ID列表批量查找消息
func (s *MessageRepo) FindByIDs(ctx context.Context, ids []int64) (map[int64]*model.Message, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var messages []*model.Message
	err := UseTx(ctx, s.db).WithContext(ctx).Where("id IN ?", ids).Find(&messages).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]*model.Message, len(messages))
	for _, m := range messages {
		result[m.ID] = m
	}
	return result, nil
}

// FindUnreadForVisitor 查找访客所有未读消息（跨活跃会话）
func (s *MessageRepo) FindUnreadForVisitor(ctx context.Context, visitorID int64) ([]*model.Message, error) {
	var messages []*model.Message
	err := UseTx(ctx, s.db).WithContext(ctx).
		Joins("JOIN ccsim_sessions s ON ccsim_messages.session_id = s.id").
		Where("s.visitor_id = ? AND s.status != ? AND ccsim_messages.is_read = ?", visitorID, model.SessionStatusClosed, model.ReadStatusUnread).
		Order("ccsim_messages.created_at ASC").
		Find(&messages).Error
	return messages, err
}

// FindUnreadForAgent 查找客服所有未读消息（跨活跃会话）
func (s *MessageRepo) FindUnreadForAgent(ctx context.Context, agentID int64) ([]*model.Message, error) {
	var messages []*model.Message
	err := UseTx(ctx, s.db).WithContext(ctx).
		Joins("JOIN ccsim_sessions s ON ccsim_messages.session_id = s.id").
		Where("s.agent_id = ? AND s.status != ? AND ccsim_messages.is_read = ?", agentID, model.SessionStatusClosed, model.ReadStatusUnread).
		Order("ccsim_messages.created_at ASC").
		Find(&messages).Error
	return messages, err
}

// CountBySession 统计会话消息总数（SQL COUNT）
func (s *MessageRepo) CountBySession(ctx context.Context, sessionID int64) (int64, error) {
	var count int64
	err := UseTx(ctx, s.db).WithContext(ctx).Model(&model.Message{}).Where("session_id = ?", sessionID).Count(&count).Error
	return count, err
}

// CountBySessionAndRole 按角色统计会话消息数
func (s *MessageRepo) CountBySessionAndRole(ctx context.Context, sessionID int64, role model.SenderRole) (int64, error) {
	var count int64
	err := UseTx(ctx, s.db).WithContext(ctx).Model(&model.Message{}).Where("session_id = ? AND sender_role = ?", sessionID, role).Count(&count).Error
	return count, err
}

// SearchByContent 按内容搜索消息（SQL LIKE）
func (s *MessageRepo) SearchByContent(ctx context.Context, sessionID int64, keyword string) ([]*model.Message, error) {
	var messages []*model.Message
	err := UseTx(ctx, s.db).WithContext(ctx).
		Where("session_id = ? AND content LIKE ?", sessionID, "%"+keyword+"%").
		Order("seq_num DESC").
		Limit(100).
		Find(&messages).Error
	return messages, err
}

// AvgResponseTime 计算平均客服响应时间
func (s *MessageRepo) AvgResponseTime(ctx context.Context, sessionID int64) (time.Duration, error) {
	var result struct {
		AvgDiff float64
	}
	err := UseTx(ctx, s.db).WithContext(ctx).
		Model(&model.Message{}).
		Select(`AVG(
			(SELECT MIN(m2.created_at) FROM ccsim_messages m2 
			 WHERE m2.session_id = ccsim_messages.session_id 
			 AND m2.sender_role = 'agent' 
			 AND m2.created_at > ccsim_messages.created_at)
			- ccsim_messages.created_at
		)`).
		Where("session_id = ? AND sender_role = ?", sessionID, model.SenderRoleVisitor).
		Scan(&result).Error
	if err != nil {
		return 0, err
	}
	return time.Duration(result.AvgDiff), nil
}
