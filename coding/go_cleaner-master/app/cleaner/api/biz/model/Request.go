package model

type TaskListCondition struct {
	PageSize  int64 `json:"page_size"`  // 一页数量
	PageIndex int64 `json:"page_index"` // 第几页
	Offset    int64 `json:"offset"`     // 偏移量，前端传，后端计算生成
}

func (t *TaskListCondition) validateOffset() {
	//兼容老逻辑，默认400个一页
	if t.PageIndex == 0 {
		t.PageIndex = 1
		t.PageSize = 400
	}
	if t.PageIndex < 0 {
		t.PageIndex = 1
		t.PageSize = 100
	}
	if t.PageSize > 400 {
		t.PageSize = 400
	}
	t.Offset = (t.PageIndex - 1) * t.PageSize
}
