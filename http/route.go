package httphandler

import (
	"chihqiang/ccsim-svr/app"
	"chihqiang/ccsim-svr/http/handler"
	"github.com/chihqiang/ten-go/rest"
	"net/http"
)

// Routes 返回所有 HTTP 路由
func Routes(app *app.App) rest.Option {
	return func(server *rest.Server) {
		server.AddRoutes([]rest.Route{
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
		}...)
	}
}
