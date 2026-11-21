package protocol

// ServerMessage 服务端消息基础结构
type ServerMessage struct {
	Type    ServerMessageType `json:"type"`
	TraceID string            `json:"trace_id,omitempty"`
}

// AuthOKMessage 认证成功
type AuthOKMessage struct {
	ServerMessage
	ConnID    string `json:"conn_id"`
	AgentID   int64  `json:"agent_id,omitempty"`
	VisitorID int64  `json:"visitor_id,omitempty"`
}

// ChatACKMessage 发送确认
type ChatACKMessage struct {
	ServerMessage
	TempID    string `json:"temp_id"`
	MsgID     int64  `json:"msg_id"`
	SessionID int64  `json:"session_id"`
	SeqNum    int    `json:"seq_num"`
	CreatedAt int64  `json:"created_at"`
}

// ChatPushMessage 推送聊天消息
type ChatPushMessage struct {
	ServerMessage
	MsgID      int64  `json:"msg_id"`
	SessionID  int64  `json:"session_id"`
	SenderRole string `json:"sender_role"`
	SenderID   int64  `json:"sender_id"`
	Nickname   string `json:"nickname"`
	Content    string `json:"content"`
	MsgType    string `json:"msg_type"`
	SeqNum     int    `json:"seq_num"`
	CreatedAt  int64  `json:"created_at"`
}

// OfflinePushMessage 离线消息推送
type OfflinePushMessage struct {
	ServerMessage
	Messages []ChatPushMessage `json:"messages"`
}

// HistoryBatchMessage 历史消息批次
type HistoryBatchMessage struct {
	ServerMessage
	SessionID int64             `json:"session_id"`
	Count     int               `json:"count,omitempty"`
	Data      []ChatPushMessage `json:"data"`
}

// SessionCreatedMessage 会话已创建
type SessionCreatedMessage struct {
	ServerMessage
	SessionID int64  `json:"session_id"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

// SessionAssignedMessage 会话已分配
type SessionAssignedMessage struct {
	ServerMessage
	SessionID     int64  `json:"session_id"`
	AgentID       int64  `json:"agent_id"`
	AgentNickname string `json:"agent_nickname"`
	AgentAvatar   string `json:"agent_avatar"`
}

// SessionClosedMessage 会话已关闭
type SessionClosedMessage struct {
	ServerMessage
	SessionID   int64  `json:"session_id"`
	CloseReason string `json:"close_reason"`
}

// SessionListItem 会话列表项
type SessionListItem struct {
	SessionID         int64  `json:"session_id"`
	VisitorID         int64  `json:"visitor_id"`
	VisitorNickname   string `json:"visitor_nickname"`
	VisitorPhone      string `json:"visitor_phone,omitempty"`
	VisitorExternalID string `json:"visitor_external_id,omitempty"`
	AgentID           int64  `json:"agent_id"`
	AgentNickname     string `json:"agent_nickname"`
	Status            string `json:"status"`
	Source            string `json:"source,omitempty"`
	IP                string `json:"ip,omitempty"`
	Country           string `json:"country,omitempty"`
	Province          string `json:"province,omitempty"`
	City              string `json:"city,omitempty"`
	UserAgent         string `json:"user_agent,omitempty"`
	Platform          string `json:"platform,omitempty"`
	LastMsgContent    string `json:"last_msg_content"`
	LastMsgTime       int64  `json:"last_msg_time"`
	UnreadCount       int    `json:"unread_count"`
	CreatedAt         int64  `json:"created_at"`
}

// SessionListResMessage 会话列表响应
type SessionListResMessage struct {
	ServerMessage
	Items []SessionListItem `json:"items"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}

// WaitingSessionListItem 等待会话列表项
type WaitingSessionListItem struct {
	SessionID         int64  `json:"session_id"`
	VisitorID         int64  `json:"visitor_id"`
	VisitorNickname   string `json:"visitor_nickname"`
	VisitorAvatar     string `json:"visitor_avatar"`
	VisitorPhone      string `json:"visitor_phone,omitempty"`
	VisitorExternalID string `json:"visitor_external_id,omitempty"`
	Source            string `json:"source"`
	IP                string `json:"ip,omitempty"`
	Country           string `json:"country,omitempty"`
	Province          string `json:"province,omitempty"`
	City              string `json:"city,omitempty"`
	UserAgent         string `json:"user_agent,omitempty"`
	Platform          string `json:"platform,omitempty"`
	LastMsgContent    string `json:"last_msg_content"`
	CreatedAt         int64  `json:"created_at"`
	WaitingSeconds    int    `json:"waiting_seconds"`
}

// WaitingSessionListResMessage 等待会话列表响应
type WaitingSessionListResMessage struct {
	ServerMessage
	Items []WaitingSessionListItem `json:"items"`
}

// NewSessionMessage 新会话通知
type NewSessionMessage struct {
	ServerMessage
	SessionID         int64  `json:"session_id"`
	VisitorID         int64  `json:"visitor_id"`
	VisitorNickname   string `json:"visitor_nickname"`
	VisitorPhone      string `json:"visitor_phone,omitempty"`
	VisitorExternalID string `json:"visitor_external_id,omitempty"`
	Source            string `json:"source,omitempty"`
	IP                string `json:"ip,omitempty"`
	Platform          string `json:"platform,omitempty"`
	LastMsgContent    string `json:"last_msg_content"`
	CreatedAt         int64  `json:"created_at"`
}

// AgentOnlineAckMessage 客服上线确认
type AgentOnlineAckMessage struct {
	ServerMessage
	AgentID int64  `json:"agent_id"`
	Status  string `json:"status"`
}

// AgentOfflineAckMessage 客服下线确认
type AgentOfflineAckMessage struct {
	ServerMessage
	AgentID int64  `json:"agent_id"`
	Status  string `json:"status"`
}

// AgentStatusMessage 客服状态
type AgentStatusMessage struct {
	ServerMessage
	OnlineCount    int  `json:"online_count"`
	HasOnlineAgent bool `json:"has_online_agent"`
}

// TypingPushMessage 正在输入推送
type TypingPushMessage struct {
	ServerMessage
	SessionID  int64  `json:"session_id"`
	SenderRole string `json:"sender_role"`
	SenderID   int64  `json:"sender_id"`
}

// MessageReadPushMessage 消息已读推送
type MessageReadPushMessage struct {
	ServerMessage
	SessionID  int64  `json:"session_id"`
	ReaderRole string `json:"reader_role"`
	ReaderID   int64  `json:"reader_id"`
	MsgID      int64  `json:"msg_id"`
	SeqNum     int    `json:"seq_num"`
}

// VisitorInfoUpdatedMessage 访客信息已更新
type VisitorInfoUpdatedMessage struct {
	ServerMessage
	SessionID int64  `json:"session_id"`
	VisitorID int64  `json:"visitor_id"`
	Nickname  string `json:"nickname,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
}

// SatisfactionRateResponseMessage 满意度评价响应
type SatisfactionRateResponseMessage struct {
	ServerMessage
	SessionID int64  `json:"session_id"`
	Status    string `json:"status"`
}

// ErrorMessage 错误消息
type ErrorMessage struct {
	ServerMessage
	Code   string `json:"code"`
	ErrMsg string `json:"err_msg,omitempty"`
}

// HeartbeatACKMessage 心跳确认
type HeartbeatACKMessage struct {
	ServerMessage
}

// SessionAcceptAckMessage 会话接受确认
type SessionAcceptAckMessage struct {
	ServerMessage
	SessionID int64  `json:"session_id"`
	Status    string `json:"status"`
}

// SessionCloseAckMessage 会话关闭确认
type SessionCloseAckMessage struct {
	ServerMessage
	SessionID   int64  `json:"session_id"`
	CloseReason string `json:"close_reason"`
}
