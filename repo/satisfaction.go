package repo

import (
	"context"

	"chihqiang/ccsim-svr/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SatisfactionRatingRepo 满意度评价存储实现
type SatisfactionRatingRepo struct {
	db *gorm.DB
}

// NewSatisfactionRatingRepo 创建满意度评价存储
func NewSatisfactionRatingRepo(db *gorm.DB) *SatisfactionRatingRepo {
	return &SatisfactionRatingRepo{db: db}
}

// FindBySessionID 根据会话ID查找
func (s *SatisfactionRatingRepo) FindBySessionID(ctx context.Context, sessionID int64) (*model.SatisfactionRating, error) {
	var rating model.SatisfactionRating
	err := UseTx(ctx, s.db).WithContext(ctx).Where("session_id = ?", sessionID).First(&rating).Error
	if err != nil {
		return nil, MapGormError(err)
	}
	return &rating, nil
}

// Create 创建或更新评价（同一会话仅保留一条）
func (s *SatisfactionRatingRepo) Create(ctx context.Context, rating *model.SatisfactionRating) error {
	return UseTx(ctx, s.db).WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"rating", "visitor_id", "agent_id"}),
	}).Create(rating).Error
}
