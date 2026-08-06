package fsstore

import (
	"fmt"
	"sort"
	"strings"
)

const (
	MergeStrategyNoMerge    = "no_merge"
	MergeStrategySingleLeaf = "single_leaf"
	MergeStrategySequential = "sequential"
)

type FanoutContract struct {
	LeafID           string   `json:"leaf_id,omitempty"`
	OwnershipScopes  []string `json:"ownership_scopes,omitempty"`
	MergeStrategy    string   `json:"merge_strategy,omitempty"`
	MergeOwnerLeafID string   `json:"merge_owner_leaf_id,omitempty"`
}

type FanoutContractError struct {
	Message string
}

func (e *FanoutContractError) Error() string { return e.Message }

func fanoutContractErrorf(format string, args ...any) error {
	return &FanoutContractError{Message: fmt.Sprintf(format, args...)}
}

func NormalizeFanoutContract(c FanoutContract) FanoutContract {
	c.LeafID = strings.TrimSpace(c.LeafID)
	c.MergeStrategy = strings.TrimSpace(c.MergeStrategy)
	c.MergeOwnerLeafID = strings.TrimSpace(c.MergeOwnerLeafID)
	c.OwnershipScopes = normalizeOwnershipScopes(c.OwnershipScopes)
	if c.MergeStrategy == "" && c.MergeOwnerLeafID == "" && len(c.OwnershipScopes) == 0 {
		return c
	}
	if c.MergeStrategy == "" {
		c.MergeStrategy = MergeStrategyNoMerge
	}
	return c
}

func normalizeOwnershipScopes(scopes []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(scopes))
	for _, raw := range scopes {
		scope := strings.TrimSpace(raw)
		if scope == "" || seen[scope] {
			continue
		}
		seen[scope] = true
		out = append(out, scope)
	}
	sort.Strings(out)
	return out
}

func ValidateFanoutContract(c FanoutContract) error {
	c = NormalizeFanoutContract(c)
	if (len(c.OwnershipScopes) > 0 || c.MergeStrategy != "") && c.LeafID == "" {
		return fanoutContractErrorf("leaf_id is required when ownership or merge contract is declared")
	}
	switch c.MergeStrategy {
	case "", MergeStrategyNoMerge:
		if c.MergeOwnerLeafID != "" {
			return fanoutContractErrorf("merge_owner_leaf_id requires merge_strategy single_leaf or sequential")
		}
	case MergeStrategySingleLeaf, MergeStrategySequential:
		if c.MergeOwnerLeafID == "" {
			return fanoutContractErrorf("merge_strategy %s requires merge_owner_leaf_id", c.MergeStrategy)
		}
	default:
		return fanoutContractErrorf("merge_strategy must be single_leaf | sequential | no_merge")
	}
	return nil
}

func (q *QuestMeta) FanoutContract() FanoutContract {
	if q == nil {
		return FanoutContract{}
	}
	return NormalizeFanoutContract(FanoutContract{
		LeafID:           q.FanoutLeafID,
		OwnershipScopes:  q.OwnershipScopes,
		MergeStrategy:    q.MergeStrategy,
		MergeOwnerLeafID: q.MergeOwnerLeafID,
	})
}

func (q *QuestMeta) ApplyFanoutContract(c FanoutContract) {
	c = NormalizeFanoutContract(c)
	q.FanoutLeafID = c.LeafID
	q.OwnershipScopes = append([]string(nil), c.OwnershipScopes...)
	q.MergeStrategy = c.MergeStrategy
	q.MergeOwnerLeafID = c.MergeOwnerLeafID
}

func (qs *QuestStore) ValidateFanoutSpawn(groupID string, contract FanoutContract) error {
	contract = NormalizeFanoutContract(contract)
	if err := ValidateFanoutContract(contract); err != nil {
		return err
	}
	if strings.TrimSpace(groupID) == "" || (contract.LeafID == "" && len(contract.OwnershipScopes) == 0) {
		return nil
	}
	items, err := qs.ListQuests()
	if err != nil {
		return err
	}
	incoming := scopeSet(contract.OwnershipScopes)
	for _, existing := range items {
		if existing == nil || existing.GroupID != groupID || existing.FanoutLeafID == "" {
			continue
		}
		if contract.LeafID != "" && existing.FanoutLeafID == contract.LeafID {
			return fanoutContractErrorf("fanout leaf_id %s already exists in group %s", contract.LeafID, groupID)
		}
		if len(contract.OwnershipScopes) == 0 {
			continue
		}
		if !hasScopeOverlap(incoming, existing.OwnershipScopes) {
			continue
		}
		if fanoutContractsShareMergeOwner(contract, existing.FanoutContract()) {
			continue
		}
		return fanoutContractErrorf("ownership scope conflict in group %s between leaf %s and %s; declare a shared merge_owner_leaf_id", groupID, existing.FanoutLeafID, contract.LeafID)
	}
	return nil
}

func ValidateFanoutBatch(contracts []FanoutContract) error {
	seenLeaf := map[string]bool{}
	seen := make([]FanoutContract, 0, len(contracts))
	for _, raw := range contracts {
		contract := NormalizeFanoutContract(raw)
		if err := ValidateFanoutContract(contract); err != nil {
			return err
		}
		if contract.LeafID != "" {
			if seenLeaf[contract.LeafID] {
				return fanoutContractErrorf("fanout leaf_id %s is duplicated", contract.LeafID)
			}
			seenLeaf[contract.LeafID] = true
		}
		incoming := scopeSet(contract.OwnershipScopes)
		if len(incoming) > 0 {
			for _, existing := range seen {
				if !hasScopeOverlap(incoming, existing.OwnershipScopes) {
					continue
				}
				if fanoutContractsShareMergeOwner(contract, existing) {
					continue
				}
				return fanoutContractErrorf("ownership scope conflict between leaf %s and %s; declare a shared merge_owner_leaf_id", existing.LeafID, contract.LeafID)
			}
		}
		seen = append(seen, contract)
	}
	return nil
}

func scopeSet(scopes []string) map[string]bool {
	out := map[string]bool{}
	for _, scope := range scopes {
		out[scope] = true
	}
	return out
}

func hasScopeOverlap(left map[string]bool, right []string) bool {
	for _, scope := range right {
		if left[scope] {
			return true
		}
	}
	return false
}

func fanoutContractsShareMergeOwner(a, b FanoutContract) bool {
	a = NormalizeFanoutContract(a)
	b = NormalizeFanoutContract(b)
	if a.MergeOwnerLeafID == "" || b.MergeOwnerLeafID == "" {
		return false
	}
	return a.MergeOwnerLeafID == b.MergeOwnerLeafID &&
		a.MergeStrategy != MergeStrategyNoMerge &&
		b.MergeStrategy != MergeStrategyNoMerge
}
