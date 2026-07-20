package repo

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

// WithTransaction 将事务注入上下文
func WithTransaction(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// DBFromContext 从上下文获取事务DB，无事务时返回nil
func DBFromContext(ctx context.Context) *gorm.DB {
	tx, _ := ctx.Value(txKey{}).(*gorm.DB)
	return tx
}

// TxDo 在事务中执行函数，支持嵌套复用
func TxDo(ctx context.Context, db *gorm.DB, fn func(txCtx context.Context) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := WithTransaction(ctx, tx)
		return fn(txCtx)
	})
}

// UseTx 如果上下文中有事务则用事务DB，否则用原始DB
func UseTx(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx := DBFromContext(ctx); tx != nil {
		return tx
	}
	return db
}
