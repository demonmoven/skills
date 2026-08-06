package fsstore

import (
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"errors"
	"os"
)

// ==================== 成长（经验值/胜率）：workspace/stats/<adv_id>.json ====================
// 热数据同步写到 adventurers/*.json，但冷数据(分任务的成长历史)放这里
// MVP 里只放累计数值，MVP 之后再引入时序

// 等级称号表（10 级上限，按职业区分）
var levelTitles = map[model.AdventurerClass][]string{
	model.ClassWarrior: {
		1:  "新兵",
		2:  "剑士",
		3:  "大剑士",
		4:  "剑豪",
		5:  "剑圣",
		6:  "剑尊",
		7:  "剑神",
		8:  "宗师",
		9:  "传奇剑客",
		10: "神话剑士",
	},
	model.ClassMage: {
		1:  "学徒",
		2:  "法师",
		3:  "大法师",
		4:  "魔导士",
		5:  "魔导师",
		6:  "大魔导师",
		7:  "贤者",
		8:  "宗师",
		9:  "传奇法师",
		10: "神话法师",
	},
}

// LevelTitle 根据职业和等级返回称号。
func LevelTitle(class model.AdventurerClass, level int) string {
	titles, ok := levelTitles[class]
	if !ok || level < 1 || level >= len(titles) {
		return ""
	}
	return titles[level]
}

// AdvRuntimeStats 是单个冒险者的累计运行统计（回合、token、时长、费用、分模型统计）。
type AdvRuntimeStats struct {
	ID              string               `json:"id"`
	TotalQuests     int                  `json:"total_quests"`
	TotalTurns      int64                `json:"total_turns"`
	TotalTokensIn   int64                `json:"total_tokens_in"`
	TotalTokensOut  int64                `json:"total_tokens_out"`
	TotalDurationMs int64                `json:"total_duration_ms"`
	TotalCostUSD    float64              `json:"total_cost_usd"`
	PerModelStats   map[string]*PerModel `json:"per_model"`
}

// PerModel 是单个模型的细分统计：token 消耗、胜负次数、费用。
type PerModel struct {
	TokensIn  int64   `json:"tokens_in"`
	TokensOut int64   `json:"tokens_out"`
	Win       int     `json:"win"`
	Lose      int     `json:"lose"`
	CostUSD   float64 `json:"cost_usd"`
}

// StatsStore 负责单个冒险者运行统计的持久化（JSON 文件）。
type StatsStore struct{ root *Root }

// NewStatsStore 创建 StatsStore，root 为根目录封装。
func NewStatsStore(root *Root) *StatsStore { return &StatsStore{root: root} }

func (s *StatsStore) path(advID string) string {
	return s.root.Sub(SubdirStats, advID+".json")
}

// Load 读取冒险者统计。文件不存在返回空统计（不报错）；
// 权限/损坏等错误会上报以避免下次 Save 覆盖真实数据。
func (s *StatsStore) Load(advID string) (*AdvRuntimeStats, error) {
	st, err := ReadJSON[AdvRuntimeStats](s.path(advID))
	if err != nil {
		// 仅 os.ErrNotExist 视为「空」；权限/损坏等错误上报（避免下次 Save 覆盖真数据）
		if errors.Is(err, os.ErrNotExist) {
			return &AdvRuntimeStats{
				ID:            advID,
				PerModelStats: map[string]*PerModel{},
			}, nil
		}
		return nil, err
	}
	if st.PerModelStats == nil {
		st.PerModelStats = map[string]*PerModel{}
	}
	return st, nil
}

// Save 持久化冒险者统计。
func (s *StatsStore) Save(st *AdvRuntimeStats) error {
	return WriteJSON(s.path(st.ID), st)
}

// RecordTurn 累加一回合使用
func (s *StatsStore) RecordTurn(advID, model string, tokensIn, tokensOut int64, costUSD float64, durationMs int64) error {
	st, err := s.Load(advID)
	if err != nil {
		return err
	}
	st.TotalTurns++
	st.TotalTokensIn += tokensIn
	st.TotalTokensOut += tokensOut
	st.TotalDurationMs += durationMs
	st.TotalCostUSD += costUSD
	pm := st.PerModelStats[model]
	if pm == nil {
		pm = &PerModel{}
		st.PerModelStats[model] = pm
	}
	pm.TokensIn += tokensIn
	pm.TokensOut += tokensOut
	pm.CostUSD += costUSD
	return s.Save(st)
}

// RecordVerdict 追加胜负到对应模型 + 同步到 adventurers/*.json
// 注意：只在用户终审后调用，不是每轮法师评审都调
func (s *StatsStore) RecordVerdict(a *AdventurerFile, verdict model.QuestVerdict, expGain int64, modelName string) error {
	// 先写 stats
	st, err := s.Load(a.ID)
	if err != nil {
		return err
	}
	st.TotalQuests++
	pm := st.PerModelStats[modelName]
	if pm == nil {
		pm = &PerModel{}
		st.PerModelStats[modelName] = pm
	}
	switch verdict {
	case model.VerdictPass:
		pm.Win++
		a.WinCount++
	case model.VerdictReject:
		pm.Lose++
		a.LoseCount++
	case model.VerdictRequestChange:
		// request_changes 不记胜败
	}
	if err := s.Save(st); err != nil {
		return err
	}
	// 经验值
	a.Exp += expGain
	level := LevelFromExp(a.Exp)
	if level > a.Level {
		a.Level = level
	}
	return s.root.SaveAdventurer(a)
}

// LevelFromExp 按等差级数反算等级（10 级上限）。
// Lv.1 = 0, Lv.2 = 100, Lv.3 = 300, Lv.4 = 600, Lv.5 = 1000, ..., Lv.10 = 4500
// 公式：累计到 lv 的经验 = 100 * (lv-1) * lv / 2
func LevelFromExp(exp int64) int {
	if exp < 100 {
		return 1
	}
	for lv := 2; lv <= 10; lv++ {
		need := int64(100 * (lv - 1) * lv / 2)
		if exp < need {
			return lv - 1
		}
	}
	return 10
}
