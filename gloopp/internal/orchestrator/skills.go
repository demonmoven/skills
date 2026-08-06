package orchestrator

import (
	"fmt"

	"code.byted.org/lihuanyu.0w0/gloop/internal/skills"
)

// ==================== 技能（Engine 层包装） ====================
//
// gloop v2 的技能全部是内置只读的，随二进制 embed，不支持用户导入。
// 这里只做一层薄包装，供 API 层调用。

// SkillManifestExt 是 ListSkills 返回的扩展条目（元数据 + 来源）。
type SkillManifestExt struct {
	skills.Manifest
	Source string `json:"source"` // 恒为 "builtin"
}

func (e *Engine) ListSkillsManifests() ([]SkillManifestExt, error) {
	if e.skills == nil {
		return nil, fmt.Errorf("技能注册表未初始化")
	}
	items := e.skills.ListManifests()
	out := make([]SkillManifestExt, 0, len(items))
	for _, m := range items {
		out = append(out, SkillManifestExt{Manifest: m, Source: "builtin"})
	}
	return out, nil
}

func (e *Engine) GetSkillDetail(name string) (ext *SkillManifestExt, body, formatted string, err error) {
	if e.skills == nil {
		return nil, "", "", fmt.Errorf("技能注册表未初始化")
	}
	sk, err := e.skills.Get(name)
	if err != nil {
		return nil, "", "", err
	}
	ext = &SkillManifestExt{Manifest: sk.Manifest(), Source: "builtin"}
	body = sk.Body
	formatted = sk.FormatMarkdown()
	return ext, body, formatted, nil
}
