package idl

import (
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/skills"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

const ExportVersion = "gloop.idl.v1"

// Bundle 是 gloop 对外披露的接口定义。
//
// v0.3.4 起 native platform tool 分发层已移除——agent→gloop 统一走 CLI syscall
// （phase done / review / post / quest ask 等），工具协议通过 skills 披露。
// 因此 Bundle 只导出 skills 和 per-class skill 可见性，不再导出 tools / agent_schemas。
type Bundle struct {
	IDLVersion string                  `json:"idl_version"`
	App        string                  `json:"app"`
	AppVersion string                  `json:"app_version"`
	Skills     []skills.Manifest       `json:"skills"`
	Classes    map[string]ClassExports `json:"classes"`
}

type ClassExports struct {
	Skills []string `json:"skills"`
}

func BuildBundle() Bundle {
	return Bundle{
		IDLVersion: ExportVersion,
		App:        version.AppName,
		AppVersion: version.Version,
		Skills:     skills.Default().ListManifests(),
		Classes:    buildClassExports(),
	}
}

func buildClassExports() map[string]ClassExports {
	out := map[string]ClassExports{}
	for _, class := range []model.AdventurerClass{model.ClassWarrior, model.ClassMage} {
		skillNames := []string{}
		for _, s := range skills.Default().ListManifestsForClass(class) {
			skillNames = append(skillNames, s.Name)
		}
		out[string(class)] = ClassExports{Skills: skillNames}
	}
	return out
}
