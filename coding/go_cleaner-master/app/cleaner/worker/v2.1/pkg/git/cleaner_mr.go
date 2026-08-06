package git

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"text/template"

	"github.com/xanzy/go-gitlab"

	"code.byted.org/gopkg/lang/v2/conv"
)

type CleanerMR struct {
	TaskURL      string
	Repo         string
	SourceBranch string
	TargetBranch string
	Reviewer     []string
	ReviewerID   []int

	DeletedEndpoint               []string
	VerifiedButNotLocatedEndpoint []string

	UnusedLoC int

	MeegoID string

	DownstreamRepoUnChecked map[string]bool
	DownstreamRepoChecked   map[string]bool
}

//go:embed mr_description.md
var descriptionTemplate string

var descriptionTpl = func() *template.Template {
	v, err := template.New("description").Parse(descriptionTemplate)
	if err != nil {
		panic(err)
	}

	return v
}()

func CreateCleanerMR(mr CleanerMR) (url string, err error) {
	buf := bytes.Buffer{}

	err = descriptionTpl.Execute(&buf, mr)
	if err != nil {
		return "", err
	}

	resp, err := CreateMR(mr.Repo, &gitlab.CreateMergeRequestOptions{
		Title:                conv.Ptr(fmt.Sprintf("chore: [冗余资产平台]clean unused code %s", mr.MeegoID)),
		Description:          conv.Ptr(buf.String()),
		SourceBranch:         conv.Ptr(mr.SourceBranch),
		TargetBranch:         conv.Ptr(mr.TargetBranch),
		Labels:               nil,
		AssigneeID:           nil,
		AssigneeIDs:          nil,
		ReviewerIDs:          &mr.ReviewerID,
		TargetProjectID:      nil,
		MilestoneID:          nil,
		RemoveSourceBranch:   nil,
		Squash:               nil,
		AllowCollaboration:   nil,
		ApprovalsBeforeMerge: nil,
	})

	if err != nil {
		return "", err
	}

	if resp == nil {
		return "", errors.New("nil mr")
	}

	return resp.WebURL, nil
}

func CreateCustomMR(mr CleanerMR, title string, description string) (url string, err error) {
	resp, err := CreateMR(mr.Repo, &gitlab.CreateMergeRequestOptions{
		Title:                conv.Ptr(title),
		Description:          conv.Ptr(description),
		SourceBranch:         conv.Ptr(mr.SourceBranch),
		TargetBranch:         conv.Ptr(mr.TargetBranch),
		Labels:               nil,
		AssigneeID:           nil,
		AssigneeIDs:          nil,
		ReviewerIDs:          &mr.ReviewerID,
		TargetProjectID:      nil,
		MilestoneID:          nil,
		RemoveSourceBranch:   nil,
		Squash:               nil,
		AllowCollaboration:   nil,
		ApprovalsBeforeMerge: nil,
	})

	if err != nil {
		return "", err
	}

	if resp == nil {
		return "", errors.New("nil mr")
	}

	return resp.WebURL, nil
}
