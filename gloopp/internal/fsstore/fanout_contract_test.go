package fsstore

import (
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestValidateFanoutContractRequiresLeafForOwnership(t *testing.T) {
	err := ValidateFanoutContract(FanoutContract{OwnershipScopes: []string{"internal/server"}})
	if err == nil || !strings.Contains(err.Error(), "leaf_id") {
		t.Fatalf("expected leaf_id error, got %v", err)
	}
}

func TestQuestStoreValidateFanoutSpawnConflicts(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	existing := &QuestMeta{
		ID:              "qst_leaf_a",
		ShortID:         "leaf_a",
		Query:           "leaf a",
		Type:            model.QuestTypeExecute,
		Status:          model.QuestStatusPending,
		CreatedBy:       "user",
		GroupID:         "grp_test",
		FanoutLeafID:    "leaf-a",
		OwnershipScopes: []string{"internal/server"},
		MergeStrategy:   MergeStrategyNoMerge,
	}
	if err := qs.CreateQuest(existing); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	err = qs.ValidateFanoutSpawn("grp_test", FanoutContract{
		LeafID:          "leaf-b",
		OwnershipScopes: []string{"internal/server"},
		MergeStrategy:   MergeStrategyNoMerge,
	})
	if err == nil || !strings.Contains(err.Error(), "ownership scope conflict") {
		t.Fatalf("expected ownership conflict, got %v", err)
	}
}

func TestQuestStoreValidateFanoutSpawnAllowsSharedMergeOwner(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	qs := NewQuestStore(root)
	existing := &QuestMeta{
		ID:               "qst_leaf_a",
		ShortID:          "leaf_a",
		Query:            "leaf a",
		Type:             model.QuestTypeExecute,
		Status:           model.QuestStatusPending,
		CreatedBy:        "user",
		GroupID:          "grp_test",
		FanoutLeafID:     "leaf-a",
		OwnershipScopes:  []string{"internal/server"},
		MergeStrategy:    MergeStrategySingleLeaf,
		MergeOwnerLeafID: "leaf-a",
	}
	if err := qs.CreateQuest(existing); err != nil {
		t.Fatalf("CreateQuest failed: %v", err)
	}

	err = qs.ValidateFanoutSpawn("grp_test", FanoutContract{
		LeafID:           "leaf-b",
		OwnershipScopes:  []string{"internal/server"},
		MergeStrategy:    MergeStrategySingleLeaf,
		MergeOwnerLeafID: "leaf-a",
	})
	if err != nil {
		t.Fatalf("expected shared merge owner to pass, got %v", err)
	}
}

func TestValidateFanoutBatchRejectsDuplicateLeafAndOwnershipConflict(t *testing.T) {
	err := ValidateFanoutBatch([]FanoutContract{
		{LeafID: "leaf-a"},
		{LeafID: "leaf-a"},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("expected duplicate leaf error, got %v", err)
	}

	err = ValidateFanoutBatch([]FanoutContract{
		{LeafID: "leaf-a", OwnershipScopes: []string{"internal/server"}, MergeStrategy: MergeStrategyNoMerge},
		{LeafID: "leaf-b", OwnershipScopes: []string{"internal/server"}, MergeStrategy: MergeStrategyNoMerge},
	})
	if err == nil || !strings.Contains(err.Error(), "ownership scope conflict") {
		t.Fatalf("expected ownership conflict, got %v", err)
	}

	err = ValidateFanoutBatch([]FanoutContract{
		{LeafID: "leaf-a", OwnershipScopes: []string{"internal/server"}, MergeStrategy: MergeStrategySingleLeaf, MergeOwnerLeafID: "leaf-a"},
		{LeafID: "leaf-b", OwnershipScopes: []string{"internal/server"}, MergeStrategy: MergeStrategySingleLeaf, MergeOwnerLeafID: "leaf-a"},
	})
	if err != nil {
		t.Fatalf("expected shared merge owner to pass, got %v", err)
	}
}
