package model

import "time"

type Base struct {
	Deleted    int8
	CreateTime *time.Time `gorm:"default:current_time"`
	ModifyTime *time.Time `gorm:"default:current_time"`
}
