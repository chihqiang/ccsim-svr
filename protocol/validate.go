package protocol

import "errors"

var (
	ErrInvalidTenantNo   = errors.New("租户编号不能为空")
	ErrInvalidRole       = errors.New("角色不能为空")
	ErrInvalidSessionID  = errors.New("会话ID无效")
	ErrInvalidContent    = errors.New("消息内容不能为空")
	ErrInvalidRating     = errors.New("评分必须在1-3之间")
	ErrInvalidExternalID = errors.New("外部访客标识不能为空")
	ErrInvalidAccount    = errors.New("客服账号不能为空")
	ErrInvalidPassword   = errors.New("客服密码不能为空")
)

func (m *AuthMessage) Validate() error {
	if m.TenantNo == "" {
		return ErrInvalidTenantNo
	}
	if m.Role == "" {
		return ErrInvalidRole
	}
	if Role(m.Role) == RoleVisitor && m.ExternalVisitorID == "" {
		return ErrInvalidExternalID
	}
	if Role(m.Role) == RoleAgent {
		if m.AgentAccount == "" {
			return ErrInvalidAccount
		}
		if m.AgentPassword == "" {
			return ErrInvalidPassword
		}
	}
	return nil
}

func (m *ChatSendMessage) Validate() error {
	if m.Content == "" {
		return ErrInvalidContent
	}
	return nil
}

func (m *SessionAcceptMessage) Validate() error {
	if m.SessionID <= 0 {
		return ErrInvalidSessionID
	}
	return nil
}

func (m *SessionCloseMessage) Validate() error {
	if m.SessionID <= 0 {
		return ErrInvalidSessionID
	}
	return nil
}

func (m *SessionHistoryMessage) Validate() error {
	if m.SessionID <= 0 {
		return ErrInvalidSessionID
	}
	return nil
}

func (m *SatisfactionRateMessage) Validate() error {
	if m.SessionID <= 0 {
		return ErrInvalidSessionID
	}
	if m.Rating < 1 || m.Rating > 3 {
		return ErrInvalidRating
	}
	return nil
}

func (m *TypingMessage) Validate() error {
	if m.SessionID <= 0 {
		return ErrInvalidSessionID
	}
	return nil
}

func (m *MessageReadMessage) Validate() error {
	if m.SessionID <= 0 {
		return ErrInvalidSessionID
	}
	return nil
}
