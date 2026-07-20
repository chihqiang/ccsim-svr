package handler

import (
	"context"
	"time"

	"chihqiang/ccsim-svr/model"
	"chihqiang/ccsim-svr/protocol"

	"github.com/chihqiang/infra-go/logger"
)

// buildSessionListItems 构建会话列表项（公共逻辑）
func buildSessionListItems(sessions []*model.Session, lastMsgMap map[int64]string, visitorMap map[int64]*model.Visitor) []protocol.SessionListItem {
	items := make([]protocol.SessionListItem, 0, len(sessions))
	for _, s := range sessions {
		lastMsgTime := int64(0)
		if s.LastMsgTime != nil {
			lastMsgTime = s.LastMsgTime.UnixMilli()
		}

		item := protocol.SessionListItem{
			SessionID:       s.ID,
			VisitorID:       s.VisitorID,
			VisitorNickname: s.VisitorNickname,
			AgentID:         s.AgentID,
			AgentNickname:   s.AgentNickname,
			Status:          string(s.Status),
			Source:          s.Source,
			IP:              s.IP,
			Country:         s.Country,
			Province:        s.Province,
			City:            s.City,
			UserAgent:       s.UserAgent,
			Platform:        s.Platform,
			LastMsgContent:  lastMsgMap[s.LastMsgID],
			LastMsgTime:     lastMsgTime,
			UnreadCount:     s.UnreadCount,
			CreatedAt:       s.CreatedAt.UnixMilli(),
		}

		if v, ok := visitorMap[s.VisitorID]; ok {
			item.VisitorPhone = v.Phone
			item.VisitorExternalID = v.ExternalID
		}

		items = append(items, item)
	}
	return items
}

// buildWaitingListItems 构建等待会话列表项（公共逻辑）
func buildWaitingListItems(sessions []*model.Session, lastMsgMap map[int64]string, visitorMap map[int64]*model.Visitor) []protocol.WaitingSessionListItem {
	items := make([]protocol.WaitingSessionListItem, 0, len(sessions))
	for _, s := range sessions {
		waitingSeconds := int(time.Since(s.CreatedAt).Seconds())

		item := protocol.WaitingSessionListItem{
			SessionID:       s.ID,
			VisitorID:       s.VisitorID,
			VisitorNickname: s.VisitorNickname,
			Source:          s.Source,
			IP:              s.IP,
			Country:         s.Country,
			Province:        s.Province,
			City:            s.City,
			UserAgent:       s.UserAgent,
			Platform:        s.Platform,
			LastMsgContent:  lastMsgMap[s.LastMsgID],
			CreatedAt:       s.CreatedAt.UnixMilli(),
			WaitingSeconds:  waitingSeconds,
		}

		if v, ok := visitorMap[s.VisitorID]; ok {
			item.VisitorAvatar = v.Avatar
			item.VisitorPhone = v.Phone
			item.VisitorExternalID = v.ExternalID
		}

		items = append(items, item)
	}
	return items
}

// fetchSessionExtras 批量获取会话关联数据（最后消息、访客信息）
func (h *SessionHandler) fetchSessionExtras(ctx context.Context, sessions []*model.Session) (map[int64]string, map[int64]*model.Visitor) {
	lastMsgIDs := make([]int64, 0)
	visitorIDSet := make(map[int64]struct{}, len(sessions))
	for _, s := range sessions {
		if s.LastMsgID > 0 {
			lastMsgIDs = append(lastMsgIDs, s.LastMsgID)
		}
		if s.VisitorID > 0 {
			visitorIDSet[s.VisitorID] = struct{}{}
		}
	}

	lastMsgMap := make(map[int64]string)
	if len(lastMsgIDs) > 0 {
		if msgs, err := h.messageService.GetMessagesByIDs(ctx, lastMsgIDs); err != nil {
			logger.ErrorfCtx(ctx, "批量获取最后消息失败, 错误: %v", err)
		} else {
			for id, m := range msgs {
				lastMsgMap[id] = m.Content
			}
		}
	}

	visitorMap := make(map[int64]*model.Visitor)
	if len(visitorIDSet) > 0 {
		visitorIDs := make([]int64, 0, len(visitorIDSet))
		for id := range visitorIDSet {
			visitorIDs = append(visitorIDs, id)
		}
		if visitors, err := h.visitorService.GetVisitorsByIDs(ctx, visitorIDs); err != nil {
			logger.ErrorfCtx(ctx, "批量获取访客信息失败, 错误: %v", err)
		} else {
			visitorMap = visitors
		}
	}

	return lastMsgMap, visitorMap
}
