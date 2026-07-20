package model

import "time"

// SessionStatus 会话状态
type SessionStatus string

const (
	SessionStatusPending SessionStatus = "pending" // 等待接入
	SessionStatusActive  SessionStatus = "active"  // 进行中
	SessionStatusClosed  SessionStatus = "closed"  // 已关闭
)

// Session 会话
type Session struct {
	ID              int64         `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	TenantNo        string        `json:"tenantNo" gorm:"type:varchar(64);not null;index;comment:租户编号"`
	VisitorID       int64         `json:"visitorId" gorm:"not null;index;comment:访客ID"`
	AgentID         int64         `json:"agentId" gorm:"index;comment:客服ID"`
	VisitorNickname string        `json:"visitorNickname" gorm:"type:varchar(128);default:'';comment:访客昵称"`
	AgentNickname   string        `json:"agentNickname" gorm:"type:varchar(128);default:'';comment:客服昵称"`
	Status          SessionStatus `json:"status" gorm:"type:varchar(32);not null;default:'pending';index:idx_tenant_status;comment:状态 pending/active/closed"`
	Source          string        `json:"source" gorm:"type:varchar(64);default:'';comment:来源渠道"`
	IP              string        `json:"ip" gorm:"type:varchar(45);default:'';comment:访客IP"`
	Country         string        `json:"country" gorm:"type:varchar(64);default:'';comment:国家"`
	Province        string        `json:"province" gorm:"type:varchar(64);default:'';comment:省份"`
	City            string        `json:"city" gorm:"type:varchar(64);default:'';comment:城市"`
	UserAgent       string        `json:"userAgent" gorm:"type:varchar(512);default:'';comment:浏览器UA"`
	Platform        string        `json:"platform" gorm:"type:varchar(32);default:'';comment:平台"`
	CloseReason     string        `json:"closeReason" gorm:"type:varchar(256);default:'';comment:关闭原因"`
	LastMsgID       int64         `json:"lastMsgId" gorm:"comment:最近消息ID"`
	LastMsgTime     *time.Time    `json:"lastMsgTime" gorm:"comment:最近消息时间"`
	UnreadCount     int           `json:"unreadCount" gorm:"not null;default:0;comment:未读消息数"`
	CreatedAt       time.Time     `json:"createdAt" gorm:"autoCreateTime;comment:创建时间"`
	AssignedAt      *time.Time    `json:"assignedAt" gorm:"comment:分配客服时间"`
	ClosedAt        *time.Time    `json:"closedAt" gorm:"comment:关闭时间"`
}

// TableName 表名
func (Session) TableName() string {
	return "ccsim_sessions"
}
