package config

import (
	"github.com/chihqiang/infra-go/conf"
	"github.com/chihqiang/infra-go/logger"
)

func Load(file string) *Config {
	cfg := &Config{}
	if err := conf.Load(file, cfg); err != nil {
		logger.Fatalf("加载配置失败: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		logger.Fatalf("配置校验失败: %v", err)
	}
	return cfg
}
