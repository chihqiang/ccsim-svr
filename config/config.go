package config

import (
	"errors"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/orm"
	"github.com/chihqiang/infra-go/trace"
	gws "github.com/chihqiang/infra-go/websocket"
)

// Config 应用配置
type Config struct {
	Env      string             `json:"env" default:"development"`
	Server   httpx.ServerConfig `json:"server"`
	WS       gws.Config         `json:"ws"`
	Database orm.Config         `json:"database"`
	Redis    RedisConfig        `json:"redis"`
	Trace    trace.Config       `json:"trace"`
	Log      logger.Config      `json:"log"`
}

// Validate 校验配置
func (c *Config) Validate() error {
	if c.Env == "" {
		c.Env = "development"
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return errors.New("server.port 必须在 1-65535 之间")
	}
	if c.Database.Driver == "" && c.Database.DSN == "" {
		return errors.New("database.driver 和 database.dsn 至少配置一个")
	}
	if c.Redis.Enabled && c.Redis.Addr == "" {
		return errors.New("redis.addr 在启用 Redis 时不能为空")
	}
	if c.Redis.Enabled && (c.Redis.DB < 0 || c.Redis.DB > 15) {
		return errors.New("redis.db 必须在 0-15 之间")
	}
	return nil
}

// RedisConfig Redis配置
type RedisConfig struct {
	Enabled     bool   `json:"enabled"`
	Addr        string `json:"addr" default:"127.0.0.1:6379"`
	Password    string `json:"password"`
	DB          int    `json:"db"`
	RegistryTTL int    `json:"registryTTL" default:"90"` // 用户注册TTL（秒），需大于心跳超时
}
