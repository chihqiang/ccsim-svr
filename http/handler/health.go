package handler

import (
	"encoding/json"
	"net/http"

	"chihqiang/ccsim-svr/app"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	app *app.App
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(app *app.App) *HealthHandler {
	return &HealthHandler{app: app}
}

// Handle 处理健康检查请求
func (h *HealthHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "ok",
			"node_id":     h.app.NodeID,
			"connections": h.app.Hub.Count(r.Context()),
		})
	}
}
