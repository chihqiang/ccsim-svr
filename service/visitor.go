package service

import (
	"context"

	"chihqiang/ccsim-svr/model"
	"chihqiang/ccsim-svr/repo"

	"github.com/chihqiang/infra-go/logger"
)

// VisitorService 访客服务
type VisitorService struct {
	visitorStore *repo.VisitorRepo
}

// NewVisitorService 创建访客服务
func NewVisitorService(visitorStore *repo.VisitorRepo) *VisitorService {
	return &VisitorService{visitorStore: visitorStore}
}

// Authenticate 认证访客
func (s *VisitorService) Authenticate(ctx context.Context, tenantNo, externalID, nickname, phone, platform string) (*model.Visitor, error) {
	// 查找现有访客
	visitor, err := s.visitorStore.FindByExternalID(ctx, tenantNo, externalID)
	if err == nil {
		logger.InfofCtx(ctx, "访客已存在, ID: %d, 租户: %s", visitor.ID, tenantNo)
		return visitor, nil
	}

	// 创建新访客
	visitor = &model.Visitor{
		TenantNo:   tenantNo,
		ExternalID: externalID,
		Nickname:   nickname,
		Phone:      phone,
		Platform:   platform,
	}
	if err := s.visitorStore.Create(ctx, visitor); err != nil {
		logger.ErrorfCtx(ctx, "创建访客失败, 租户: %s, 错误: %v", tenantNo, err)
		return nil, err
	}

	logger.InfofCtx(ctx, "访客创建成功, ID: %d, 租户: %s", visitor.ID, tenantNo)
	return visitor, nil
}

// GetVisitor 获取访客信息
func (s *VisitorService) GetVisitor(ctx context.Context, id int64) (*model.Visitor, error) {
	return s.visitorStore.FindByID(ctx, id)
}

// GetVisitorsByIDs 根据ID列表批量获取访客信息
func (s *VisitorService) GetVisitorsByIDs(ctx context.Context, ids []int64) (map[int64]*model.Visitor, error) {
	return s.visitorStore.FindByIDs(ctx, ids)
}

// UpdateVisitor 更新访客信息
func (s *VisitorService) UpdateVisitor(ctx context.Context, id int64, nickname, phone, avatar string) error {
	visitor, err := s.visitorStore.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if nickname != "" {
		visitor.Nickname = nickname
	}
	if phone != "" {
		visitor.Phone = phone
	}
	if avatar != "" {
		visitor.Avatar = avatar
	}

	return s.visitorStore.Update(ctx, visitor)
}
