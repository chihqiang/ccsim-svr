package repo

import (
	"context"

	"chihqiang/ccsim-svr/model"

	"gorm.io/gorm"
)

// VisitorRepo 访客存储实现
type VisitorRepo struct {
	db *gorm.DB
}

// NewVisitorRepo 创建访客存储
func NewVisitorRepo(db *gorm.DB) *VisitorRepo {
	return &VisitorRepo{db: db}
}

// FindByID 根据ID查找
func (s *VisitorRepo) FindByID(ctx context.Context, id int64) (*model.Visitor, error) {
	var visitor model.Visitor
	err := UseTx(ctx, s.db).WithContext(ctx).Where("id = ?", id).First(&visitor).Error
	if err != nil {
		return nil, MapGormError(err)
	}
	return &visitor, nil
}

// FindByExternalID 根据外部ID查找
func (s *VisitorRepo) FindByExternalID(ctx context.Context, tenantNo, externalID string) (*model.Visitor, error) {
	var visitor model.Visitor
	err := UseTx(ctx, s.db).WithContext(ctx).Where("tenant_no = ? AND external_id = ?", tenantNo, externalID).First(&visitor).Error
	if err != nil {
		return nil, MapGormError(err)
	}
	return &visitor, nil
}

// Create 创建访客
func (s *VisitorRepo) Create(ctx context.Context, visitor *model.Visitor) error {
	return UseTx(ctx, s.db).WithContext(ctx).Create(visitor).Error
}

// FindByIDs 根据ID列表批量查找访客
func (s *VisitorRepo) FindByIDs(ctx context.Context, ids []int64) (map[int64]*model.Visitor, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var visitors []*model.Visitor
	err := UseTx(ctx, s.db).WithContext(ctx).Where("id IN ?", ids).Find(&visitors).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]*model.Visitor, len(visitors))
	for _, v := range visitors {
		result[v.ID] = v
	}
	return result, nil
}

// Update 更新访客
func (s *VisitorRepo) Update(ctx context.Context, visitor *model.Visitor) error {
	return UseTx(ctx, s.db).WithContext(ctx).Save(visitor).Error
}
