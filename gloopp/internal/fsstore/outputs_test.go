package fsstore

import (
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestDeclaredOutput_AddAndDedup(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)

	q := &QuestMeta{
		ID:     "qst_test_outputs",
		Query:  "测试产出物",
		Type:   model.QuestTypeExecute,
		Status: model.QuestStatusPending,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	// 添加第一个声明型产出物
	art1, err := qs.AddDeclaredOutput(q.ID, DeclaredOutput{
		Name:   "飞书文档",
		Kind:   "document",
		URL:    "https://feishu.cn/doc/abc",
		Source: "warrior_phase",
	})
	if err != nil {
		t.Fatalf("AddDeclaredOutput failed: %v", err)
	}
	if art1.Name != "飞书文档" {
		t.Errorf("name mismatch: %s", art1.Name)
	}
	if art1.Kind != "document" {
		t.Errorf("kind mismatch: %s", art1.Kind)
	}
	if !strings.HasPrefix(art1.StoragePath, "https://") {
		t.Errorf("storage_path should be URL, got %s", art1.StoragePath)
	}
	if art1.Source != "warrior_phase" {
		t.Errorf("source mismatch: %s", art1.Source)
	}

	// 添加第二个
	_, err = qs.AddDeclaredOutput(q.ID, DeclaredOutput{
		Name:   "架构图",
		Kind:   "image",
		URL:    "https://feishu.cn/drive/xyz",
		Source: "warrior_phase",
	})
	if err != nil {
		t.Fatalf("AddDeclaredOutput 2 failed: %v", err)
	}

	// 验证数量
	q2, err := qs.LoadQuest(q.ID)
	if err != nil {
		t.Fatalf("LoadQuest failed: %v", err)
	}
	if len(q2.Outputs) != 2 {
		t.Errorf("expected 2 outputs, got %d", len(q2.Outputs))
	}

	// 去重：同名同 URL 不重复添加
	artDup, err := qs.AddDeclaredOutput(q.ID, DeclaredOutput{
		Name:   "飞书文档",
		Kind:   "document",
		URL:    "https://feishu.cn/doc/abc",
		Source: "warrior_phase",
	})
	if err != nil {
		t.Fatalf("AddDeclaredOutput dedup failed: %v", err)
	}
	if artDup.ID != art1.ID {
		t.Errorf("dedup should return same artifact: %s vs %s", artDup.ID, art1.ID)
	}
	q3, _ := qs.LoadQuest(q.ID)
	if len(q3.Outputs) != 2 {
		t.Errorf("after dedup, expected 2 outputs, got %d", len(q3.Outputs))
	}
}

func TestDeclaredOutput_ClearBySource(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)

	q := &QuestMeta{
		ID:     "qst_test_clear",
		Query:  "测试清空",
		Type:   model.QuestTypeExecute,
		Status: model.QuestStatusPending,
	}
	qs.CreateQuest(q)

	qs.AddDeclaredOutput(q.ID, DeclaredOutput{
		Name: "产出物1", Kind: "document", URL: "https://a.com", Source: "warrior_phase",
	})
	qs.AddDeclaredOutput(q.ID, DeclaredOutput{
		Name: "产出物2", Kind: "file", URL: "https://b.com", Source: "other_source",
	})

	// 清空 warrior_phase 来源
	if err := qs.ClearOutputs(q.ID, "warrior_phase"); err != nil {
		t.Fatalf("ClearOutputs failed: %v", err)
	}
	q2, _ := qs.LoadQuest(q.ID)
	if len(q2.Outputs) != 1 {
		t.Errorf("after clear warrior_phase, expected 1 output, got %d", len(q2.Outputs))
	}
	if q2.Outputs[0].Source != "other_source" {
		t.Errorf("remaining output should be other_source, got %s", q2.Outputs[0].Source)
	}

	// 清空所有
	if err := qs.ClearOutputs(q.ID, ""); err != nil {
		t.Fatalf("ClearOutputs(all) failed: %v", err)
	}
	q3, _ := qs.LoadQuest(q.ID)
	if len(q3.Outputs) != 0 {
		t.Errorf("after clear all, expected 0 outputs, got %d", len(q3.Outputs))
	}
}

func TestHasOnlyExternalOutputs(t *testing.T) {
	// 只有外部链接 → true
	q1 := &QuestMeta{
		Outputs: []QuestArtifact{
			{Name: "doc", StoragePath: "https://feishu.cn/doc/abc"},
			{Name: "img", StoragePath: "http://example.com/img.png"},
		},
	}
	if !hasOnlyExternalOutputsLocal(q1) {
		t.Error("expected hasOnlyExternalOutputs=true for only external outputs")
	}

	// 有本地文件 → false
	q2 := &QuestMeta{
		Outputs: []QuestArtifact{
			{Name: "doc", StoragePath: "https://feishu.cn/doc/abc"},
			{Name: "file", StoragePath: "artifacts/abc_test.txt"},
		},
	}
	if hasOnlyExternalOutputsLocal(q2) {
		t.Error("expected hasOnlyExternalOutputs=false when there's a local file")
	}

	// 没有产出物 → false
	q3 := &QuestMeta{}
	if hasOnlyExternalOutputsLocal(q3) {
		t.Error("expected hasOnlyExternalOutputs=false when no outputs")
	}
}

// 本地等价实现，用于验证逻辑（与 orchestrator 包的 hasOnlyExternalOutputs 保持一致）
func hasOnlyExternalOutputsLocal(q *QuestMeta) bool {
	if q == nil || len(q.Outputs) == 0 {
		return false
	}
	hasExternal := false
	hasFile := false
	for _, out := range q.Outputs {
		if strings.HasPrefix(out.StoragePath, "http://") || strings.HasPrefix(out.StoragePath, "https://") {
			hasExternal = true
		} else if out.StoragePath != "" {
			hasFile = true
		}
	}
	return hasExternal && !hasFile
}

// TestAdapterSavePreservesFsstoreOnlyFields 回归：domain round-trip 不得清空 fsstore 独有字段。
//
// 场景：剑士声明产出物后（fsstore 直接路径写入 Outputs），法师评审通过走 domain 状态机
// round-trip（MoveToUserReview/CompleteQuest 等）。若 adapter.Save 直接用 FromDomain 结果
// 覆盖，Outputs（domain 不认识）会被清空——委托完成后看不到做了啥。
// 回归：qst_2606253040 已登记 count=9 但 meta.outputs 为空。
func TestAdapterSavePreservesFsstoreOnlyFields(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	repo := NewQuestRepository(root)
	qs := NewQuestStore(root)

	qid := "qst_adapter_outputs"
	q := &QuestMeta{
		ID:     qid,
		Query:  "测试 adapter 保留 outputs",
		Type:   model.QuestTypeExecute,
		Status: model.QuestStatusPending,
	}
	if err := qs.CreateQuest(q); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	// fsstore 直接路径写入产物声明（模拟剑士 --deliverable）
	if _, err := qs.AddDeclaredOutput(qid, DeclaredOutput{
		Name:   "byteio 上报器",
		Kind:   "file",
		URL:    "metrics/byteio_reporter.go",
		Source: "warrior_phase",
	}); err != nil {
		t.Fatalf("AddDeclaredOutput failed: %v", err)
	}
	if _, err := qs.AddDeclaredOutput(qid, DeclaredOutput{
		Name:   "KR2 指标",
		Kind:   "document",
		URL:    "docs/kr2.md",
		Source: "warrior_phase",
	}); err != nil {
		t.Fatalf("AddDeclaredOutput 2 failed: %v", err)
	}

	// domain round-trip：Load → 改状态 → Save（模拟 MoveToUserReview 等状态机流转）
	d, err := repo.Load(qid)
	if err != nil {
		t.Fatalf("repo.Load failed: %v", err)
	}
	d.Status = model.QuestStatusUserReview // domain 改状态机字段
	if err := repo.Save(d); err != nil {
		t.Fatalf("repo.Save failed: %v", err)
	}

	// 验证 fsstore 独有字段未丢
	after, err := qs.LoadQuest(qid)
	if err != nil {
		t.Fatalf("LoadQuest after round-trip failed: %v", err)
	}
	if len(after.Outputs) != 2 {
		t.Fatalf("domain round-trip 清空了 Outputs: got %d, want 2", len(after.Outputs))
	}
	if after.Status != model.QuestStatusUserReview {
		t.Errorf("domain 状态字段未生效: status=%s", after.Status)
	}
}
