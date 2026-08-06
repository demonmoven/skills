package scm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"code.byted.org/lang/gg/gslice"
)

type RepoInfo struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	HttpUrl            string `json:"http_url"`
	GitUrl             string `json:"git_url"`
	CreateUser         string `json:"create_user"`
	Desc               string `json:"desc"`
	Type               string `json:"type"`
	ParallelConcurrent int    `json:"parallel_concurrent"`
	ProxySwitch        bool   `json:"proxy_switch"`
	GitSource          string `json:"git_source"`
	RepoName           string `json:"repo_name"`
	FavorUsers         []int  `json:"favor_users"`
	Image              string `json:"image"`
	CompileScriptPath  string `json:"compile_script_path"`
}

type SCMResponse struct {
	Data []*RepoInfo `json:"data"`
}

func getCompilerVersionFromSCMBySCMRepoName(repoName string, extractFunc func(string) string) (string, error) {
	url := fmt.Sprintf("https://scm.byted.org/api/v2/repos/by_names?repo_names=%s", repoName)
	resp, err := http.Get(url)
	if err != nil {
		return "", ErrDataNotFoundInResponse.With(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ErrDataNotFoundInResponse.With(err)
	}

	var result []RepoInfo
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", ErrDataNotFoundInResponse.With(err)
	}
	if len(result) == 0 {
		return "", ErrDataNotFoundInResponse
	}

	buildImage := result[0].Image

	version := extractFunc(buildImage)
	if version != "" {
		return version, nil
	}
	return "", ErrCompilerVersionNotFoundInImage
}

func getSCMResponseByGitName(gitName string) (SCMResponse, error) {
	var result SCMResponse

	url := fmt.Sprintf("https://scm.byted.org/api/v2/repos/by_git_name/?name=%s", gitName)
	resp, err := http.Get(url)
	if err != nil {
		return result, ErrDataNotFoundInResponse.With(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, ErrDataNotFoundInResponse.With(err)
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return result, ErrDataNotFoundInResponse.With(err)
	}
	if len(result.Data) == 0 {
		return result, ErrDataNotFoundInResponse
	}

	return result, nil
}

func getBuildScriptFromSCMByGitName(gitName string) (string, error) {
	result, err := getSCMResponseByGitName(gitName)
	if err != nil {
		return "", err
	}

	var scripts []string

	for _, repo := range result.Data {
		scripts = append(scripts, repo.CompileScriptPath)
	}

	defaultScript := "build.sh"

	if gslice.Contains(scripts, "build.sh") {
		return defaultScript, nil
	}

	for _, s := range scripts {
		if !strings.Contains(s, "run.sh") {
			return s, nil
		}
	}

	return defaultScript, nil
}

func getCompilerVersionFromSCMByGitName(gitName string, extractFunc func(string) string) (string, error) {
	result, err := getSCMResponseByGitName(gitName)
	if err != nil {
		return "", err
	}

	// 仓库和SCM是一对多的关系，下面取这些SCM中的编译器最高版本号
	var versions []string
	for _, repo := range result.Data {
		buildImage := repo.Image

		// "1.xx" 格式版本号
		version := extractFunc(buildImage)
		if version != "" {
			versions = append(versions, version)
		}
	}
	if len(versions) == 0 {
		return "", ErrCompilerVersionNotFoundInImage
	}

	sort.Strings(versions)

	return versions[len(versions)-1], nil
}

func extractGoVersion(s string) string {
	// 定义匹配 1.xx 或 1_xx（xx为数字）格式的正则表达式
	pattern := `1[._]\d+`
	reg := regexp.MustCompile(pattern)
	match := reg.FindString(s)

	if match != "" {
		match = strings.ReplaceAll(match, "_", ".")
	}

	return match
}

func extractCxxVersion(s string) string {
	defaultLLVMVersion := "llvm16"

	prefixes := []string{
		"buster_llvm",
		"toutiao.compile_cpp",
		"buster_cpp",
		"toutiao.aml.compile_cuda",
		"compile_spider",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return defaultLLVMVersion
		}
	}

	// 定义匹配 llvmxx（xx为数字）格式的正则表达式
	pattern := `llvm\d+`
	reg := regexp.MustCompile(pattern)
	return reg.FindString(s)
}
