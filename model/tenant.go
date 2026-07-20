package model

import "time"

// TenantStatus 租户状态
type TenantStatus int8

const (
	TenantStatusEnabled  TenantStatus = 1 // 启用
	TenantStatusDisabled TenantStatus = 0 // 禁用
)

// Tenant 租户
type Tenant struct {
	ID        int64        `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	TenantNo  string       `json:"tenantNo" gorm:"type:varchar(64);uniqueIndex;not null;comment:租户编号"`
	Name      string       `json:"name" gorm:"type:varchar(128);default:'';comment:租户名称"`
	Status    TenantStatus `json:"status" gorm:"type:tinyint;not null;default:1;comment:状态 1:启用 0:禁用"`
	CreatedAt time.Time    `json:"createdAt" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt time.Time    `json:"updatedAt" gorm:"autoUpdateTime;comment:更新时间"`
}

// TableName 表名
func (Tenant) TableName() string {
	return "ccsim_tenants"
}
