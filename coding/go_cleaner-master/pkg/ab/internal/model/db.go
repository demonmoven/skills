package model

import (
	"sync"

	"gorm.io/gorm"

	"code.byted.org/gorm/bytedgorm"
)

var db *gorm.DB
var once sync.Once

func DB() *gorm.DB {
	once.Do(func() {
		var err error
		db, err = gorm.Open(
			bytedgorm.MySQL("toutiao.mysql.gdp_data", "gdp_data"),
			bytedgorm.WithDefaults(),
		)
		if err != nil {
			panic("failed to connect database")
		}
	})
	return db
}
