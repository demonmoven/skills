package fsstore

import (
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

func TestApplyDefaults_CommandAllowlist_MergesByID(t *testing.T) {
	// 模拟默认白名单（简化版，只放关键几条）
	def := &GlobalConfig{
		CommandAllowlist: []AllowedCommand{
			{ID: "go-test", Command: "go", SideEffectLevel: "L0"},
			{ID: "go-vet", Command: "go", SideEffectLevel: "L0"},
			{ID: "git-diff-stat", Command: "git", SideEffectLevel: "L0"},
		},
	}

	// 用户配置只有 3 条老命令（go-test 参数覆盖 + 1 条自定义）
	dst := &GlobalConfig{
		CommandAllowlist: []AllowedCommand{
			{ID: "go-test", Command: "go", TimeoutMs: 9999}, // 同 ID，覆盖默认参数
			{ID: "my-custom", Command: "custom"},            // 用户自定义，默认中没有
		},
	}

	applyDefaults(dst, def)

	// 合并后应该有 4 条：go-test(用户版) + go-vet(默认) + git-diff-stat(默认) + my-custom(用户)
	if len(dst.CommandAllowlist) != 4 {
		t.Fatalf("expected 4 commands after merge, got %d", len(dst.CommandAllowlist))
	}

	// 验证用户的 go-test 保留了自定义 timeout
	gt, ok := FindAllowedCommand(dst.CommandAllowlist, "go-test")
	if !ok {
		t.Fatal("go-test should exist")
	}
	if gt.TimeoutMs != 9999 {
		t.Fatalf("go-test timeout should be 9999 (user override), got %d", gt.TimeoutMs)
	}

	// 验证默认中有的、用户没配的命令被补入
	if _, ok := FindAllowedCommand(dst.CommandAllowlist, "go-vet"); !ok {
		t.Fatal("go-vet (default) should be merged in")
	}
	if _, ok := FindAllowedCommand(dst.CommandAllowlist, "git-diff-stat"); !ok {
		t.Fatal("git-diff-stat (default) should be merged in")
	}

	// 验证用户自定义命令保留
	if _, ok := FindAllowedCommand(dst.CommandAllowlist, "my-custom"); !ok {
		t.Fatal("my-custom (user-defined) should be preserved")
	}

	// 验证默认值仍然被正确填充（用户自定义命令缺 side_effect_level → L1）
	mc, _ := FindAllowedCommand(dst.CommandAllowlist, "my-custom")
	if mc.SideEffectLevel != "L1" {
		t.Fatalf("my-custom side_effect_level should default to L1, got %q", mc.SideEffectLevel)
	}
}

func TestApplyDefaults_CommandAllowlist_NilUserConfig(t *testing.T) {
	def := &GlobalConfig{
		CommandAllowlist: []AllowedCommand{
			{ID: "go-test", Command: "go"},
		},
	}
	dst := &GlobalConfig{} // CommandAllowlist 为 nil

	applyDefaults(dst, def)

	if len(dst.CommandAllowlist) != 1 {
		t.Fatalf("expected 1 default command, got %d", len(dst.CommandAllowlist))
	}
}

func TestApplyDefaults_BudgetMigration_v0_2_12(t *testing.T) {
	def := DefaultGlobalConfig()

	// 老 config（version 0.2.5）回合/时长过紧，应向上提升到下限
	dst := &GlobalConfig{
		Version:                   "0.2.5",
		MaxTurnsPerPhase:          6,
		MaxConsecutiveAgentErrors: 2,
	}
	applyDefaults(dst, def)
	if dst.MaxTurnsPerPhase != 20 {
		t.Fatalf("MaxTurnsPerPhase 应迁移到 20，实际 %d", dst.MaxTurnsPerPhase)
	}
	if dst.MaxConsecutiveAgentErrors != 3 {
		t.Fatalf("MaxConsecutiveAgentErrors 应迁移到 3，实际 %d", dst.MaxConsecutiveAgentErrors)
	}

	// 新 config（version 0.2.12）用户显式设的合理值不应被覆盖
	dst2 := &GlobalConfig{
		Version:          "0.2.12",
		MaxTurnsPerPhase: 8,
	}
	applyDefaults(dst2, def)
	if dst2.MaxTurnsPerPhase != 8 {
		t.Fatalf("0.2.12 用户显式设的 8 不应被覆盖，实际 %d", dst2.MaxTurnsPerPhase)
	}

	// 已是合理值的旧 config 不动
	dst3 := &GlobalConfig{
		Version:          "0.2.5",
		MaxTurnsPerPhase: 50,
	}
	applyDefaults(dst3, def)
	if dst3.MaxTurnsPerPhase != 50 {
		t.Fatalf("合理值 50 不应被迁移，实际 %d", dst3.MaxTurnsPerPhase)
	}
}

func TestGlobalConfigPatch_ContextFileMode(t *testing.T) {
	base := DefaultGlobalConfig()
	if base.ContextFileMode {
		t.Fatal("context file mode should default to false")
	}
	enabled := true
	got := (&GlobalConfigPatch{ContextFileMode: &enabled}).Apply(base)
	if !got.ContextFileMode {
		t.Fatal("ContextFileMode patch should enable file context mode")
	}
}

func TestGlobalConfig_HotlAutoCloseDefaultAndPatch(t *testing.T) {
	base := DefaultGlobalConfig()
	if base.HotlAutoClose == nil || !*base.HotlAutoClose {
		t.Fatalf("HOTL auto-close should default to enabled, got %#v", base.HotlAutoClose)
	}

	disabled := false
	got := (&GlobalConfigPatch{HotlAutoClose: &disabled}).Apply(base)
	if got.HotlAutoClose == nil || *got.HotlAutoClose {
		t.Fatalf("HotlAutoClose patch should preserve explicit false, got %#v", got.HotlAutoClose)
	}

	legacy := &GlobalConfig{Version: "0.1.9"}
	applyDefaults(legacy, base)
	if legacy.HotlAutoClose == nil || !*legacy.HotlAutoClose {
		t.Fatalf("legacy config missing hotl_auto_close should migrate to true, got %#v", legacy.HotlAutoClose)
	}

	explicitFalse := &GlobalConfig{Version: version.Version, HotlAutoClose: &disabled}
	applyDefaults(explicitFalse, base)
	if explicitFalse.HotlAutoClose == nil || *explicitFalse.HotlAutoClose {
		t.Fatalf("new config explicit false must not be overwritten, got %#v", explicitFalse.HotlAutoClose)
	}
}
