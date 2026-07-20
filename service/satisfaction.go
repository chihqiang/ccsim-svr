package service

import (
	"context"

	"chihqiang/ccsim-svr/model"
	"chihqiang/ccsim-svr/repo"

	"github.com/chihqiang/infra-go/logger"
)

// SatisfactionService 满意度评价服务
type SatisfactionService struct {
	satisfactionStore *repo.SatisfactionRatingRepo
}

// NewSatisfactionService 创建满意度评价服务
func NewSatisfactionService(satisfactionStore *repo.SatisfactionRatingRepo) *SatisfactionService {
	return &SatisfactionService{satisfactionStore: satisfactionStore}
}

// SubmitRating 提交满意度评价
func (s *SatisfactionService) SubmitRating(ctx context.Context, rating *model.SatisfactionRating) error {
	if err := s.satisfactionStore.Create(ctx, rating); err != nil {
		logger.ErrorfCtx(ctx, "保存满意度评价失败, 会话ID: %d, 错误: %v", rating.SessionID, err)
		return err
	}
	logger.InfofCtx(ctx, "满意度评价已保存, 会话ID: %d, 评分: %d", rating.SessionID, rating.Rating)
	return nil
}

// GetBySessionID 获取会话的满意度评价
func (s *SatisfactionService) GetBySessionID(ctx context.Context, sessionID int64) (*model.SatisfactionRating, error) {
	return s.satisfactionStore.FindBySessionID(ctx, sessionID)
}
