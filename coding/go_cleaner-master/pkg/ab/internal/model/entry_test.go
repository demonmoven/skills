package model

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestSubmit(t *testing.T) {
	task := &ABEntry{
		ID:      0,
		Package: "code.byted.org/iesarch/abtest/ab",
		Recv:    "AbTest",
		Func: []string{
			"GetBoolV",
			"GetIntV",
			"GetStringV",
		},
		KeyIndex:     []int{1},
		DefaultIndex: 2,
		KeyPrefix:    "",
		DefaultVal:   "",
		DefaultTyp:   "",
		Author:       []string{"xiaoxing.sn"},
		Extra:        "",
		CreatedAt:    time.Time{},
		UpdatedAt:    time.Time{},
		DeletedAt:    gorm.DeletedAt{},
	}
	task.Save()
	tasks := (&ABEntry{}).MultiGet()
	fmt.Println(*tasks[0])
	task = tasks[0]
	task.Author.Add("x")
	task.Save()
	tasks = (&ABEntry{}).MultiGet()
	fmt.Println(*tasks[0])
	tasks[0].Delete()
	tasks = (&ABEntry{}).MultiGet()
	fmt.Println(tasks)
}
