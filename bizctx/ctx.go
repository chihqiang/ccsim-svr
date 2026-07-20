package bizctx

import (
	"context"

	"github.com/chihqiang/infra-go/logger"
)

type contextKey string

const (
	keyNodeID    contextKey = "node_id"
	keyConnID    contextKey = "conn_id"
	keyTenantNo  contextKey = "tenant_no"
	keyUserID    contextKey = "user_id"
	keyRole      contextKey = "role"
	keySessionID contextKey = "session_id"
	keyMsgType   contextKey = "msg_type"
)

func WithNodeID(ctx context.Context, nodeID string) context.Context {
	return context.WithValue(ctx, keyNodeID, nodeID)
}

func WithConnID(ctx context.Context, connID string) context.Context {
	return context.WithValue(ctx, keyConnID, connID)
}

func WithTenantNo(ctx context.Context, tenantNo string) context.Context {
	return context.WithValue(ctx, keyTenantNo, tenantNo)
}

func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, keyUserID, userID)
}

func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, keyRole, role)
}

func WithSessionID(ctx context.Context, sessionID int64) context.Context {
	return context.WithValue(ctx, keySessionID, sessionID)
}

func WithMsgType(ctx context.Context, msgType string) context.Context {
	return context.WithValue(ctx, keyMsgType, msgType)
}

func NodeIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(keyNodeID).(string)
	return v
}

func ConnIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(keyConnID).(string)
	return v
}

func TenantNoFrom(ctx context.Context) string {
	v, _ := ctx.Value(keyTenantNo).(string)
	return v
}

func UserIDFrom(ctx context.Context) int64 {
	v, _ := ctx.Value(keyUserID).(int64)
	return v
}

func RoleFrom(ctx context.Context) string {
	v, _ := ctx.Value(keyRole).(string)
	return v
}

func SessionIDFrom(ctx context.Context) int64 {
	v, _ := ctx.Value(keySessionID).(int64)
	return v
}

func MsgTypeFrom(ctx context.Context) string {
	v, _ := ctx.Value(keyMsgType).(string)
	return v
}

// Register 注册日志上下文提取器，自动将 ctx 中的连接字段注入日志
func Register() {
	logger.RegisterContextExtractor(func(ctx context.Context) []logger.Field {
		fields := make([]logger.Field, 0, 6)
		if v := NodeIDFrom(ctx); v != "" {
			fields = append(fields, logger.String("node_id", v))
		}
		if v := ConnIDFrom(ctx); v != "" {
			fields = append(fields, logger.String("conn_id", v))
		}
		if v := TenantNoFrom(ctx); v != "" {
			fields = append(fields, logger.String("tenant_no", v))
		}
		if v := UserIDFrom(ctx); v > 0 {
			fields = append(fields, logger.Int64("user_id", v))
		}
		if v := RoleFrom(ctx); v != "" {
			fields = append(fields, logger.String("role", v))
		}
		if v := SessionIDFrom(ctx); v > 0 {
			fields = append(fields, logger.Int64("session_id", v))
		}
		return fields
	})
}
