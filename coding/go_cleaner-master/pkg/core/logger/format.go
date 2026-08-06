package logger

import (
	"encoding/json"
	"fmt"

	"golang.org/x/tools/go/packages"
)

// PrettyPackagesConfigJSON returns an indented JSON string for a packages.Config,
// skipping non-serializable fields and expanding Mode to readable flags.
func PrettyPackagesConfigJSON(cfg *packages.Config) string {
	if cfg == nil {
		return "{}"
	}

	type snap struct {
		// Mode as hex and decomposed flags (readable)
		ModeHex   string   `json:"modeHex"`
		ModeFlags []string `json:"modeFlags"`

		Dir        string         `json:"dir,omitempty"`
		Env        []string       `json:"env,omitempty"` // 注意可能含敏感信息
		BuildFlags []string       `json:"buildFlags,omitempty"`
		Tests      bool           `json:"tests,omitempty"`
		OverlayLen map[string]int `json:"overlayBytes,omitempty"` // 仅记录大小，避免输出内容

		HasFset      bool `json:"hasFset"`
		HasParseFile bool `json:"hasParseFile"`
		HasLogf      bool `json:"hasLogf"`
	}

	out := snap{
		ModeHex:      fmt.Sprintf("0x%X", cfg.Mode),
		ModeFlags:    modeFlags(cfg.Mode),
		Dir:          cfg.Dir,
		Env:          cfg.Env,
		BuildFlags:   cfg.BuildFlags,
		Tests:        cfg.Tests,
		HasFset:      cfg.Fset != nil,
		HasParseFile: cfg.ParseFile != nil,
		HasLogf:      cfg.Logf != nil,
		OverlayLen:   overlayLens(cfg.Overlay),
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		// 极端情况下也保证有输出
		return fmt.Sprintf(`{"error":"%v"}`, err)
	}
	return string(b)
}

func overlayLens(m map[string][]byte) map[string]int {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[k] = len(v)
	}
	return out
}

// 把 packages.LoadMode 拆成可读的标志名列表。
// 未识别的位会以 "UNKNOWN(0x...)" 形式保留。
func modeFlags(m packages.LoadMode) []string {
	type flag struct {
		val packages.LoadMode
		s   string
	}
	known := []flag{
		{packages.NeedName, "NeedName"},
		{packages.NeedFiles, "NeedFiles"},
		{packages.NeedCompiledGoFiles, "NeedCompiledGoFiles"},
		{packages.NeedImports, "NeedImports"},
		{packages.NeedDeps, "NeedDeps"},
		{packages.NeedTypes, "NeedTypes"},
		{packages.NeedSyntax, "NeedSyntax"},
		{packages.NeedTypesInfo, "NeedTypesInfo"},
		{packages.NeedTypesSizes, "NeedTypesSizes"},
		{packages.NeedModule, "NeedModule"},
		{packages.NeedEmbedFiles, "NeedEmbedFiles"},
		{packages.NeedEmbedPatterns, "NeedEmbedPatterns"},
		// 有些版本还包含 NeedExportFile / NeedExportData 等，按需添加
	}
	var out []string
	var mask packages.LoadMode
	for _, k := range known {
		if m&k.val != 0 {
			out = append(out, k.s)
			mask |= k.val
		}
	}
	unknown := m &^ mask
	if unknown != 0 {
		out = append(out, fmt.Sprintf("UNKNOWN(%#x)", uint64(unknown)))
	}
	return out
}
