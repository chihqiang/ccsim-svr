package service

import (
	"context"
	"sync"
	"time"

	"chihqiang/ccsim-svr/model"
	"chihqiang/ccsim-svr/protocol"
	"chihqiang/ccsim-svr/repo"

	"github.com/chihqiang/infra-go/logger"
)

// SessionService 会话服务
type SessionService struct {
	sessionStore *repo.SessionRepo
	visitorStore *repo.VisitorRepo
	agentStore   *repo.AgentRepo
	peerMu       sync.RWMutex
	peerCache    map[int64][2]int64 // sessionID -> [visitorID, agentID]（热路径缓存，避免DB查询）
}

// NewSessionService 创建会话服务
func NewSessionService(sessionStore *repo.SessionRepo, visitorStore *repo.VisitorRepo, agentStore *repo.AgentRepo) *SessionService {
	return &SessionService{
		sessionStore: sessionStore,
		visitorStore: visitorStore,
		agentStore:   agentStore,
		peerCache:    make(map[int64][2]int64, 128),
	}
}

// CreateOrGetSession 创建或获取会话（事务保护防止并发重复创建）
func (s *SessionService) CreateOrGetSession(ctx context.Context, tenantNo string, visitorID int64, visitorNickname string) (*model.Session, error) {
	if visitorID <= 0 {
		return nil, ErrUnauthorized
	}

	var resultSession *model.Session
	err := repo.TxDo(ctx, s.sessionStore.DB(), func(txCtx context.Context) error {
		// 先查找访客现有会话
		session, err := s.sessionStore.FindByVisitorID(txCtx, visitorID)
		if err == nil && session.Status != model.SessionStatusClosed {
			logger.InfofCtx(ctx, "访客已有活跃会话, 会话ID: %d, 访客ID: %d", session.ID, visitorID)
			resultSession = session
			s.cachePeer(session.ID, session.VisitorID, session.AgentID)
			return nil
		}
		// 创建新会话
		session = &model.Session{
			TenantNo:        tenantNo,
			VisitorID:       visitorID,
			VisitorNickname: visitorNickname,
			Status:          model.SessionStatusPending,
		}
		if err := s.sessionStore.Create(txCtx, session); err != nil {
			logger.ErrorfCtx(ctx, "创建会话失败, 租户: %s, 错误: %v", tenantNo, err)
			return err
		}
		logger.InfofCtx(ctx, "会话创建成功, ID: %d, 访客ID: %d", session.ID, visitorID)
		resultSession = session
		s.cachePeer(session.ID, session.VisitorID, 0)
		return nil
	})
	return resultSession, err
}

// GetSession 获取会话
func (s *SessionService) GetSession(ctx context.Context, id int64) (*model.Session, error) {
	return s.sessionStore.FindByID(ctx, id)
}

// AcceptSession 客服接受会话
func (s *SessionService) AcceptSession(ctx context.Context, sessionID, agentID int64) (*model.Session, error) {
	// 获取客服昵称用于写入会话
	agent, err := s.agentStore.FindByID(ctx, agentID)
	if err != nil {
		logger.ErrorfCtx(ctx, "获取客服信息失败, 客服ID: %d, 错误: %v", agentID, err)
		return nil, err
	}

	agentNickname := ""
	if agent != nil {
		agentNickname = agent.Nickname
	}

	if err := s.sessionStore.AssignAgent(ctx, sessionID, agentID, agentNickname); err != nil {
		if err == repo.ErrSessionAlreadyAssigned {
			return nil, ErrSessionConflict
		}
		logger.ErrorfCtx(ctx, "分配客服失败, 会话ID: %d, 错误: %v", sessionID, err)
		return nil, err
	}

	// 重置未读计数
	if err := s.sessionStore.ResetUnread(ctx, sessionID); err != nil {
		logger.ErrorfCtx(ctx, "重置未读计数失败, 会话ID: %d, 错误: %v", sessionID, err)
	}

	session, err := s.sessionStore.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	logger.InfofCtx(ctx, "会话已分配, 会话ID: %d, 客服ID: %d", sessionID, agentID)
	s.cachePeer(sessionID, session.VisitorID, agentID)
	return session, nil
}

// CloseSession 关闭会话
func (s *SessionService) CloseSession(ctx context.Context, sessionID int64, closeReason string) (*model.Session, error) {
	if err := s.sessionStore.Close(ctx, sessionID, closeReason); err != nil {
		logger.ErrorfCtx(ctx, "关闭会话失败, 会话ID: %d, 错误: %v", sessionID, err)
		return nil, err
	}

	s.peerMu.Lock()
	delete(s.peerCache, sessionID)
	s.peerMu.Unlock()

	session, err := s.sessionStore.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	logger.InfofCtx(ctx, "会话已关闭, 会话ID: %d", sessionID)
	return session, nil
}

// ListSessions 获取会话列表
func (s *SessionService) ListSessions(ctx context.Context, tenantNo string, status string, page, limit int) ([]*model.Session, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	return s.sessionStore.ListByTenant(ctx, tenantNo, status, page, limit)
}

// GetWaitingSessions 获取等待接入的会话
func (s *SessionService) GetWaitingSessions(ctx context.Context, tenantNo string) ([]*model.Session, error) {
	return s.sessionStore.FindWaitingByTenant(ctx, tenantNo)
}

// GetActiveSessionByVisitor 获取访客当前活跃会话
func (s *SessionService) GetActiveSessionByVisitor(ctx context.Context, tenantNo string, visitorID int64) (*model.Session, error) {
	session, err := s.sessionStore.FindByVisitorID(ctx, visitorID)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// UpdateLastMessage 更新最后消息
func (s *SessionService) UpdateLastMessage(ctx context.Context, sessionID, msgID int64) error {
	return s.sessionStore.UpdateLastMessage(ctx, sessionID, msgID)
}

// IncrementUnread 增加未读数
func (s *SessionService) IncrementUnread(ctx context.Context, sessionID int64) error {
	return s.sessionStore.IncrementUnread(ctx, sessionID)
}

// ResetUnread 重置未读数
func (s *SessionService) ResetUnread(ctx context.Context, sessionID int64) error {
	return s.sessionStore.ResetUnread(ctx, sessionID)
}

// GetSessionDuration 获取会话时长
func (s *SessionService) GetSessionDuration(ctx context.Context, sessionID int64) (time.Duration, error) {
	session, err := s.sessionStore.FindByID(ctx, sessionID)
	if err != nil {
		return 0, err
	}

	if session.ClosedAt != nil {
		return session.ClosedAt.Sub(session.CreatedAt), nil
	}
	return time.Since(session.CreatedAt), nil
}

// 错误定义
var (
	ErrUnauthorized    = &ServiceError{Code: "UNAUTHORIZED", Message: "未授权"}
	ErrSessionConflict = &ServiceError{Code: "SESSION_CONFLICT", Message: "会话已被其他客服接受"}
)

// cachePeer 缓存会话双方用户ID（热路径避免DB查询）
func (s *SessionService) cachePeer(sessionID, visitorID, agentID int64) {
	s.peerMu.Lock()
	s.peerCache[sessionID] = [2]int64{visitorID, agentID}
	s.peerMu.Unlock()
}

// GetPeerIDs 获取会话对方用户ID（先查缓存，miss时回源DB）
func (s *SessionService) GetPeerIDs(ctx context.Context, sessionID int64, selfID int64) (int64, protocol.Role) {
	s.peerMu.RLock()
	peer, ok := s.peerCache[sessionID]
	s.peerMu.RUnlock()

	if ok {
		visitorID, agentID := peer[0], peer[1]
		if selfID == visitorID && agentID > 0 {
			return agentID, protocol.RoleAgent
		}
		if selfID == agentID && visitorID > 0 {
			return visitorID, protocol.RoleVisitor
		}
	}
	// 缓存未命中，回源DB
	session, err := s.sessionStore.FindByID(ctx, sessionID)
	if err != nil {
		return 0, ""
	}
	s.cachePeer(sessionID, session.VisitorID, session.AgentID)
	if selfID == session.VisitorID && session.AgentID > 0 {
		return session.AgentID, protocol.RoleAgent
	}
	return session.VisitorID, protocol.RoleVisitor
}
