package repo

import (
	"context"

	"chihqiang/ccsim-svr/model"

	"gorm.io/gorm"
)

// AgentRepo 客服存储实现
type AgentRepo struct {
	db *gorm.DB
}

// NewAgentRepo 创建客服存储
func NewAgentRepo(db *gorm.DB) *AgentRepo {
	return &AgentRepo{db: db}
}

// FindByID 根据ID查找
func (s *AgentRepo) FindByID(ctx context.Context, id int64) (*model.Agent, error) {
	var agent model.Agent
	err := UseTx(ctx, s.db).WithContext(ctx).Where("id = ?", id).First(&agent).Error
	if err != nil {
		return nil, MapGormError(err)
	}
	return &agent, nil
}

// FindByAccount 根据账号查找
func (s *AgentRepo) FindByAccount(ctx context.Context, tenantNo, account string) (*model.Agent, error) {
	var agent model.Agent
	err := UseTx(ctx, s.db).WithContext(ctx).Where("tenant_no = ? AND account = ?", tenantNo, account).First(&agent).Error
	if err != nil {
		return nil, MapGormError(err)
	}
	return &agent, nil
}

// Create 创建客服
func (s *AgentRepo) Create(ctx context.Context, agent *model.Agent) error {
	return UseTx(ctx, s.db).WithContext(ctx).Create(agent).Error
}

// Update 更新客服
func (s *AgentRepo) Update(ctx context.Context, agent *model.Agent) error {
	return UseTx(ctx, s.db).WithContext(ctx).Save(agent).Error
}

// UpdateOnlineStatus 更新在线状态
func (s *AgentRepo) UpdateOnlineStatus(ctx context.Context, id int64, isOnline bool) error {
	var status model.AgentStatus
	if isOnline {
		status = model.AgentStatusOnline
	} else {
		status = model.AgentStatusOffline
	}
	return UseTx(ctx, s.db).WithContext(ctx).Model(&model.Agent{}).Where("id = ?", id).Update("is_online", status).Error
}

// CountOnlineByTenant 统计租户在线客服数
func (s *AgentRepo) CountOnlineByTenant(ctx context.Context, tenantNo string) (int64, error) {
	var count int64
	err := UseTx(ctx, s.db).WithContext(ctx).Model(&model.Agent{}).Where("tenant_no = ? AND is_online = ?", tenantNo, model.AgentStatusOnline).Count(&count).Error
	return count, err
}
