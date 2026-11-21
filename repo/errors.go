package repo

import (
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("记录不存在")

// MapGormError 将 gorm 错误映射为业务错误
func MapGormError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
