package service

import (
	"context"

	"chihqiang/ccsim-svr/model"
	"chihqiang/ccsim-svr/repo"
	"chihqiang/ccsim-svr/ws"

	"github.com/chihqiang/infra-go/hash"
	"github.com/chihqiang/infra-go/logger"
)

// AgentService 客服服务
type AgentService struct {
	agentStore  *repo.AgentRepo
	distributor ws.Distributor // 集群模式下使用（nil = standalone）
	clusterMode bool
}

// NewAgentService 创建客服服务
func NewAgentService(agentStore *repo.AgentRepo) *AgentService {
	return &AgentService{agentStore: agentStore}
}

// SetDistributor 设置分布式分发器（集群模式）
func (s *AgentService) SetDistributor(d ws.Distributor) {
	s.distributor = d
	s.clusterMode = true
}

// Authenticate 认证客服
func (s *AgentService) Authenticate(ctx context.Context, tenantNo, account, password string) (*model.Agent, error) {
	agent, err := s.agentStore.FindByAccount(ctx, tenantNo, account)
	if err != nil {
		logger.ErrorfCtx(ctx, "客服账号不存在, 租户: %s, 账号: %s", tenantNo, account)
		return nil, err
	}

	if !hash.BcryptMatch(agent.Password, password) {
		logger.ErrorfCtx(ctx, "客服密码错误, 租户: %s, 账号: %s", tenantNo, account)
		return nil, ErrInvalidPassword
	}

	logger.InfofCtx(ctx, "客服认证成功, ID: %d, 账号: %s", agent.ID, account)
	return agent, nil
}

// GetAgent 获取客服信息
func (s *AgentService) GetAgent(ctx context.Context, id int64) (*model.Agent, error) {
	return s.agentStore.FindByID(ctx, id)
}

// SetOnline 设置客服在线状态
func (s *AgentService) SetOnline(ctx context.Context, id int64, isOnline bool, tenantNo string) error {
	if s.clusterMode && s.distributor != nil {
		// 集群模式：写 Redis Set，不写 DB
		if isOnline {
			if err := s.distributor.RegisterAgent(ctx, tenantNo, id); err != nil {
				logger.ErrorfCtx(ctx, "注册客服到Redis失败, ID: %d, 错误: %v", id, err)
				return err
			}
		} else {
			if err := s.distributor.UnregisterAgent(ctx, tenantNo, id); err != nil {
				logger.ErrorfCtx(ctx, "从Redis移除客服失败, ID: %d, 错误: %v", id, err)
				return err
			}
		}
	} else {
		// 单机模式：写 DB
		if err := s.agentStore.UpdateOnlineStatus(ctx, id, isOnline); err != nil {
			logger.ErrorfCtx(ctx, "设置客服在线状态失败, ID: %d, 错误: %v", id, err)
			return err
		}
	}

	status := "下线"
	if isOnline {
		status = "上线"
	}
	logger.InfofCtx(ctx, "客服已%s, ID: %d", status, id)
	return nil
}

// GetOnlineCount 获取在线客服数
func (s *AgentService) GetOnlineCount(ctx context.Context, tenantNo string) (int64, error) {
	if s.clusterMode && s.distributor != nil {
		// 集群模式：从 Redis Set 统计
		return s.distributor.CountOnlineAgents(ctx, tenantNo)
	}
	// 单机模式：从 DB 统计
	return s.agentStore.CountOnlineByTenant(ctx, tenantNo)
}

// 错误定义
var (
	ErrInvalidPassword = &ServiceError{Code: "INVALID_PASSWORD", Message: "密码错误"}
)

// ServiceError 服务错误
type ServiceError struct {
	Code    string
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}
