package ws

import "context"

// MessageHandler 消息处理器接口
type MessageHandler interface {
	Handle(ctx context.Context, conn Connection, data []byte) error
}

// ConnectionHandler 连接处理器接口
type ConnectionHandler interface {
	OnConnect(ctx context.Context, conn Connection) error
	OnDisconnect(ctx context.Context, conn Connection) error
	OnMessage(ctx context.Context, conn Connection, data []byte) error
}

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	OnError(ctx context.Context, conn Connection, err error)
}

var (
	ErrUnknownMessageType = &WsError{Code: "UNKNOWN_MESSAGE_TYPE", Message: "未知消息类型"}
	ErrInvalidRole        = &WsError{Code: "INVALID_ROLE", Message: "无效角色"}
	ErrUnauthorized       = &WsError{Code: "UNAUTHORIZED", Message: "未授权"}
	ErrRateLimited        = &WsError{Code: "RATE_LIMITED", Message: "请求过于频繁"}
)
