package serv

import (
	"context"

	"chihqiang/ccsim-svr/app"
	"chihqiang/ccsim-svr/ws"

	"github.com/chihqiang/infra-go/logger"
)

// Cluster 集群服务（分布式模式）
type Cluster struct {
	ctx context.Context
	app *app.App
}

// NewCluster 创建集群服务
func NewCluster(ctx context.Context, app *app.App) *Cluster {
	return &Cluster{ctx: ctx, app: app}
}

// Start 初始化分布式分发器 + 启动 Redis 订阅
func (c *Cluster) Start() {
	if !c.app.Cfg.Redis.Enabled {
		logger.InfofCtx(c.ctx, "单机模式运行, 节点ID: %s", c.app.NodeID)
		return
	}
	distributor, err := ws.NewRedisDistributor(c.app.NodeID, c.app.Cfg.Redis)
	if err != nil {
		logger.FatalfCtx(c.ctx, "初始化Redis分发器失败: %v", err)
	}

	c.app.Hub.SetDistributor(c.ctx, distributor, c.app.NodeID)
	c.app.Hub.SetClusterMode(true)
	c.app.AgentService.SetDistributor(distributor)
	c.app.Hub.GetDistributor(c.ctx).StartSubscribers(c.ctx, c.app.NodeID, c.app.Hub.HandleRemoteNodeMessage, c.app.Hub.HandleRemoteTenantMessage)

	logger.InfofCtx(c.ctx, "分布式模式已启用, 节点ID: %s", c.app.NodeID)
}

// Stop 停止集群服务
func (c *Cluster) Stop() {
	logger.InfoCtx(c.ctx, "集群服务关闭中...")
}
