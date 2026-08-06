package model

import (
	"encoding/json"
	"fmt"
	"strings"
)

type FixerTask struct {
	PSM           *string `thrift:"PSM,1,optional" frugal:"1,optional,string" json:"PSM,omitempty"`             // 服务名
	ErrorId       *string `thrift:"ErrorID,2,required" frugal:"2,required,string" json:"ErrorID"`               // ErrorId
	PanicType     *string `thrift:"PanicType,3,required" frugal:"3,required,string" json:"PanicType"`           // panic类型
	Traceback     *string `thrift:"Traceback,4,required" frugal:"4,required,string" json:"Traceback"`           // 报错的堆栈信息
	GitUrl        *string `thrift:"GitUrl,5,required" frugal:"5,required,string" json:"GitUrl"`                 // 仓库的git链接
	GitBranch     *string `thrift:"GitBranch,6,required" frugal:"6,required,string" json:"GitBranch"`           // 报错的branch
	GitCommit     *string `thrift:"GitCommit,7,required" frugal:"7,required,string" json:"GitCommit"`           // 报错的commit
	RealFileName  *string `thrift:"RealFileName,8,required" frugal:"8,required,string" json:"RealFileName"`     // 报错的文件
	ErrCodeIndex  *int32  `thrift:"ErrCodeIndex,9,required" frugal:"9,required,i32" json:"ErrCodeIndex"`        // 报错的行号
	TraceBackLine *string `thrift:"TraceBackLine,10,required" frugal:"10,required,string" json:"TraceBackLine"` // 堆栈信息中报错行的文件的提示内容
}

func (f *FixerTask) GetRepoName() string {
	if f == nil || f.GitUrl == nil {
		return ""
	}

	urlParts := strings.SplitN(*f.GitUrl, ":", 2)
	path := urlParts[1]

	parts := strings.Split(path, "/")
	repoWithGit := fmt.Sprintf("%s/%s", parts[len(parts)-2], parts[len(parts)-1])
	return strings.TrimSuffix(repoWithGit, ".git")
}

func (f *FixerTask) String() string {
	data, _ := json.Marshal(f)
	return string(data)
}
