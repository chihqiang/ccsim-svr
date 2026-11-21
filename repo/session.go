package repo

import (
	"context"
	"errors"
	"time"

	"chihqiang/ccsim-svr/model"

	"gorm.io/gorm"
)

// SessionRepo 会话存储实现
type SessionRepo struct {
	db *gorm.DB
}

// NewSessionRepo 创建会话存储
func NewSessionRepo(db *gorm.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

// DB 返回底层数据库连接（供事务使用）
func (s *SessionRepo) DB() *gorm.DB {
	return s.db
}

// Transaction 在事务中执行函数
func (s *SessionRepo) Transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return TxDo(ctx, s.db, fn)
}

// FindByID 根据ID查找
func (s *SessionRepo) FindByID(ctx context.Context, id int64) (*model.Session, error) {
	var session model.Session
	err := UseTx(ctx, s.db).WithContext(ctx).Where("id = ?", id).First(&session).Error
	if err != nil {
		return nil, MapGormError(err)
	}
	return &session, nil
}

// Create 创建会话
func (s *SessionRepo) Create(ctx context.Context, session *model.Session) error {
	return UseTx(ctx, s.db).WithContext(ctx).Create(session).Error
}

// Update 更新会话
func (s *SessionRepo) Update(ctx context.Context, session *model.Session) error {
	return UseTx(ctx, s.db).WithContext(ctx).Save(session).Error
}

// UpdateStatus 更新会话状态
func (s *SessionRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return UseTx(ctx, s.db).WithContext(ctx).Model(&model.Session{}).Where("id = ?", id).Update("status", status).Error
}

var ErrSessionAlreadyAssigned = errors.New("会话已被其他客服接受")

// AssignAgent 分配客服（条件更新：仅 pending 状态可接受，防止分布式双接受）
func (s *SessionRepo) AssignAgent(ctx context.Context, id, agentID int64, agentNickname string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"agent_id":       agentID,
		"agent_nickname": agentNickname,
		"status":         model.SessionStatusActive,
		"assigned_at":    now,
	}
	result := UseTx(ctx, s.db).WithContext(ctx).Model(&model.Session{}).
		Where("id = ? AND status = ?", id, model.SessionStatusPending).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSessionAlreadyAssigned
	}
	return nil
}

// Close 关闭会话
func (s *SessionRepo) Close(ctx context.Context, id int64, reason string) error {
	now := time.Now()
	return UseTx(ctx, s.db).WithContext(ctx).Model(&model.Session{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       model.SessionStatusClosed,
		"close_reason": reason,
		"closed_at":    now,
	}).Error
}

// FindByVisitorID 根据访客ID查找会话
func (s *SessionRepo) FindByVisitorID(ctx context.Context, visitorID int64) (*model.Session, error) {
	var session model.Session
	err := UseTx(ctx, s.db).WithContext(ctx).Where("visitor_id = ? AND status != ?", visitorID, model.SessionStatusClosed).Order("created_at DESC").First(&session).Error
	if err != nil {
		return nil, MapGormError(err)
	}
	return &session, nil
}

// FindByAgentID 根据客服ID查找会话
func (s *SessionRepo) FindByAgentID(ctx context.Context, agentID int64, status string) ([]*model.Session, error) {
	var sessions []*model.Session
	query := UseTx(ctx, s.db).WithContext(ctx).Where("agent_id = ?", agentID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}

// FindWaitingByTenant 查找租户等待接入的会话
func (s *SessionRepo) FindWaitingByTenant(ctx context.Context, tenantNo string) ([]*model.Session, error) {
	var sessions []*model.Session
	err := UseTx(ctx, s.db).WithContext(ctx).Where("tenant_no = ? AND status = ?", tenantNo, model.SessionStatusPending).Order("created_at ASC").Find(&sessions).Error
	return sessions, err
}

// ListByTenant 分页查询租户会话
func (s *SessionRepo) ListByTenant(ctx context.Context, tenantNo string, status string, page, limit int) ([]*model.Session, int64, error) {
	var sessions []*model.Session
	var total int64

	query := UseTx(ctx, s.db).WithContext(ctx).Where("tenant_no = ?", tenantNo)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Model(&model.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&sessions).Error
	return sessions, total, err
}

// UpdateLastMessage 更新最后消息
func (s *SessionRepo) UpdateLastMessage(ctx context.Context, id, msgID int64) error {
	now := time.Now()
	return UseTx(ctx, s.db).WithContext(ctx).Model(&model.Session{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_msg_id":   msgID,
		"last_msg_time": now,
	}).Error
}

// IncrementUnread 增加未读数
func (s *SessionRepo) IncrementUnread(ctx context.Context, id int64) error {
	return UseTx(ctx, s.db).WithContext(ctx).Model(&model.Session{}).Where("id = ?", id).UpdateColumn("unread_count", gorm.Expr("unread_count + 1")).Error
}

// ResetUnread 重置未读数
func (s *SessionRepo) ResetUnread(ctx context.Context, id int64) error {
	return UseTx(ctx, s.db).WithContext(ctx).Model(&model.Session{}).Where("id = ?", id).Update("unread_count", 0).Error
}
