package git

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xanzy/go-gitlab"

	"code.byted.org/gopkg/env"
	"code.byted.org/gopkg/lang/v2/conv"
	"code.byted.org/gopkg/logs/v2"
	"code.byted.org/live/wcc_gen_code/webcast_corelink_faas"
)

var ciIesOpsClient = func() *gitlab.Client {
	if env.IsBoe() {
		return nil
	}
	token := os.Getenv("TOKEN_CI_IESOPS")
	if token == "" {
		var err error
		token, err = webcast_corelink_faas.GetAssertCodebaseTokenCiIesops()
		if err != nil {
			logs.Error("GetAssertCodebaseTokenCiIesops fail, err=%+v", err)
			panic(err)
		}
	}

	ciIesOpsClient, err := gitlab.NewClient(token, gitlab.WithBaseURL("https://code.byted.org/api/v4"))
	if err != nil {
		logs.Error("ciIesOpsClient create fail, err=%+v", err)
		panic(err)
	}
	logs.Info("ciIesOpsClient init success")

	return ciIesOpsClient
}()

func CreateMR(repoName string, mrOption *gitlab.CreateMergeRequestOptions) (*gitlab.MergeRequest, error) {
	if mrOption == nil {
		mrOption = &gitlab.CreateMergeRequestOptions{}
	}

	if mrOption.Title == nil {
		mrOption.Title = conv.Ptr("go cleaner generated MR")
	}

	mr, rsp, err := ciIesOpsClient.MergeRequests.CreateMergeRequest(repoName, mrOption)

	if err != nil && !strings.Contains(err.Error(), "Cannot Create: This merge request already exists") {
		return nil, fmt.Errorf("create merge request failed, rsp: %v, err: %v", rsp, err)
	}

	// 获取MR的ID：因为可能之前已有MR，所以本次可能并没有创建成功
	time.Sleep(3 * time.Second) // gitlab拿到commit hash有延迟，需要等待一段时间，否则会一直没办法跳过

	opt := &gitlab.ListProjectMergeRequestsOptions{
		State:        conv.Ptr("opened"),
		SourceBranch: mrOption.SourceBranch,
		TargetBranch: mrOption.TargetBranch,
	}
	mrs, _, err := ciIesOpsClient.MergeRequests.ListProjectMergeRequests(repoName, opt)
	if err != nil {
		return nil, fmt.Errorf("service.Client().MergeRequests.ListProjectMergeRequests(repoName=%v, opt) failed: %v", repoName, err)
	}
	flag := false
	for _, m := range mrs {
		if m.SourceBranch == *mrOption.SourceBranch {
			mr = m
			flag = true
			break
		}
	}
	if !flag || mr == nil {
		return nil, fmt.Errorf("find old MR (repoName=%v) failed: %v", repoName, err)
	}

	mr, _, err = ciIesOpsClient.MergeRequests.UpdateMergeRequest(repoName, mr.IID, &gitlab.UpdateMergeRequestOptions{
		Title:        mrOption.Title,
		Description:  mrOption.Description,
		TargetBranch: mrOption.TargetBranch,
		AssigneeID:   mrOption.AssigneeID,
		AssigneeIDs:  mrOption.AssigneeIDs,
		ReviewerIDs:  mrOption.ReviewerIDs,
	})

	return mr, err
}
