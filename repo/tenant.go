package repo

import (
	"context"

	"chihqiang/ccsim-svr/model"

	"gorm.io/gorm"
)

// TenantRepo 租户存储实现
type TenantRepo struct {
	db *gorm.DB
}

// NewTenantRepo 创建租户存储
func NewTenantRepo(db *gorm.DB) *TenantRepo {
	return &TenantRepo{db: db}
}

// FindByTenantNo 根据租户编号查找
func (s *TenantRepo) FindByTenantNo(ctx context.Context, tenantNo string) (*model.Tenant, error) {
	var tenant model.Tenant
	err := s.db.WithContext(ctx).Where("tenant_no = ?", tenantNo).First(&tenant).Error
	if err != nil {
		return nil, MapGormError(err)
	}
	return &tenant, nil
}

// Create 创建租户
func (s *TenantRepo) Create(ctx context.Context, tenant *model.Tenant) error {
	return s.db.WithContext(ctx).Create(tenant).Error
}

// Update 更新租户
func (s *TenantRepo) Update(ctx context.Context, tenant *model.Tenant) error {
	return s.db.WithContext(ctx).Save(tenant).Error
}
