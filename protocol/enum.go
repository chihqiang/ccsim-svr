package protocol

// ClientMessageType 客户端消息类型
type ClientMessageType string

const (
	ClientMsgAuth             ClientMessageType = "auth"
	ClientMsgHeartbeat        ClientMessageType = "heartbeat"
	ClientMsgChatSend         ClientMessageType = "chat_send"
	ClientMsgAgentOnline      ClientMessageType = "agent_online"
	ClientMsgAgentOffline     ClientMessageType = "agent_offline"
	ClientMsgSessionAccept    ClientMessageType = "session_accept"
	ClientMsgSessionClose     ClientMessageType = "session_close"
	ClientMsgSessionList      ClientMessageType = "session_list"
	ClientMsgSessionHistory   ClientMessageType = "session_history"
	ClientMsgWaitingList      ClientMessageType = "waiting_session_list"
	ClientMsgSatisfactionRate ClientMessageType = "satisfaction_rate"
	ClientMsgVisitorUpdate    ClientMessageType = "visitor_update"
	ClientMsgTyping           ClientMessageType = "typing"
	ClientMsgMessageRead      ClientMessageType = "message_read"
)

// ServerMessageType 服务端消息类型
type ServerMessageType string

const (
	ServerMsgAuthOK             ServerMessageType = "auth_ok"
	ServerMsgHeartbeatACK       ServerMessageType = "heartbeat_ack"
	ServerMsgChatACK            ServerMessageType = "chat_ack"
	ServerMsgChatPush           ServerMessageType = "chat_push"
	ServerMsgOfflinePush        ServerMessageType = "offline_push"
	ServerMsgHistoryBatch       ServerMessageType = "history_batch"
	ServerMsgSessionCreated     ServerMessageType = "session_created"
	ServerMsgSessionAssigned    ServerMessageType = "session_assigned"
	ServerMsgSessionClosed      ServerMessageType = "session_closed"
	ServerMsgSessionListRes     ServerMessageType = "session_list_res"
	ServerMsgWaitingListRes     ServerMessageType = "waiting_session_list_res"
	ServerMsgNewSession         ServerMessageType = "new_session"
	ServerMsgAgentStatus        ServerMessageType = "agent_status"
	ServerMsgAgentOnlineAck     ServerMessageType = "agent_online"
	ServerMsgAgentOfflineAck    ServerMessageType = "agent_offline"
	ServerMsgTypingPush         ServerMessageType = "typing_push"
	ServerMsgMessageReadPush    ServerMessageType = "message_read_push"
	ServerMsgVisitorInfoUpdated ServerMessageType = "visitor_info_updated"
	ServerMsgSatisfactionRate   ServerMessageType = "satisfaction_rate"
	ServerMsgVisitorUpdateOK    ServerMessageType = "visitor_update_ok"
	ServerMsgSessionAcceptACK   ServerMessageType = "session_accept"
	ServerMsgSessionCloseACK    ServerMessageType = "session_close"
	ServerMsgError              ServerMessageType = "error"
)

// Role 连接角色
type Role string

const (
	RoleVisitor Role = "visitor"
	RoleAgent   Role = "agent"
)

// AgentOnlineStatus 客服在线状态（用于 ack 响应）
type AgentOnlineStatus string

const (
	AgentOnlineStatusOnline  AgentOnlineStatus = "online"
	AgentOnlineStatusOffline AgentOnlineStatus = "offline"
)

// CloseReason 会话关闭原因
type CloseReason string

const (
	CloseReasonAgent   CloseReason = "客服关闭"
	CloseReasonVisitor CloseReason = "访客关闭"
	CloseReasonSystem  CloseReason = "系统关闭"
	CloseReasonTimeout CloseReason = "超时关闭"
)
