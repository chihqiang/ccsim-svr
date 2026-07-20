package ws

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"chihqiang/ccsim-svr/protocol"

	gws "github.com/chihqiang/infra-go/websocket"
)

// Connection WebSocket连接接口
type Connection interface {
	GetID() string
	GetUserID() int64
	SetUserID(ctx context.Context, userID int64)
	GetTenantNo() string
	SetTenantNo(ctx context.Context, tenantNo string)
	GetRole() protocol.Role
	SetRole(ctx context.Context, role protocol.Role)
	GetNickname() string
	SetNickname(ctx context.Context, nickname string)
	Send(ctx context.Context, data []byte) error
	Close(ctx context.Context) error
	IsClosed() bool
	GetLastHeartbeat() time.Time
	UpdateHeartbeat(ctx context.Context)
}

// Conn 包装 infra-go/websocket.Conn，实现 Connection 接口
type Conn struct {
	conn          *gws.Conn
	id            string // string(ConnID)，handler 层使用
	closed        atomic.Bool
	lastHeartbeat atomic.Int64 // UnixNano，避免 mutex
	onClose       func(ctx context.Context)
}

// NewConn 创建新连接
func NewConn(conn *gws.Conn, onClose func(ctx context.Context)) *Conn {
	return &Conn{
		conn:    conn,
		id:      formatConnID(conn.ID()),
		onClose: onClose,
	}
}

func formatConnID(id gws.ConnID) string {
	return fmt.Sprintf("%d", id)
}

// GetID 获取连接ID（string 格式，handler 层使用）
func (c *Conn) GetID() string {
	return c.id
}

// GetUserID 获取用户ID
func (c *Conn) GetUserID() int64 {
	v, _ := c.conn.Get("userID")
	if v == nil {
		return 0
	}
	return v.(int64)
}

// SetUserID 设置用户ID
func (c *Conn) SetUserID(_ context.Context, userID int64) {
	c.conn.Set("userID", userID)
}

// GetTenantNo 获取租户编号
func (c *Conn) GetTenantNo() string {
	v, _ := c.conn.Get("tenantNo")
	if v == nil {
		return ""
	}
	return v.(string)
}

// SetTenantNo 设置租户编号
func (c *Conn) SetTenantNo(_ context.Context, tenantNo string) {
	c.conn.Set("tenantNo", tenantNo)
}

// GetRole 获取角色
func (c *Conn) GetRole() protocol.Role {
	v, _ := c.conn.Get("role")
	if v == nil {
		return ""
	}
	return v.(protocol.Role)
}

// SetRole 设置角色
func (c *Conn) SetRole(_ context.Context, role protocol.Role) {
	c.conn.Set("role", role)
}

// GetNickname 获取昵称
func (c *Conn) GetNickname() string {
	v, _ := c.conn.Get("nickname")
	if v == nil {
		return ""
	}
	return v.(string)
}

// SetNickname 设置昵称
func (c *Conn) SetNickname(_ context.Context, nickname string) {
	c.conn.Set("nickname", nickname)
}

// Send 发送消息（infra-go 内部 mutex 保证线程安全）
func (c *Conn) Send(_ context.Context, data []byte) error {
	if c.closed.Load() {
		return ErrConnectionClosed
	}
	return c.conn.Push(data)
}

// Close 关闭连接
func (c *Conn) Close(ctx context.Context) error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	if c.onClose != nil {
		c.onClose(ctx)
	}
	return c.conn.Close()
}

// IsClosed 是否已关闭
func (c *Conn) IsClosed() bool {
	return c.closed.Load() || c.conn.IsClosed()
}

// GetLastHeartbeat 获取最后心跳时间
func (c *Conn) GetLastHeartbeat() time.Time {
	nano := c.lastHeartbeat.Load()
	if nano == 0 {
		return time.Time{}
	}
	return time.Unix(0, nano)
}

// UpdateHeartbeat 更新心跳时间
func (c *Conn) UpdateHeartbeat(_ context.Context) {
	c.lastHeartbeat.Store(time.Now().UnixNano())
}

// 错误定义
var (
	ErrConnectionClosed = &WsError{Code: "CONNECTION_CLOSED", Message: "连接已关闭"}
	ErrSendBufferFull   = &WsError{Code: "SEND_BUFFER_FULL", Message: "发送缓冲区已满"}
)

// WsError WebSocket错误
type WsError struct {
	Code    string
	Message string
}

func (e *WsError) Error() string {
	return e.Message
}
