package serv

import (
	"chihqiang/ccsim-svr/app"
	"context"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/logger"
)

// Server 统一服务（HTTP + WebSocket 共用端口）
type Server struct {
	ctx    context.Context
	app    *app.App
	server *httpx.Server
}

// NewServer 创建服务，注册 HTTP 路由到内部 mux
func NewServer(ctx context.Context, app *app.App, option httpx.RunOption) *Server {
	server := httpx.NewServer(app.Cfg.Server, option)
	return &Server{ctx: ctx, server: server, app: app}
}

// Start 启动服务
func (s *Server) Start() {
	logger.InfoCtx(s.ctx, "HTTP服务启动中...")
	s.app.Hub.StartCleanup(s.ctx)
	if err := s.server.Start(); err != nil {
		logger.ErrorfCtx(s.ctx, "HTTP服务异常退出: %v", err)
	}
}

// Stop 停止服务
func (s *Server) Stop() {
	logger.InfoCtx(s.ctx, "HTTP服务关闭中...")
	s.app.Hub.Shutdown(s.ctx)
	if s.server != nil {
		if err := s.server.Shutdown(); err != nil {
			logger.ErrorfCtx(s.ctx, "HTTP服务关闭失败: %v", err)
		}
	}
	logger.InfoCtx(s.ctx, "HTTP服务已关闭")
}
