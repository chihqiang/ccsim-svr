package handler

import (
	"context"

	"chihqiang/ccsim-svr/protocol"
	"chihqiang/ccsim-svr/ws"

	"github.com/chihqiang/infra-go/logger"
)

// HeartbeatHandler 心跳处理器
type HeartbeatHandler struct {
	hub *ws.Hub
}

// NewHeartbeatHandler 创建心跳处理器
func NewHeartbeatHandler(hub *ws.Hub) *HeartbeatHandler {
	return &HeartbeatHandler{hub: hub}
}

// Handle 处理心跳消息
func (h *HeartbeatHandler) Handle(ctx context.Context, conn ws.Connection, data []byte) error {
	logger.DebugfCtx(ctx, "收到心跳消息, 连接ID: %s", conn.GetID())
	conn.UpdateHeartbeat(ctx)

	resp := protocol.HeartbeatACKMessage{
		ServerMessage: protocol.ServerMessage{Type: protocol.ServerMsgHeartbeatACK},
	}
	return sendJSON(ctx, conn, resp)
}
