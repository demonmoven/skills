package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type Task struct {
	ID                   uint64         `gorm:"column:id" json:"id"`                                             // 主键id
	RepoName             string         `gorm:"column:repo_name" json:"repo_name"`                               // 仓库名
	TaskStatus           int64          `gorm:"column:task_status" json:"task_status"`                           // 任务状态,-1未执行，1正在执行，2成功，3失败, -2执行了，但版本不匹配，任务跳过，等待下一次执行, 4 不完全成功，废弃接口识别不全
	TaskOpt              string         `gorm:"column:task_opt;type:text" json:"task_opt"`                       // 参数, 这个仓库的go mod version
	TaskResultErrMsg     string         `gorm:"column:task_result_err_msg;type:text" json:"task_result_err_msg"` // 错误信息
	TaskResultTotalLine  int64          `gorm:"column:task_result_total_line" json:"task_result_total_line"`     // 总行数
	TaskResultUnusedLine int64          `gorm:"column:task_result_unused_line" json:"task_result_unused_line"`   // 无用行数
	TaskResultStdout     string         `gorm:"column:task_result_stdout" json:"task_result_stdout"`             // stdout
	TaskResultStderr     string         `gorm:"column:task_result_stderr" json:"task_result_stderr"`             // stderr
	TaskResultSHA        string         `gorm:"column:task_result_sha" json:"task_result_sha"`                   // 扫描Git版本
	TaskResultMRLink     string         `gorm:"column:task_result_mr_link;type:text" json:"task_result_mr_link"` // MRLink
	Extra                string         `gorm:"column:extra;type:text" json:"extra"`                             // 其它信息
	CreatedAt            time.Time      `gorm:"column:created_at;type:text" json:"created_at"`                   // 创建时间
	UpdatedAt            time.Time      `gorm:"column:updated_at" json:"updated_at"`                             // 更新时间
	DeletedAt            gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`                             // 删除时间
}

func (*Task) TableName() string {
	return "sg_asset_code_clean_task"
}

func (t *Task) split() (tasks []*Task) {
	for _, v := range strings.Split(t.RepoName, ",") {
		if len(v) == 0 {
			continue
		}
		task := *t
		task.RepoName = v
		tasks = append(tasks, &task)
	}
	return
}

func (t *Task) Submit() {
	for _, v := range t.split() {
		v.TaskStatus = -1
		var vv Task
		DB().Where(Task{TaskStatus: v.TaskStatus, RepoName: v.RepoName}).FirstOrCreate(&vv)
		v.ID = vv.ID
		v.CreatedAt = vv.CreatedAt
		_ = DB().Save(v).Error
	}
}

func (t *Task) TaskList(condition *TaskListCondition) (tasks []*Task) {
	condition.validateOffset()
	tasks = []*Task{}
	_ = DB().Model(t).Where(t).Order("id DESC").Offset(int(condition.Offset)).Limit(int(condition.PageSize)).Scan(&tasks)
	return
}

func (t *Task) Delete() {
	DB().Where(t).Delete(t)
	for _, v := range t.split() {
		DB().Where(v).Delete(v)
	}
}

func (t *Task) Fetch() *Task {
	DB().Model(t).Where(t).Order("id").First(t)
	if DB().Model(t).Where(t).Update("task_status", 1).RowsAffected > 0 {
		t.TaskStatus = 1
		return t
	}
	return nil
}

func (t *Task) FetchUnStartedWithVersion(vs []string) *Task {
	DB().Model(t).Where("task_status = ?", -2).Where("task_opt IN ?", vs).Order("id").First(t)
	if t.ID == 0 {
		return nil
	}
	if DB().Model(t).Where(t).Update("task_status", 1).RowsAffected > 0 {
		t.TaskStatus = 1
		return t
	}
	return nil
}

func (t *Task) Save() {
	_ = DB().Save(t).Error
}

type (
	taskOpt struct {
		limit int
	}

	TaskOption func(opt *taskOpt)
)

func newTaskOpt(options ...TaskOption) taskOpt {
	var o = taskOpt{limit: -1}

	for _, f := range options {
		f(&o)
	}

	return o
}

func WithLimit(limit int) TaskOption {
	return func(opt *taskOpt) {
		opt.limit = limit
	}
}
