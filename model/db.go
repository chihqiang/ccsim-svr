package model

import (
	"github.com/chihqiang/infra-go/hash"
	"gorm.io/gorm"
)

// Migrate 数据库迁移
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Tenant{},
		&Visitor{},
		&Agent{},
		&Session{},
		&Message{},
		&SatisfactionRating{},
	)
}

// Seed 生成测试数据
func Seed(db *gorm.DB) error {
	// 检查是否已有数据
	var count int64
	db.Model(&Tenant{}).Count(&count)
	if count > 0 {
		return nil
	}

	// 创建租户
	tenant := &Tenant{
		TenantNo: "test",
		Name:     "测试租户",
		Status:   TenantStatusEnabled,
	}
	if err := db.Create(tenant).Error; err != nil {
		return err
	}

	// 创建客服
	pwd, err := hash.BcryptHashDefault("admin123")
	if err != nil {
		return err
	}
	agent := &Agent{
		TenantNo: "test",
		Account:  "admin",
		Password: pwd,
		Nickname: "客服小王",
		Status:   AgentEnabledEnabled,
	}
	if err := db.Create(agent).Error; err != nil {
		return err
	}
	return nil
}
