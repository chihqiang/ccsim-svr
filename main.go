package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"chihqiang/ccsim-svr/app"
	"chihqiang/ccsim-svr/bizctx"
	"chihqiang/ccsim-svr/config"
	httphandler "chihqiang/ccsim-svr/http"
	"chihqiang/ccsim-svr/serv"

	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/service"
	"github.com/chihqiang/infra-go/trace"
)

func main() {
	rootCtx := context.Background()
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	flag.Parse()
	cfg := config.Load(*configPath)
	bizctx.Register()
	traceCtx, rootSpan := trace.StartSpan(bizctx.WithNodeID(rootCtx, "boot"), cfg.Log.AppName)
	defer rootSpan.End()

	traceCfg := cfg.Trace
	if !traceCfg.Disabled && traceCfg.Endpoint == "" {
		traceCfg.Disabled = true
	}
	trace.AddResources(
		trace.AttrString("service", cfg.Log.AppName),
		trace.AttrString("env", cfg.Env),
	)
	trace.StartAgent(traceCfg)
	defer trace.StopAgent()
	a := app.NewApp(traceCtx, cfg)
	defer logger.Sync()
	sg := service.NewServiceGroup()
	sg.Add(serv.NewCluster(traceCtx, a))
	sg.Add(serv.NewServer(traceCtx, a, httphandler.Routes(a)))
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.InfoCtx(traceCtx, "收到退出信号，正在关闭服务...")
		sg.Stop()
	}()
	sg.Start()
	logger.InfoCtx(traceCtx, "服务已关闭")
}
