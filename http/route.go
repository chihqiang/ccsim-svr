package httphandler

import (
	"chihqiang/ccsim-svr/app"
	"chihqiang/ccsim-svr/http/handler"
	"github.com/chihqiang/infra-go/httpx"
	"net/http"
)

// Routes 返回所有 HTTP 路由
func Routes(app *app.App) httpx.RunOption {
	return func(server *httpx.Server) {
		server.AddRoutes([]httpx.Route{
			{
				Method:  http.MethodGet,
				Path:    "/health",
				Handler: handler.NewHealthHandler(app).Handle(),
			},
			{
				Method:  http.MethodGet,
				Path:    "/ws",
				Handler: app.Hub.Server().ServeHTTP,
			},
		})
	}
}
