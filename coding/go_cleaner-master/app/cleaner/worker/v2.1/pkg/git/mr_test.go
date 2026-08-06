package git

import (
	"fmt"
	"testing"

	"github.com/xanzy/go-gitlab"

	"code.byted.org/gopkg/lang/v2/conv"
)

func TestName(t *testing.T) {
	mr, err := CreateMR("xiaoxing.sn/demod_consumer", &gitlab.CreateMergeRequestOptions{
		Title:                conv.Ptr("chore: clean unused code"),
		Description:          nil,
		SourceBranch:         conv.Ptr("dev"),
		TargetBranch:         conv.Ptr("master"),
		Labels:               nil,
		AssigneeID:           nil,
		AssigneeIDs:          nil,
		ReviewerIDs:          nil,
		TargetProjectID:      nil,
		MilestoneID:          nil,
		RemoveSourceBranch:   nil,
		Squash:               nil,
		AllowCollaboration:   nil,
		ApprovalsBeforeMerge: nil,
	})

	fmt.Println(mr.WebURL, err)
}

func TestName2(t *testing.T) {
	mr, err := CreateCleanerMR(CleanerMR{
		Repo:                          "xiaoxing.sn/demod_consumer",
		SourceBranch:                  "dev",
		TargetBranch:                  "master",
		Reviewer:                      []string{"xiaoxing.sn"},
		DeletedEndpoint:               []string{"GetAAAA", "POSTXXXX"},
		VerifiedButNotLocatedEndpoint: []string{"GetVVVVV", "POSTVVVVV"},
		UnusedLoC:                     100,
		DownstreamRepoUnChecked:       map[string]bool{"x/x": true},
		DownstreamRepoChecked:         map[string]bool{"a/b": true},
	})

	fmt.Println(mr, err)
}
