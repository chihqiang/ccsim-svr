package protocol

// ClientMessage 客户端消息基础结构
type ClientMessage struct {
	Type      ClientMessageType `json:"type"`
	TraceID   string            `json:"trace_id"`
	SessionID int64             `json:"session_id"`
}

// AuthMessage 认证消息
type AuthMessage struct {
	ClientMessage
	TenantNo          string `json:"tenant_no"`
	Role              string `json:"role"`
	ExternalVisitorID string `json:"external_visitor_id,omitempty"`
	Nickname          string `json:"nickname,omitempty"`
	Phone             string `json:"phone,omitempty"`
	Platform          string `json:"platform,omitempty"`
	AgentAccount      string `json:"agent_account,omitempty"`
	AgentPassword     string `json:"agent_password,omitempty"`
}

// ChatSendMessage 发送聊天消息
type ChatSendMessage struct {
	ClientMessage
	Content string `json:"content"`
	MsgType string `json:"msg_type"`
	TempID  string `json:"temp_id"`
}

// SessionAcceptMessage 接受会话
type SessionAcceptMessage struct {
	ClientMessage
}

// SessionCloseMessage 关闭会话
type SessionCloseMessage struct {
	ClientMessage
}

// SessionListMessage 请求会话列表
type SessionListMessage struct {
	ClientMessage
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Status string `json:"status,omitempty"`
}

// SessionHistoryMessage 请求历史消息
type SessionHistoryMessage struct {
	ClientMessage
	BeforeSeq int `json:"before_seq,omitempty"`
	Limit     int `json:"limit,omitempty"`
}

// SatisfactionRateMessage 满意度评价
type SatisfactionRateMessage struct {
	ClientMessage
	Rating int `json:"rating"`
}

// VisitorUpdateMessage 更新访客信息
type VisitorUpdateMessage struct {
	ClientMessage
	Nickname string `json:"nickname,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
	Metadata string `json:"metadata,omitempty"`
}

// TypingMessage 正在输入
type TypingMessage struct {
	ClientMessage
}

// MessageReadMessage 消息已读
type MessageReadMessage struct {
	ClientMessage
	MsgID  int64 `json:"msg_id"`
	SeqNum int   `json:"seq_num"`
}
