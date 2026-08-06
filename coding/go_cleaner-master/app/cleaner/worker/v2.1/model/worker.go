package model

type AcquireCodeCleanTaskRequest struct {
	TaskOpts []string `json:"task_opts"`

	// [obselete] ['1.18','1.19', ..]
	GoVersion []string `thrift:"GoVersion,1,optional" frugal:"1,optional,list<string>" json:"GoVersion,omitempty"`
	// 发起请求的worker
	Worker *string `thrift:"Worker,2,optional" frugal:"2,optional,string" json:"Worker,omitempty"`
	// 机器状态
	MachineStat *string `thrift:"MachineStat,3,optional" frugal:"3,optional,string" json:"MachineStat,omitempty"`
}

func (p *AcquireCodeCleanTaskRequest) GetGoVersion() []string {
	if p != nil {
		return p.GoVersion
	}
	return nil
}
func (p *AcquireCodeCleanTaskRequest) GetWorker() string {
	if p != nil && p.Worker != nil {
		return *p.Worker
	}
	return ""
}
func (p *AcquireCodeCleanTaskRequest) GetMachineStat() string {
	if p != nil && p.MachineStat != nil {
		return *p.MachineStat
	}
	return ""
}

// 废弃代码清理任务的增删查接口
type CodeCleanTaskExtra struct {
	PSM          *string `thrift:"PSM,1,optional" frugal:"1,optional,string" json:"PSM,omitempty"`
	RepoName     *string `thrift:"RepoName,2,optional" frugal:"2,optional,string" json:"RepoName,omitempty"`
	SubDir       *string `thrift:"SubDir,3,optional" frugal:"3,optional,string" json:"SubDir,omitempty"`
	Branch       *string `thrift:"Branch,4,optional" frugal:"4,optional,string" json:"Branch,omitempty"`
	CommitBranch *string `thrift:"CommitBranch,5,optional" frugal:"5,optional,string" json:"CommitBranch,omitempty"`
	// 清理工具执行参数
	CleanerFlag *string `thrift:"CleanerFlag,6,optional" frugal:"6,optional,string" json:"CleanerFlag,omitempty"`
	// 无用接口
	Endpoint []string `thrift:"Endpoint,7,optional" frugal:"7,optional,list<string>" json:"Endpoint,omitempty"`
	// MR Reviewer
	Reviewer []string `thrift:"Reviewer,8,optional" frugal:"8,optional,list<string>" json:"Reviewer,omitempty"`
	Timeout  *int64   `thrift:"Timeout,10,optional" frugal:"10,optional,i64" json:"Timeout,omitempty"`
	// 提交人
	Author *string `thrift:"Author,11,optional" frugal:"11,optional,string" json:"Author,omitempty"`
	//成功删除的无用接口
	DeletedEndpoint []string `thrift:"DeletedEndpoint,12,optional" frugal:"12,optional,list<string>" json:"DeletedEndpoint,omitempty"`
	// 被依赖的仓库
	DownstreamRepo []string `thrift:"DownstreamRepo,13,optional" frugal:"13,optional,list<string>" json:"DownstreamRepo,omitempty"`
	// 已经检查了的被依赖的仓库
	DownstreamRepoChecked []string `thrift:"DownstreamRepoChecked,14,optional" frugal:"14,optional,list<string>" json:"DownstreamRepoChecked,omitempty"`
	// 标注保留行信息
	AnnotateRetainedCode []string `thrift:"AnnotateRetainedCode,15,optional" frugal:"15,optional,list<string>" json:"AnnotateRetainedCode,omitempty"`
	// 指定的基准commit
	AssignedBaseCommit *string `thrift:"AssignedBaseCommit,16,optional" frugal:"16,optional,string" json:"AssignedBaseCommit,omitempty"`
	// main-pkg所在目录
	MainDir *string `thrift:"MainDir,17,optional" frugal:"17,optional,string" json:"MainDir,omitempty"`
	// 预处理脚本
	PreScript *string `thrift:"PreScript,18,optional" frugal:"18,optional,string" json:"PreScript,omitempty"`
	// 仓库下的所有apps的main-pkg目录，逗号分隔
	RepoAppMains *string `thrift:"RepoAppMains,19,optional" frugal:"19,optional,string" json:"RepoAppMains,omitempty"`
	// 近期构建用的go版本
	RecentBuildGoVersion *string `thrift:"RecentBuildGoVersion,20,optional" frugal:"20,optional,string" json:"RecentBuildGoVersion,omitempty"`
	// SCM上的go版本
	SCMGoVersion *string `thrift:"SCMGoVersion,21,optional" frugal:"21,optional,string" json:"SCMGoVersion,omitempty"`
	// 创建MR所需最少变更行数
	CreateMRIfLineChangeGreaterThan *int32 `thrift:"CreateMRIfLineChangeGreaterThan,22,optional" frugal:"22,optional,i32" json:"CreateMRIfLineChangeGreaterThan,omitempty"`
	// 近期编译时间
	RecentBuildTimeUnix *int64 `thrift:"RecentBuildTimeUnix,23,optional" frugal:"23,optional,i64" json:"RecentBuildTimeUnix,omitempty"`

	SkipCreateMR bool `json:"skip_create_mr"`
}

func (p *CodeCleanTaskExtra) GetPSM() string {
	if p != nil && p.PSM != nil {
		return *p.PSM
	}
	return ""
}
func (p *CodeCleanTaskExtra) GetRepoName() string {
	if p != nil && p.RepoName != nil {
		return *p.RepoName
	}
	return ""
}

func (p *CodeCleanTaskExtra) GetSubDir() string {
	if p != nil && p.SubDir != nil {
		return *p.SubDir
	}
	return ""
}

func (p *CodeCleanTaskExtra) GetBranch() string {
	if p != nil && p.Branch != nil {
		return *p.Branch
	}
	return ""
}

func (p *CodeCleanTaskExtra) GetCommitBranch() string {
	if p != nil && p.CommitBranch != nil {
		return *p.CommitBranch
	}
	return ""
}

func (p *CodeCleanTaskExtra) GetCleanerFlag() string {
	if p != nil && p.CleanerFlag != nil {
		return *p.CleanerFlag
	}
	return ""
}

func (p *CodeCleanTaskExtra) GetEndpoint() []string {
	if p != nil {
		return p.Endpoint
	}
	return nil
}
func (p *CodeCleanTaskExtra) GetReviewer() []string {
	if p != nil {
		return p.Reviewer
	}
	return nil
}
func (p *CodeCleanTaskExtra) GetTimeout() int64 {
	if p != nil && p.Timeout != nil {
		return *p.Timeout
	}
	return 0
}
func (p *CodeCleanTaskExtra) GetAuthor() string {
	if p != nil && p.Author != nil {
		return *p.Author
	}
	return ""
}
func (p *CodeCleanTaskExtra) GetDeletedEndpoint() []string {
	if p != nil {
		return p.DeletedEndpoint
	}
	return nil
}
func (p *CodeCleanTaskExtra) GetDownstreamRepo() []string {
	if p != nil {
		return p.DownstreamRepo
	}
	return nil
}
func (p *CodeCleanTaskExtra) GetDownstreamRepoChecked() []string {
	if p != nil {
		return p.DownstreamRepoChecked
	}
	return nil
}
func (p *CodeCleanTaskExtra) GetAnnotateRetainedCode() []string {
	if p != nil {
		return p.AnnotateRetainedCode
	}
	return nil
}
func (p *CodeCleanTaskExtra) GetAssignedBaseCommit() string {
	if p != nil && p.AssignedBaseCommit != nil {
		return *p.AssignedBaseCommit
	}
	return ""
}
func (p *CodeCleanTaskExtra) GetMainDir() string {
	if p != nil && p.MainDir != nil {
		return *p.MainDir
	}
	return ""
}
func (p *CodeCleanTaskExtra) GetPreScript() string {
	if p != nil && p.PreScript != nil {
		return *p.PreScript
	}
	return ""
}
func (p *CodeCleanTaskExtra) GetRepoAppMains() string {
	if p != nil && p.RepoAppMains != nil {
		return *p.RepoAppMains
	}
	return ""
}
func (p *CodeCleanTaskExtra) GetRecentBuildGoVersion() string {
	if p != nil && p.RecentBuildGoVersion != nil {
		return *p.RecentBuildGoVersion
	}
	return ""
}
func (p *CodeCleanTaskExtra) GetSCMGoVersion() string {
	if p != nil && p.SCMGoVersion != nil {
		return *p.SCMGoVersion
	}
	return ""
}
func (p *CodeCleanTaskExtra) GetCreateMRIfLineChangeGreaterThan() int32 {
	if p != nil && p.CreateMRIfLineChangeGreaterThan != nil {
		return *p.CreateMRIfLineChangeGreaterThan
	}
	return 0
}
func (p *CodeCleanTaskExtra) GetRecentBuildTimeUnix() int64 {
	if p != nil && p.RecentBuildTimeUnix != nil {
		return *p.RecentBuildTimeUnix
	}
	return 0
}

type IDLCleanTask struct {
	// 服务名
	PSM *string `thrift:"PSM,1,optional" frugal:"1,optional,string" json:"PSM,omitempty"`
	// idl仓库名
	RepoName string `thrift:"RepoName,2,required" frugal:"2,required,string" json:"RepoName"`
	// 需要清理的方法
	Methods []string `thrift:"Methods,3,required" frugal:"3,required,list<string>" json:"Methods"`
	// idl文件路径
	IDLPath string `thrift:"IDLPath,4,required" frugal:"4,required,string" json:"IDLPath"`
	// 清理分支
	SourceBranch string `thrift:"SourceBranch,5,required" frugal:"5,required,string" json:"SourceBranch"`
	// 目标分支
	CommitBranch string `thrift:"CommitBranch,6,required" frugal:"6,required,string" json:"CommitBranch"`
	// 提交人
	Author string `thrift:"Author,8,required" frugal:"8,required,string" json:"Author"`
}

type StorageCleanTask struct {
	// 存储psm
	StoragePSM string `thrift:"StoragePSM,1" frugal:"1,default,string" json:"StoragePSM"`
	// 存储类型
	StorageType string `thrift:"StorageType,2" frugal:"2,default,string" json:"StorageType"`
	// 是否只检查
	OnlyCheck bool `thrift:"OnlyCheck,3" frugal:"3,default,bool" json:"OnlyCheck"`
}

func (s *StorageCleanTask) GetStoragePSM() string {
	if s != nil {
		return s.StoragePSM
	}
	return ""
}
func (s *StorageCleanTask) GetStorageType() string {
	if s != nil {
		return s.StorageType
	}
	return ""
}
func (s *StorageCleanTask) GetOnlyCheck() bool {
	if s != nil {
		return s.OnlyCheck
	}
	return false
}

type CodeCleanTask struct {
	ID *int64 `thrift:"ID,1,optional" frugal:"1,optional,i64" json:"ID,omitempty"`
	// -1未执行，1正在执行，2成功，3失败, -2执行了，但版本不匹配，任务跳过，等待下一次执行, 4 不完全成功，废弃接口识别不全
	TaskStatus *int64 `thrift:"TaskStatus,2,optional" frugal:"2,optional,i64" json:"TaskStatus,omitempty"`
	// 仓库名
	RepoName *string `thrift:"RepoName,3,optional" frugal:"3,optional,string" json:"RepoName,omitempty"`
	// 用于标记Go版本
	TaskOpt *string `thrift:"TaskOpt,4,optional" frugal:"4,optional,string" json:"TaskOpt,omitempty"`
	// 任务成功时输出的结果
	TaskResultSha *string `thrift:"TaskResultSha,5,optional" frugal:"5,optional,string" json:"TaskResultSha,omitempty"`
	// 总行数
	TaskResultTotalLine *int64 `thrift:"TaskResultTotalLine,6,optional" frugal:"6,optional,i64" json:"TaskResultTotalLine,omitempty"`
	// 删除函数
	TaskResultUnusedLine *int64 `thrift:"TaskResultUnusedLine,7,optional" frugal:"7,optional,i64" json:"TaskResultUnusedLine,omitempty"`
	// 清理MR
	TaskResultMrLink *string `thrift:"TaskResultMrLink,8,optional" frugal:"8,optional,string" json:"TaskResultMrLink,omitempty"`
	// debug信息
	TaskResultErrMsg *string `thrift:"TaskResultErrMsg,9,optional" frugal:"9,optional,string" json:"TaskResultErrMsg,omitempty"`
	TaskResultStdErr *string `thrift:"TaskResultStdErr,10,optional" frugal:"10,optional,string" json:"TaskResultStdErr,omitempty"`
	TaskResultStdout *string `thrift:"TaskResultStdout,11,optional" frugal:"11,optional,string" json:"TaskResultStdout,omitempty"`
	// extra信息
	Extra *CodeCleanTaskExtra `thrift:"Extra,12,optional" frugal:"12,optional,CodeCleanTaskExtra" json:"Extra,omitempty"`
	// 时间信息
	CreatedAt *string `thrift:"CreatedAt,13,optional" frugal:"13,optional,string" json:"CreatedAt,omitempty"`
	UpdatedAt *string `thrift:"UpdatedAt,14,optional" frugal:"14,optional,string" json:"UpdatedAt,omitempty"`
	// 是否是idl清理任务
	IsIDLClean *bool `thrift:"IsIDLClean,15,optional" frugal:"15,optional,bool" json:"IsIDLClean,omitempty"`
	// IDL清理任务的处理信息
	IDLCleanTask *IDLCleanTask `thrift:"IDLCleanTask,16,optional" frugal:"16,optional,IDLCleanTask" json:"IDLCleanTask,omitempty"`
	// 优先级
	Priority *int32 `thrift:"Priority,17,optional" frugal:"17,optional,i32" json:"Priority,omitempty"`
	// 基准commit
	BaseCommitID *string `thrift:"BaseCommitID,18,optional" frugal:"18,optional,string" json:"BaseCommitID,omitempty"`
	// 任务链接
	TaskURL *string `thrift:"TaskURL,19,optional" frugal:"19,optional,string" json:"TaskURL,omitempty"`
	// 清理工具版本
	CleanerVersion *string `thrift:"CleanerVersion,20,optional" frugal:"20,optional,string" json:"CleanerVersion,omitempty"`
	// 存储清理任务的处理信息
	StorageCleanTask *StorageCleanTask `thrift:"StorageCleanTask,21,optional" frugal:"21,optional,StorageCleanTask" json:"StorageCleanTask,omitempty"`
	// 是否是代码治理的自动修复任务
	IsCodeGovernFixerTask bool `thrift:"IsCodeGovernFixerTask,22" frugal:"22,bool" json:"IsCodeGovernFixerTask"`
	// Panic修复任务的处理信息
	FixerTask *FixerTask `thrift:"FixerTask,23,optional" frugal:"23,optional,FixerTask" json:"FixerTask,omitempty"`
}

func (p *CodeCleanTask) GetID() int64 {
	if p != nil && p.ID != nil {
		return *p.ID
	}
	return 0
}

func (p *CodeCleanTask) GetTaskStatus() int64 {
	if p != nil && p.TaskStatus != nil {
		return *p.TaskStatus
	}
	return 0
}

func (p *CodeCleanTask) GetRepoName() string {
	if p != nil && p.RepoName != nil {
		return *p.RepoName
	}
	return ""
}

func (p *CodeCleanTask) GetTaskOpt() string {
	if p != nil && p.TaskOpt != nil {
		return *p.TaskOpt
	}
	return ""
}

func (p *CodeCleanTask) GetTaskResultSha() string {
	if p != nil && p.TaskResultSha != nil {
		return *p.TaskResultSha
	}
	return ""
}

func (p *CodeCleanTask) GetTaskResultTotalLine() int64 {
	if p != nil && p.TaskResultTotalLine != nil {
		return *p.TaskResultTotalLine
	}
	return 0
}

func (p *CodeCleanTask) GetTaskResultUnusedLine() int64 {
	if p != nil && p.TaskResultUnusedLine != nil {
		return *p.TaskResultUnusedLine
	}
	return 0
}

func (p *CodeCleanTask) GetTaskResultMrLink() string {
	if p != nil && p.TaskResultMrLink != nil {
		return *p.TaskResultMrLink
	}
	return ""
}

func (p *CodeCleanTask) GetTaskResultErrMsg() string {
	if p != nil && p.TaskResultErrMsg != nil {
		return *p.TaskResultErrMsg
	}
	return ""
}

func (p *CodeCleanTask) GetExtra() *CodeCleanTaskExtra {
	if p != nil {
		return p.Extra
	}
	return nil
}

func (p *CodeCleanTask) GetCreatedAt() string {
	if p != nil && p.CreatedAt != nil {
		return *p.CreatedAt
	}
	return ""
}

func (p *CodeCleanTask) GetUpdatedAt() string {
	if p != nil && p.UpdatedAt != nil {
		return *p.UpdatedAt
	}
	return ""
}

func (p *CodeCleanTask) GetIsIDLClean() bool {
	if p != nil && p.IsIDLClean != nil {
		return *p.IsIDLClean
	}
	return false
}

func (p *CodeCleanTask) GetIDLCleanTask() *IDLCleanTask {
	if p != nil {
		return p.IDLCleanTask
	}
	return nil
}

func (p *CodeCleanTask) IsFixerTask() bool {
	if p != nil && p.FixerTask != nil {
		return true
	}
	return false
}

func (p *CodeCleanTask) GetFixerTask() *FixerTask {
	if p != nil {
		return p.FixerTask
	}
	return nil
}

func (p *CodeCleanTask) GetPriority() int32 {
	if p != nil && p.Priority != nil {
		return *p.Priority
	}
	return 0
}

func (p *CodeCleanTask) GetBaseCommitID() string {
	if p != nil && p.BaseCommitID != nil {
		return *p.BaseCommitID
	}
	return ""
}

func (p *CodeCleanTask) GetTaskURL() string {
	if p != nil && p.TaskURL != nil {
		return *p.TaskURL
	}
	return ""
}

func (p *CodeCleanTask) GetCleanerVersion() string {
	if p != nil && p.CleanerVersion != nil {
		return *p.CleanerVersion
	}
	return ""
}

func (p *CodeCleanTask) GetStorageCleanTask() *StorageCleanTask {
	if p != nil {
		return p.StorageCleanTask
	}
	return nil
}

func (p *CodeCleanTask) GetTaskResultStdErr() string {
	if p != nil && p.TaskResultStdErr != nil {
		return *p.TaskResultStdErr
	}
	return ""
}

func (p *CodeCleanTask) GetTaskResultStdout() string {
	if p != nil && p.TaskResultStdout != nil {
		return *p.TaskResultStdout
	}
	return ""
}

type AcquireCodeCleanTaskResponse struct {
	Task *CodeCleanTask `thrift:"Task,1,optional" frugal:"1,optional,CodeCleanTask" json:"Task,omitempty"`
	// 上报间隔，单位秒
	ReportInterval *int32    `thrift:"ReportInterval,2,optional" frugal:"2,optional,i32" json:"ReportInterval,omitempty"`
	BaseResp       *BaseResp `thrift:"BaseResp,255,required" frugal:"255,required,base.BaseResp" json:"BaseResp"`
}

func (p *AcquireCodeCleanTaskResponse) GetTask() *CodeCleanTask {
	if p != nil {
		return p.Task
	}
	return nil
}
func (p *AcquireCodeCleanTaskResponse) GetReportInterval() int32 {
	if p != nil && p.ReportInterval != nil {
		return *p.ReportInterval
	}
	return 0
}
func (p *AcquireCodeCleanTaskResponse) GetBaseResp() *BaseResp {
	if p != nil {
		return p.BaseResp
	}
	return nil
}
func (p *AcquireCodeCleanTaskResponse) SetBaseResp(val *BaseResp) {
	if p != nil {
		p.BaseResp = val
	}
}

type SaveCodeCleanTaskRequest struct {
	Task *CodeCleanTask `thrift:"Task,1,optional" frugal:"1,optional,CodeCleanTask" json:"Task,omitempty"`
	// 是否正在处理中，用于定期提交信息刷新后台数据
	Processing *bool `thrift:"Processing,2,optional" frugal:"2,optional,bool" json:"Processing,omitempty"`
	// 上报处理的worker
	Worker *string `thrift:"Worker,3,optional" frugal:"3,optional,string" json:"Worker,omitempty"`
	// 机器状态
	MachineStat *string `thrift:"MachineStat,4,optional" frugal:"4,optional,string" json:"MachineStat,omitempty"`
	// 开始处理的时间
	StartTime *int64 `thrift:"StartTime,5,optional" frugal:"5,optional,i64" json:"StartTime,omitempty"`
}

func (p *SaveCodeCleanTaskRequest) GetTask() *CodeCleanTask {
	if p != nil {
		return p.Task
	}
	return nil
}
func (p *SaveCodeCleanTaskRequest) GetProcessing() bool {
	if p != nil && p.Processing != nil {
		return *p.Processing
	}
	return false
}
func (p *SaveCodeCleanTaskRequest) GetWorker() string {
	if p != nil && p.Worker != nil {
		return *p.Worker
	}
	return ""
}
func (p *SaveCodeCleanTaskRequest) GetMachineStat() string {
	if p != nil && p.MachineStat != nil {
		return *p.MachineStat
	}
	return ""
}
func (p *SaveCodeCleanTaskRequest) GetStartTime() int64 {
	if p != nil && p.StartTime != nil {
		return *p.StartTime
	}
	return 0
}

type SaveCodeCleanTaskResponse struct {
	Task     *CodeCleanTask `thrift:"Task,1,optional" frugal:"1,optional,CodeCleanTask" json:"Task,omitempty"`
	BaseResp *BaseResp      `thrift:"BaseResp,255,required" frugal:"255,required,base.BaseResp" json:"BaseResp"`
}

func (p *SaveCodeCleanTaskResponse) GetTask() *CodeCleanTask {
	if p != nil {
		return p.Task
	}
	return nil
}
func (p *SaveCodeCleanTaskResponse) GetBaseResp() *BaseResp {
	if p != nil {
		return p.BaseResp
	}
	return nil
}
func (p *SaveCodeCleanTaskResponse) SetBaseResp(val *BaseResp) {
	if p != nil {
		p.BaseResp = val
	}
}

type BaseResp struct {
	StatusMessage string            `thrift:"StatusMessage,1,required" frugal:"1,required,string" json:"StatusMessage"`
	StatusCode    int32             `thrift:"StatusCode,2,required" frugal:"2,required,i32" json:"StatusCode"`
	Extra         map[string]string `thrift:"Extra,3,optional" frugal:"3,optional,map<string:string>" json:"Extra,omitempty"`
}

func (p *BaseResp) GetStatusMessage() string {
	return p.StatusMessage
}
func (p *BaseResp) GetStatusCode() int32 {
	return p.StatusCode
}
func (p *BaseResp) GetExtra() map[string]string {
	return p.Extra
}
