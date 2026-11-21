package config

import (
	"errors"

	"github.com/chihqiang/ten-go/core/logger"
	"github.com/chihqiang/ten-go/core/orm"
	"github.com/chihqiang/ten-go/core/trace"
	"github.com/chihqiang/ten-go/rest"
	gws "github.com/chihqiang/ten-go/ws"
)

// Config 应用配置
type Config struct {
	// rest.Config HTTP 服务配置（嵌入：host/port/logger/middlewares/health 位于配置顶层）。
	rest.Config
	Env      string        `json:"env" default:"development"`
	WS       gws.Config    `json:"ws"`
	Database orm.Config    `json:"database"`
	Redis    RedisConfig   `json:"redis"`
	Trace    trace.Config  `json:"trace"`
	Log      logger.Config `json:"log"`
}

// Validate 校验配置
func (c *Config) Validate() error {
	if c.Env == "" {
		c.Env = "development"
	}
	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("port 必须在 1-65535 之间")
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
