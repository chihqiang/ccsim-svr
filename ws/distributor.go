package ws

import (
	"context"
	"fmt"
	"time"

	"chihqiang/ccsim-svr/config"

	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/redisx"
	"github.com/redis/go-redis/v9"
)

// Lua 脚本：仅当 value == nodeID 时才删除（C2 归属检查）
var unregisterUserScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

// Lua 脚本：仅当 key 不存在或 value == nodeID 时才 SET（C3 归属检查）
var registerUserScript = redis.NewScript(`
local current = redis.call("GET", KEYS[1])
if current == false or current == ARGV[1] then
	return redis.call("SET", KEYS[1], ARGV[1], "EX", ARGV[2])
end
return 0
`)

// Lua 脚本：客服上线（C1 归属检查 + Set）
var registerAgentScript = redis.NewScript(`
local key = KEYS[1]
local member = ARGV[1]
local ttl = ARGV[2]
redis.call("SADD", key, member)
redis.call("EXPIRE", key, ttl)
return 1
`)

// Lua 脚本：客服下线（C1 仅删除自己的 member）
var unregisterAgentScript = redis.NewScript(`
local key = KEYS[1]
local member = ARGV[1]
return redis.call("SREM", key, member)
`)

// Distributor 跨节点消息分发器接口
type Distributor interface {
	// PublishToNode 向指定节点发送消息
	PublishToNode(ctx context.Context, nodeID string, data []byte) error
	// PublishToTenant 向租户所有节点广播消息
	PublishToTenant(ctx context.Context, tenantNo string, data []byte) error
	// RegisterUser 注册用户到当前节点（归属检查：仅当无主或属于本节点时注册）
	RegisterUser(ctx context.Context, userID int64) error
	// UnregisterUser 注销用户（归属检查：仅当属于本节点时删除）
	UnregisterUser(ctx context.Context, userID int64) error
	// RefreshUser 续期用户注册
	RefreshUser(ctx context.Context, userID int64) error
	// RefreshUsers 批量续期用户注册
	RefreshUsers(ctx context.Context, userIDs []int64) error
	// GetUserNode 获取用户所在节点
	GetUserNode(ctx context.Context, userID int64) (string, error)
	// RegisterAgent 客服上线（加入 Redis Set）
	RegisterAgent(ctx context.Context, tenantNo string, agentID int64) error
	// UnregisterAgent 客服下线（移除 Redis Set member）
	UnregisterAgent(ctx context.Context, tenantNo string, agentID int64) error
	// CountOnlineAgents 统计租户在线客服数
	CountOnlineAgents(ctx context.Context, tenantNo string) (int64, error)
	// StartSubscribers 启动订阅监听
	StartSubscribers(ctx context.Context, nodeID string, onNodeMessage func(ctx context.Context, data []byte), onTenantMessage func(ctx context.Context, data []byte))
	// Close 关闭连接
	Close() error
}

// RedisDistributor 基于Redis的分布式消息分发器
type RedisDistributor struct {
	nodeID          string
	client          *redisx.Client
	keyPrefix       string
	registryUserTTL time.Duration
}

// NewRedisDistributor 创建Redis分发器
func NewRedisDistributor(nodeID string, cfg config.RedisConfig) (*RedisDistributor, error) {
	rdb, err := redisx.New(redisx.Config{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err != nil {
		return nil, err
	}

	d := &RedisDistributor{
		nodeID:          nodeID,
		client:          rdb,
		keyPrefix:       "ccsim:",
		registryUserTTL: time.Duration(cfg.RegistryTTL) * time.Second,
	}

	// 验证连接
	if err := rdb.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("redis连接失败: %w", err)
	}
	return d, nil
}

func (d *RedisDistributor) registryUserKey(userID int64) string {
	return fmt.Sprintf("%sregistry:user:%d", d.keyPrefix, userID)
}

func (d *RedisDistributor) onlineAgentsKey(tenantNo string) string {
	return fmt.Sprintf("%sonline:%s", d.keyPrefix, tenantNo)
}

func (d *RedisDistributor) nodeChannel(nodeID string) string {
	return fmt.Sprintf("%snode:%s", d.keyPrefix, nodeID)
}

func (d *RedisDistributor) tenantChannel(tenantNo string) string {
	return fmt.Sprintf("%stenant:%s", d.keyPrefix, tenantNo)
}

// PublishToNode 向指定节点发送消息
func (d *RedisDistributor) PublishToNode(ctx context.Context, targetNodeID string, data []byte) error {
	if targetNodeID == d.nodeID {
		return nil // 不发给自己
	}
	return d.client.Client().Publish(ctx, d.nodeChannel(targetNodeID), data).Err()
}

// PublishToTenant 向租户所有节点广播消息
func (d *RedisDistributor) PublishToTenant(ctx context.Context, tenantNo string, data []byte) error {
	return d.client.Client().Publish(ctx, d.tenantChannel(tenantNo), data).Err()
}

// RegisterUser 注册用户到当前节点（C3 归属检查：仅当无主或属于本节点时注册）
func (d *RedisDistributor) RegisterUser(ctx context.Context, userID int64) error {
	_, err := registerUserScript.Run(ctx, d.client.Client(), []string{d.registryUserKey(userID)}, d.nodeID, int(d.registryUserTTL.Seconds())).Result()
	return err
}

// UnregisterUser 注销用户（C2 归属检查：仅当属于本节点时删除）
func (d *RedisDistributor) UnregisterUser(ctx context.Context, userID int64) error {
	_, err := unregisterUserScript.Run(ctx, d.client.Client(), []string{d.registryUserKey(userID)}, d.nodeID).Result()
	return err
}

// RefreshUser 续期用户注册（直接EXPIRE，避免GET+EXPIRE两次RTT）
func (d *RedisDistributor) RefreshUser(ctx context.Context, userID int64) error {
	_, err := d.client.Expire(ctx, d.registryUserKey(userID), d.registryUserTTL)
	return err
}

// RefreshUsers 批量续期用户注册（Pipeline单次RTT）
func (d *RedisDistributor) RefreshUsers(ctx context.Context, userIDs []int64) error {
	if len(userIDs) == 0 {
		return nil
	}
	pipe := d.client.Client().Pipeline()
	for _, uid := range userIDs {
		pipe.Expire(ctx, d.registryUserKey(uid), d.registryUserTTL)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// GetUserNode 获取用户所在节点
func (d *RedisDistributor) GetUserNode(ctx context.Context, userID int64) (string, error) {
	nodeID, err := d.client.Get(ctx, d.registryUserKey(userID))
	if err != nil {
		return "", err
	}
	return nodeID, nil
}

// RegisterAgent 客服上线（C1 加入 Redis Set + Lua 归属检查）
func (d *RedisDistributor) RegisterAgent(ctx context.Context, tenantNo string, agentID int64) error {
	member := fmt.Sprintf("%s:%d", d.nodeID, agentID)
	_, err := registerAgentScript.Run(ctx, d.client.Client(), []string{d.onlineAgentsKey(tenantNo)}, member, int(d.registryUserTTL.Seconds())).Result()
	return err
}

// UnregisterAgent 客服下线（C1 移除 Redis Set member，仅删自己的）
func (d *RedisDistributor) UnregisterAgent(ctx context.Context, tenantNo string, agentID int64) error {
	member := fmt.Sprintf("%s:%d", d.nodeID, agentID)
	_, err := unregisterAgentScript.Run(ctx, d.client.Client(), []string{d.onlineAgentsKey(tenantNo)}, member).Result()
	return err
}

// CountOnlineAgents 统计租户在线客服数（SCARD 单次 RTT）
func (d *RedisDistributor) CountOnlineAgents(ctx context.Context, tenantNo string) (int64, error) {
	n, err := d.client.Client().SCard(ctx, d.onlineAgentsKey(tenantNo)).Result()
	return n, err
}

// StartSubscribers 启动订阅监听
func (d *RedisDistributor) StartSubscribers(ctx context.Context, nodeID string, onNodeMessage func(ctx context.Context, data []byte), onTenantMessage func(ctx context.Context, data []byte)) {
	go d.subscribeWithReconnect(ctx, nodeID, onNodeMessage)
	go d.subscribeTenantWithReconnect(ctx, onTenantMessage)
}

// subscribeWithReconnect 带重连的节点通道订阅
func (d *RedisDistributor) subscribeWithReconnect(ctx context.Context, nodeID string, handler func(ctx context.Context, data []byte)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		nodePubSub := d.client.Client().Subscribe(ctx, d.nodeChannel(nodeID))
		if err := nodePubSub.Ping(ctx); err != nil {
			logger.ErrorfCtx(ctx, "节点订阅连接失败: %v, 3秒后重试", err)
			nodePubSub.Close()
			time.Sleep(3 * time.Second)
			continue
		}
		logger.InfofCtx(ctx, "节点订阅已启动, 通道: %s", d.nodeChannel(nodeID))
		d.consumeWithReconnect(ctx, nodePubSub, handler)
		time.Sleep(time.Second)
	}
}

// subscribeTenantWithReconnect 带重连的租户通道订阅
func (d *RedisDistributor) subscribeTenantWithReconnect(ctx context.Context, handler func(ctx context.Context, data []byte)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		tenantPubSub := d.client.Client().PSubscribe(ctx, fmt.Sprintf("%stenant:*", d.keyPrefix))
		if err := tenantPubSub.Ping(ctx); err != nil {
			logger.ErrorfCtx(ctx, "租户订阅连接失败: %v, 3秒后重试", err)
			tenantPubSub.Close()
			time.Sleep(3 * time.Second)
			continue
		}
		logger.InfofCtx(ctx, "租户订阅已启动, 模式: %stenant:*", d.keyPrefix)
		d.consumeWithReconnect(ctx, tenantPubSub, handler)
		time.Sleep(time.Second)
	}
}

// consumeWithReconnect 消费消息（通道关闭后自动重连）
func (d *RedisDistributor) consumeWithReconnect(ctx context.Context, pubSub *redis.PubSub, handler func(ctx context.Context, data []byte)) {
	ch := pubSub.Channel()
	for {
		select {
		case <-ctx.Done():
			pubSub.Close()
			return
		case msg, ok := <-ch:
			if !ok {
				pubSub.Close()
				return
			}
			handler(ctx, []byte(msg.Payload))
		}
	}
}

// Close 关闭连接
func (d *RedisDistributor) Close() error {
	return d.client.Close()
}
