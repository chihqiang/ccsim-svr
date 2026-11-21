package model

import "time"

// Visitor 访客
type Visitor struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	TenantNo   string    `json:"tenantNo" gorm:"type:varchar(64);not null;uniqueIndex:uk_tenant_external;comment:租户编号"`
	ExternalID string    `json:"externalId" gorm:"type:varchar(128);default:'';uniqueIndex:uk_tenant_external;comment:外部访客标识"`
	Nickname   string    `json:"nickname" gorm:"type:varchar(128);default:'';comment:访客昵称"`
	Phone      string    `json:"phone" gorm:"type:varchar(32);default:'';comment:手机号"`
	Platform   string    `json:"platform" gorm:"type:varchar(32);default:'';comment:平台 web/ios/android"`
	Avatar     string    `json:"avatar" gorm:"type:varchar(512);default:'';comment:头像URL"`
	Metadata   string    `json:"metadata" gorm:"type:text;comment:扩展元数据"`
	CreatedAt  time.Time `json:"createdAt" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt  time.Time `json:"updatedAt" gorm:"autoUpdateTime;comment:更新时间"`
}

// TableName 表名
func (Visitor) TableName() string {
	return "ccsim_visitors"
}
