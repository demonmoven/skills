package model

import (
	"testing"
)

func TestDB(t *testing.T) {
	err := DB().Raw("show tables").Row().Err()
	if err != nil {
		t.Error(err)
	}
}

func TestClearAll(t *testing.T) {
	err := DB().Raw("delete from ab_entry where id>0").Row().Err()
	if err != nil {
		t.Error(err)
	}
}
