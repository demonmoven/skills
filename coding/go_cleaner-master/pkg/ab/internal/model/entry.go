package model

import (
	"time"

	"gorm.io/gorm"
)

type AuthorList []string

func (a *AuthorList) Add(s string) {
	for _, i := range *a {
		if i == s {
			return
		}
	}
	*a = append(*a, s)
	return
}

func (a *AuthorList) Has(s string) bool {
	for _, i := range *a {
		if i == s {
			return true
		}
	}
	return false
}

func (a *AuthorList) Del(s string) {
	var an []string
	for _, i := range *a {
		if i != s {
			an = append(an, i)
		}
	}
	*a = an
	return
}

type ABEntry struct {
	ID           uint64         `gorm:"column:id" json:"id"`                                // 主键id
	Package      string         `gorm:"column:package" json:"package"`                      // 仓库名
	Recv         string         `gorm:"column:recv" json:"recv"`                            // 仓库名
	Func         []string       `gorm:"column:func;serializer:json" json:"func" `           // funcNameList
	KeyIndex     []int          `gorm:"column:key_index;serializer:json" json:"key_index" ` // funcNameList
	DefaultIndex int            `gorm:"column:default_index" json:"default_index"`
	KeyPrefix    string         `gorm:"column:key_prefix" json:"key_prefix"`
	DefaultVal   string         `gorm:"column:default_val" json:"default_val"`
	DefaultTyp   string         `gorm:"column:default_typ" json:"default_typ"`
	Author       AuthorList     `gorm:"column:author;serializer:json" json:"author" `
	Extra        string         `gorm:"column:extra;type:text" json:"extra"`           // 其它信息
	CreatedAt    time.Time      `gorm:"column:created_at;type:text" json:"created_at"` // 创建时间
	UpdatedAt    time.Time      `gorm:"column:updated_at" json:"updated_at"`           // 更新时间
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`           // 删除时间
}

func (*ABEntry) TableName() string {
	return "ab_entry"
}

func (t *ABEntry) Save() {
	DB().Save(t)

}

func (t *ABEntry) MultiGet() (tasks []*ABEntry) {
	tasks = []*ABEntry{}
	_ = DB().Model(t).Where(t).Order("id DESC").Scan(&tasks)
	return
}

func (t *ABEntry) Delete() {
	DB().Debug().Where(&ABEntry{ID: t.ID}).Delete(&ABEntry{})
}
