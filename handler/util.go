package handler

import (
	"context"
	"encoding/json"

	"chihqiang/ccsim-svr/model"
	"chihqiang/ccsim-svr/protocol"
	"chihqiang/ccsim-svr/ws"

	"github.com/chihqiang/infra-go/logger"
)

// sendJSON 发送JSON消息给连接
func sendJSON(ctx context.Context, conn ws.Connection, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return conn.Send(ctx, data)
}

// mustMarshal 序列化JSON（失败时记录日志并返回nil+error）
func mustMarshal(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		logger.ErrorfCtx(context.Background(), "JSON序列化失败: %v", err)
		return nil, err
	}
	return data, nil
}

// toChatPushMessage 将 model.Message 转换为 protocol.ChatPushMessage
func toChatPushMessage(m *model.Message) protocol.ChatPushMessage {
	return protocol.ChatPushMessage{
		MsgID:      m.ID,
		SessionID:  m.SessionID,
		SenderRole: string(m.SenderRole),
		SenderID:   m.SenderID,
		Nickname:   m.Nickname,
		Content:    m.Content,
		MsgType:    string(m.MsgType),
		SeqNum:     m.SeqNum,
		CreatedAt:  m.CreatedAt.UnixMilli(),
	}
}
