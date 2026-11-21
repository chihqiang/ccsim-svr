package model

import "time"

// SenderRole 发送者角色
type SenderRole string

const (
	SenderRoleVisitor SenderRole = "visitor" // 访客
	SenderRoleAgent   SenderRole = "agent"   // 客服
	SenderRoleSystem  SenderRole = "system"  // 系统
)

// MsgType 消息类型
type MsgType string

const (
	MsgTypeText  MsgType = "text"  // 文本
	MsgTypeImage MsgType = "image" // 图片
	MsgTypeFile  MsgType = "file"  // 文件
)

// ReadStatus 已读状态
type ReadStatus int8

const (
	ReadStatusUnread ReadStatus = 0 // 未读
	ReadStatusRead   ReadStatus = 1 // 已读
)

// Message 消息
type Message struct {
	ID         int64      `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	SessionID  int64      `json:"sessionId" gorm:"not null;uniqueIndex:uk_session_seq;index:idx_session_time;comment:会话ID"`
	SenderRole SenderRole `json:"senderRole" gorm:"type:varchar(16);not null;comment:发送者角色 visitor/agent/system"`
	SenderID   int64      `json:"senderId" gorm:"not null;comment:发送者ID"`
	Nickname   string     `json:"nickname" gorm:"type:varchar(128);default:'';comment:发送者昵称"`
	Content    string     `json:"content" gorm:"type:text;not null;comment:消息内容"`
	MsgType    MsgType    `json:"msgType" gorm:"type:varchar(32);not null;default:'text';comment:消息类型 text/image/file"`
	SeqNum     int        `json:"seqNum" gorm:"not null;uniqueIndex:uk_session_seq;comment:会话内消息序号"`
	IsRead     ReadStatus `json:"isRead" gorm:"type:tinyint;not null;default:0;index:idx_session_read;comment:已读状态 1:已读 0:未读"`
	CreatedAt  time.Time  `json:"createdAt" gorm:"autoCreateTime;index:idx_session_time;comment:创建时间"`
}

// TableName 表名
func (Message) TableName() string {
	return "ccsim_messages"
}
