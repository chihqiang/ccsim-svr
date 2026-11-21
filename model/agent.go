package model

import "time"

// AgentStatus 客服在线状态
type AgentStatus int8

const (
	AgentStatusOffline AgentStatus = 0 // 离线
	AgentStatusOnline  AgentStatus = 1 // 在线
)

// AgentStatusEnabled 客服启用状态
type AgentEnabled int8

const (
	AgentEnabledDisabled AgentEnabled = 0 // 禁用
	AgentEnabledEnabled  AgentEnabled = 1 // 启用
)

// Agent 客服
type Agent struct {
	ID        int64        `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	TenantNo  string       `json:"tenantNo" gorm:"type:varchar(64);not null;uniqueIndex:uk_tenant_account;comment:租户编号"`
	Account   string       `json:"account" gorm:"type:varchar(128);not null;uniqueIndex:uk_tenant_account;comment:客服账号"`
	Password  string       `json:"password" gorm:"type:varchar(256);not null;comment:密码"`
	Nickname  string       `json:"nickname" gorm:"type:varchar(128);default:'';comment:客服昵称"`
	Avatar    string       `json:"avatar" gorm:"type:varchar(512);default:'';comment:头像URL"`
	IsOnline  AgentStatus  `json:"isOnline" gorm:"type:tinyint;not null;default:0;comment:在线状态 1:在线 0:离线"`
	Status    AgentEnabled `json:"status" gorm:"type:tinyint;not null;default:1;comment:状态 1:启用 0:禁用"`
	CreatedAt time.Time    `json:"createdAt" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt time.Time    `json:"updatedAt" gorm:"autoUpdateTime;comment:更新时间"`
}

// TableName 表名
func (Agent) TableName() string {
	return "ccsim_agents"
}
