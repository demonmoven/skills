package scope

import (
	"encoding/json"
	"fmt"
	"time"
)

// ActionJSON is a copy of [cmd/go/internal/work.ActionJSON]
type ActionJSON struct {
	ID         int
	Mode       string
	Package    string
	Deps       []int     `json:",omitempty"`
	IgnoreFail bool      `json:",omitempty"`
	Args       []string  `json:",omitempty"`
	Link       bool      `json:",omitempty"`
	Objdir     string    `json:",omitempty"`
	Target     string    `json:",omitempty"`
	Priority   int       `json:",omitempty"`
	Failed     bool      `json:",omitempty"`
	Built      string    `json:",omitempty"`
	VetxOnly   bool      `json:",omitempty"`
	NeedVet    bool      `json:",omitempty"`
	NeedBuild  bool      `json:",omitempty"`
	ActionID   string    `json:",omitempty"`
	BuildID    string    `json:",omitempty"`
	TimeReady  time.Time `json:",omitempty"`
	TimeStart  time.Time `json:",omitempty"`
	TimeDone   time.Time `json:",omitempty"`
}

func parseActions(content []byte) ([]*ActionJSON, error) {
	results := make([]*ActionJSON, 0)
	err := json.Unmarshal(content, &results)
	if err != nil {
		return nil, fmt.Errorf("failed decode module: %v", err)
	} else {
		return results, nil
	}
}
