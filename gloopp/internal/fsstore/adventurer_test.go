package fsstore

import (
	"os"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestActivateAdventurer(t *testing.T) {
	dir, err := os.MkdirTemp("", "gloop_test_adventurer_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	root, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	// 创建一个 pending_setup 状态的冒险者
	adv := &AdventurerFile{
		ID:     "adv_test_001",
		Name:   "测试剑士",
		Class:  model.ClassWarrior,
		Status: model.AdventurerPendingSetup,
		Level:  1,
		Exp:    0,
		Agent:  "mock",
	}
	if err := root.SaveAdventurer(adv); err != nil {
		t.Fatal(err)
	}

	// 激活
	a, err := root.ActivateAdventurer("adv_test_001", nil)
	if err != nil {
		t.Fatalf("激活失败: %v", err)
	}
	if a.Status != model.AdventurerActive {
		t.Errorf("激活后状态应为 active，实际 %s", a.Status)
	}

	// 幂等：再次激活不报错
	a2, err := root.ActivateAdventurer("adv_test_001", nil)
	if err != nil {
		t.Fatalf("二次激活失败（应该幂等）: %v", err)
	}
	if a2.Status != model.AdventurerActive {
		t.Errorf("二次激活状态应为 active")
	}
}

func TestActivateAdventurer_WithUpdates(t *testing.T) {
	dir, err := os.MkdirTemp("", "gloop_test_adventurer_upd_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	root, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	adv := &AdventurerFile{
		ID:          "adv_test_002",
		Name:        "原名",
		Class:       model.ClassMage,
		Status:      model.AdventurerPendingSetup,
		Level:       1,
		Agent:       "old_agent",
		Description: "原描述",
	}
	if err := root.SaveAdventurer(adv); err != nil {
		t.Fatal(err)
	}

	// 激活并更新字段
	newName := "新名字"
	newDesc := "新描述"
	updates := &AdventurerUpdate{
		Name:        newName,
		Agent:       "new_agent",
		Description: &newDesc,
		Tools:       []string{"bash", "read"},
	}
	a, err := root.ActivateAdventurer("adv_test_002", updates)
	if err != nil {
		t.Fatalf("激活失败: %v", err)
	}

	if a.Name != newName {
		t.Errorf("name = %q, want %q", a.Name, newName)
	}
	if a.Agent != "new_agent" {
		t.Errorf("agent = %q, want new_agent", a.Agent)
	}
	if a.Description != newDesc {
		t.Errorf("description = %q, want %q", a.Description, newDesc)
	}
	if len(a.Tools) != 2 || a.Tools[0] != "bash" {
		t.Errorf("tools = %v, want [bash read]", a.Tools)
	}
	if a.Status != model.AdventurerActive {
		t.Errorf("status = %s, want active", a.Status)
	}
}

func TestActivateAdventurer_InvalidStatus(t *testing.T) {
	dir, err := os.MkdirTemp("", "gloop_test_adventurer_ret_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	root, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	adv := &AdventurerFile{
		ID:     "adv_test_003",
		Name:   "退休法师",
		Class:  model.ClassMage,
		Status: model.AdventurerRetired,
		Level:  5,
	}
	if err := root.SaveAdventurer(adv); err != nil {
		t.Fatal(err)
	}

	_, err = root.ActivateAdventurer("adv_test_003", nil)
	if err == nil {
		t.Error("retired 状态的冒险者应该不能激活")
	}
}

func TestActivateAdventurer_NotFound(t *testing.T) {
	dir, err := os.MkdirTemp("", "gloop_test_adventurer_nf_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	root, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = root.ActivateAdventurer("nonexistent", nil)
	if err == nil {
		t.Error("不存在的冒险者应该报错")
	}
}

func TestFindByClass_RanksByWinRateThenLevel(t *testing.T) {
	dir, err := os.MkdirTemp("", "gloop_test_adventurer_rank_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	root, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	advs := []*AdventurerFile{
		{ID: "low_win_high_level", Name: "高等级低胜率", Class: model.ClassWarrior, Status: model.AdventurerActive, Level: 9, WinCount: 1, LoseCount: 9, CreatedAtMs: 2},
		{ID: "high_win_low_level", Name: "低等级高胜率", Class: model.ClassWarrior, Status: model.AdventurerActive, Level: 2, WinCount: 8, LoseCount: 2, CreatedAtMs: 1},
		{ID: "mage", Name: "法师", Class: model.ClassMage, Status: model.AdventurerActive, Level: 10},
	}
	for _, adv := range advs {
		if err := root.SaveAdventurer(adv); err != nil {
			t.Fatal(err)
		}
	}
	pick, err := root.FindByClass(model.ClassWarrior)
	if err != nil {
		t.Fatal(err)
	}
	if pick.ID != "high_win_low_level" {
		t.Fatalf("pick = %s, want high_win_low_level", pick.ID)
	}
}
