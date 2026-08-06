package policy

import (
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestFactBuilderReviewFacts_ContextAutomation(t *testing.T) {
	root, err := fsstore.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := root.SaveAutomation(&fsstore.AutomationConfig{
		ID:                "auto_ctx",
		Name:              "ctx",
		Enabled:           true,
		Tags:              []string{"context"},
	}); err != nil {
		t.Fatal(err)
	}
	q := &fsstore.QuestMeta{
		ID:        "q1",
		Type:      model.QuestTypeExecute,
		CreatedBy: model.QuestSourcePrefix + "auto_ctx",
		Status:    model.QuestStatusReviewing,
	}
	_ = NewFactBuilder(root).BuildReviewFacts(q, "pass", 8)
}

func TestFactBuilderReviewFacts_WorkspaceDiff(t *testing.T) {
	q := &fsstore.QuestMeta{
		ID:               "q1",
		Type:             model.QuestTypeExecute,
		CreatedBy:        string(model.SourceUser),
		Status:           model.QuestStatusReviewing,
		DiffChangedFiles: 2,
	}
	got := NewFactBuilder(nil).BuildReviewFacts(q, "pass", 8)
	if got.EffectType != EffectWorkspaceDiff || !got.HasWorkspaceDiff || got.SideEffectLevel != "medium" {
		t.Fatalf("facts = %+v, want workspace diff", got)
	}
}
