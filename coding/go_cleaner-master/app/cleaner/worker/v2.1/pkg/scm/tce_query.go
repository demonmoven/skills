package scm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type SCMRepoInfo struct {
	ID            int    `json:"id"`
	SCMRepoID     string `json:"scm_repo_id"`
	RemoteID      int    `json:"remote_id"`
	Path          string `json:"path"`
	Name          string `json:"name"`
	MainRepo      bool   `json:"main_repo"`
	NeedNotify    bool   `json:"need_notify"`
	NeedCompile   string `json:"need_compile"`
	SCMBranch     string `json:"scm_branch"`
	Description   string `json:"description"`
	DisableNotify bool   `json:"disable_notify"`
}

type Build struct {
	Language         string         `json:"language"`
	TechStack        string         `json:"tech_stack"`
	BaseImage        string         `json:"base_image"`
	FrameworkSupport string         `json:"framework_support"`
	SCMRepoInfo      []*SCMRepoInfo `json:"scm_repo_info"`
	DependencyMode   string         `json:"dependency_mode"`
}

type Data struct {
	Meta struct {
		ID              int      `json:"id"`
		Psm             string   `json:"psm"`
		Name            string   `json:"name"`
		Env             string   `json:"env"`
		CreateTime      string   `json:"create_time"`
		UpdateTime      string   `json:"update_time"`
		// Tags            []string `json:"tags"`
		// TagsDisplay     []string `json:"tags_display"`
		Status          string   `json:"status"`
		StatusDisplay   string   `json:"status_display"`
		IsSubscribe     bool     `json:"is_subscribe"`
		TlbURL          string   `json:"tlb_url"`
		ServiceURL      string   `json:"service_url"`
		ServiceCloudURL string   `json:"service_cloud_url"`
		HostType        string   `json:"host_type"`
		IsSensitive     bool     `json:"is_sensitive"`
	} `json:"meta"`
	Build *Build `json:"build"`
}

type PageInfo struct {
	TotalCount int `json:"total_count"`
	PageNum    int `json:"page_num"`
	PageSize   int `json:"page_size"`
}

type TCEResponse struct {
	PageInfo PageInfo `json:"page_info"`
	Code     int      `json:"code"`
	Error    string   `json:"error"`
	Data     []Data   `json:"data"`
}

func getTCEMainScmRepoByPSM(psm string) (string, error) {
	client := &http.Client{}
	url := fmt.Sprintf("http://paas-gw.byted.org/open-apis/v1/services?psms=%s&env=prod", psm)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", ErrFailedToGetTCEInfoByPSM.With(err)
	}
	// 设置请求头
	req.Header.Set("Domain", "tce-prod")
	req.Header.Set("Authorization", "Bearer 870208efa16d9af2d3c2cce021b61c26")

	resp, err := client.Do(req)
	if err != nil {
		return "", ErrFailedToGetTCEInfoByPSM.With(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ErrFailedToGetTCEInfoByPSM.With(err)
	}

	var result TCEResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", ErrFailedToGetTCEInfoByPSM.With(err)
	}

	if len(result.Data) > 0 && result.Data[0].Build != nil && len(result.Data[0].Build.SCMRepoInfo) > 0 {
		for _, repo := range result.Data[0].Build.SCMRepoInfo {
			if repo.MainRepo {
				return repo.Name, nil
			}
		}
	}

	return "", ErrCompilerVersionNotFoundInImage
}
