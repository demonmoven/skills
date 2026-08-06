package fsstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

const (
	ConfigPackageKind          = "gloop.config_package"
	ConfigPackageSchemaVersion = 1
)

type ConfigPackage struct {
	Kind          string                `json:"kind"`
	SchemaVersion int                   `json:"schema_version"`
	GloopVersion  string                `json:"gloop_version"`
	ExportedAtMs  int64                 `json:"exported_at_ms"`
	Sections      ConfigPackageSections `json:"sections"`
}

type ConfigPackageSections struct {
	Config      *GlobalConfig       `json:"config,omitempty"`
	Agents      []AgentPackage      `json:"agents,omitempty"`
	Adventurers []*AdventurerFile   `json:"adventurers,omitempty"`
	Automations []*AutomationConfig `json:"automations,omitempty"`
	Prompts     map[string]string   `json:"prompts,omitempty"`
	Context     *ContextPackage     `json:"context,omitempty"`
}

type AgentPackage struct {
	Name                string            `json:"name"`
	Type                string            `json:"type"`
	Command             string            `json:"command"`
	Args                []string          `json:"args"`
	Env                 map[string]string `json:"env,omitempty"`
	EnvFiles            []string          `json:"env_files,omitempty"`
	RedactedEnvKeys     []string          `json:"redacted_env_keys,omitempty"`
	DefaultModel        string            `json:"default_model"`
	DefaultAllowedTools []string          `json:"default_allowed_tools,omitempty"`
	SupportsReadOnly    *bool             `json:"supports_readonly,omitempty"`
	ExecutionTraits     []string          `json:"execution_traits,omitempty"`
	Enabled             bool              `json:"enabled"`
}

type ContextPackage struct {
	Summary string            `json:"summary,omitempty"`
	Dims    map[string]string `json:"dims,omitempty"`
}

type ConfigPackagePreview struct {
	Ok       bool                   `json:"ok"`
	Sections []ConfigSectionPreview `json:"sections"`
	Warnings []string               `json:"warnings,omitempty"`
}

type ConfigSectionPreview struct {
	Name     string   `json:"name"`
	Creates  int      `json:"creates"`
	Updates  int      `json:"updates"`
	Skips    int      `json:"skips"`
	Warnings []string `json:"warnings,omitempty"`
}

func (r *Root) ExportConfigPackage() (*ConfigPackage, error) {
	cfg, err := r.LoadConfig()
	if err != nil {
		return nil, err
	}
	agents, err := r.ListAgents()
	if err != nil {
		return nil, err
	}
	adventurers, err := r.ListAdventurers()
	if err != nil {
		return nil, err
	}
	automations, err := r.ListAutomations()
	if err != nil {
		return nil, err
	}
	prompts, err := readTextTree(r.Sub(SubdirPrompts))
	if err != nil {
		return nil, err
	}
	ctx, err := r.exportContext()
	if err != nil {
		return nil, err
	}

	return &ConfigPackage{
		Kind:          ConfigPackageKind,
		SchemaVersion: ConfigPackageSchemaVersion,
		GloopVersion:  version.Version,
		ExportedAtMs:  NowMs(),
		Sections: ConfigPackageSections{
			Config:      cfg,
			Agents:      sanitizeAgents(agents),
			Adventurers: adventurers,
			Automations: automations,
			Prompts:     prompts,
			Context:     ctx,
		},
	}, nil
}

func (r *Root) PreviewConfigPackage(pkg *ConfigPackage) (*ConfigPackagePreview, error) {
	if err := validateConfigPackage(pkg); err != nil {
		return nil, err
	}
	out := &ConfigPackagePreview{Ok: true}
	add := func(p ConfigSectionPreview) {
		out.Sections = append(out.Sections, p)
		out.Warnings = append(out.Warnings, p.Warnings...)
	}

	if pkg.Sections.Config != nil {
		add(ConfigSectionPreview{Name: "config", Updates: 1})
	}
	add(r.previewAgents(pkg.Sections.Agents))
	add(r.previewAdventurers(pkg.Sections.Adventurers))
	add(r.previewAutomations(pkg.Sections.Automations))
	add(previewTextMap("prompts", r.Sub(SubdirPrompts), pkg.Sections.Prompts))
	add(r.previewContext(pkg.Sections.Context))
	return out, nil
}

func (r *Root) ImportConfigPackage(pkg *ConfigPackage) (*GlobalConfig, error) {
	if err := validateConfigPackage(pkg); err != nil {
		return nil, err
	}
	if cfg := pkg.Sections.Config; cfg != nil {
		applyDefaults(cfg, DefaultGlobalConfig())
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("配置校验失败: %w", err)
		}
		if err := r.SaveConfig(cfg); err != nil {
			return nil, err
		}
	}
	if err := r.importAgents(pkg.Sections.Agents); err != nil {
		return nil, err
	}
	if err := r.importAdventurers(pkg.Sections.Adventurers); err != nil {
		return nil, err
	}
	if err := r.importAutomations(pkg.Sections.Automations); err != nil {
		return nil, err
	}
	if err := writeTextTree(r.Sub(SubdirPrompts), pkg.Sections.Prompts); err != nil {
		return nil, err
	}
	if err := r.importContext(pkg.Sections.Context); err != nil {
		return nil, err
	}
	return r.LoadConfig()
}

func sanitizeAgents(in []*AgentConfig) []AgentPackage {
	out := make([]AgentPackage, 0, len(in))
	for _, p := range in {
		if p == nil {
			continue
		}
		keys := make([]string, 0, len(p.Env))
		for k := range p.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out = append(out, AgentPackage{
			Name:                p.Name,
			Type:                string(p.Type),
			Command:             p.Command,
			Args:                append([]string(nil), p.Args...),
			EnvFiles:            append([]string(nil), p.EnvFiles...),
			RedactedEnvKeys:     keys,
			DefaultModel:        p.DefaultModel,
			DefaultAllowedTools: append([]string(nil), p.DefaultAllowedTools...),
			SupportsReadOnly:    cloneBoolPtr(p.SupportsReadOnly),
			ExecutionTraits:     append([]string(nil), p.ExecutionTraits...),
			Enabled:             p.Enabled,
		})
	}
	return out
}

func cloneBoolPtr(v *bool) *bool {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func validateConfigPackage(pkg *ConfigPackage) error {
	if pkg == nil {
		return fmt.Errorf("配置包为空")
	}
	if pkg.Kind != "" && pkg.Kind != ConfigPackageKind {
		return fmt.Errorf("不支持的配置包类型: %s", pkg.Kind)
	}
	if pkg.SchemaVersion != 0 && pkg.SchemaVersion > ConfigPackageSchemaVersion {
		return fmt.Errorf("配置包 schema 版本过新: %d", pkg.SchemaVersion)
	}
	return nil
}

func (r *Root) importAgents(items []AgentPackage) error {
	for _, item := range items {
		if err := validatePathSegment(item.Name, "agent"); err != nil {
			return err
		}
		env := item.Env
		if env == nil && len(item.RedactedEnvKeys) > 0 {
			if old, err := r.GetAgent(item.Name); err == nil {
				env = old.Env
			}
		}
		p := &AgentConfig{
			Name:                item.Name,
			Type:                model.AgentType(item.Type),
			Command:             item.Command,
			Args:                item.Args,
			Env:                 env,
			EnvFiles:            append([]string(nil), item.EnvFiles...),
			DefaultModel:        item.DefaultModel,
			DefaultAllowedTools: append([]string(nil), item.DefaultAllowedTools...),
			SupportsReadOnly:    cloneBoolPtr(item.SupportsReadOnly),
			ExecutionTraits:     append([]string(nil), item.ExecutionTraits...),
			Enabled:             item.Enabled,
		}
		if err := r.SaveAgent(p); err != nil {
			return err
		}
	}
	return nil
}

func (r *Root) importAdventurers(items []*AdventurerFile) error {
	for _, item := range items {
		if item == nil {
			continue
		}
		if err := validatePathSegment(item.ID, "adventurer"); err != nil {
			return err
		}
		if err := r.SaveAdventurer(item); err != nil {
			return err
		}
	}
	return nil
}

func (r *Root) importAutomations(items []*AutomationConfig) error {
	for _, item := range items {
		if item == nil {
			continue
		}
		if err := validatePathSegment(item.ID, "automation"); err != nil {
			return err
		}
		if err := r.SaveAutomation(item); err != nil {
			return err
		}
	}
	return nil
}

func (r *Root) importContext(ctx *ContextPackage) error {
	if ctx == nil {
		return nil
	}
	if strings.TrimSpace(ctx.Summary) != "" {
		if err := r.WriteContextSummary(ctx.Summary); err != nil {
			return err
		}
	}
	for name, body := range ctx.Dims {
		if _, err := r.WriteContextDim(name, body); err != nil {
			return err
		}
	}
	return nil
}

func (r *Root) previewAgents(items []AgentPackage) ConfigSectionPreview {
	p := ConfigSectionPreview{Name: "agents"}
	if len(items) == 0 {
		return p
	}
	existing, _ := r.ListAgents()
	seen := map[string]bool{}
	for _, item := range existing {
		seen[item.Name] = true
	}
	for _, item := range items {
		if seen[item.Name] {
			p.Updates++
		} else {
			p.Creates++
		}
		if len(item.RedactedEnvKeys) > 0 && len(item.Env) == 0 {
			p.Warnings = append(p.Warnings, fmt.Sprintf("agent %s 的 env 已脱敏，导入时会保留本机已有 env 或留空", item.Name))
		}
	}
	return p
}

func (r *Root) previewAdventurers(items []*AdventurerFile) ConfigSectionPreview {
	p := ConfigSectionPreview{Name: "adventurers"}
	if len(items) == 0 {
		return p
	}
	existing, _ := r.ListAdventurers()
	seen := map[string]bool{}
	for _, item := range existing {
		seen[item.ID] = true
	}
	for _, item := range items {
		if item == nil {
			p.Skips++
			continue
		}
		if seen[item.ID] {
			p.Updates++
		} else {
			p.Creates++
		}
	}
	return p
}

func (r *Root) previewAutomations(items []*AutomationConfig) ConfigSectionPreview {
	p := ConfigSectionPreview{Name: "automations"}
	if len(items) == 0 {
		return p
	}
	existing, _ := r.ListAutomations()
	seen := map[string]bool{}
	for _, item := range existing {
		seen[item.ID] = true
	}
	for _, item := range items {
		if item == nil {
			p.Skips++
			continue
		}
		if seen[item.ID] {
			p.Updates++
		} else {
			p.Creates++
		}
	}
	return p
}

func (r *Root) previewContext(ctx *ContextPackage) ConfigSectionPreview {
	p := ConfigSectionPreview{Name: "context"}
	if ctx == nil {
		return p
	}
	if strings.TrimSpace(ctx.Summary) != "" {
		p.Updates++
	}
	existing, _ := r.ListContextDims()
	seen := map[string]bool{}
	for _, item := range existing {
		seen[item.Name] = true
	}
	for name := range ctx.Dims {
		if seen[name] {
			p.Updates++
		} else {
			p.Creates++
		}
	}
	return p
}

func previewTextMap(name, dir string, items map[string]string) ConfigSectionPreview {
	p := ConfigSectionPreview{Name: name}
	for rel := range items {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err == nil {
			p.Updates++
		} else {
			p.Creates++
		}
	}
	return p
}

func (r *Root) exportContext() (*ContextPackage, error) {
	summary, err := r.ReadContextSummary()
	if err != nil {
		return nil, err
	}
	dims, err := r.ListContextDims()
	if err != nil {
		return nil, err
	}
	out := &ContextPackage{Summary: summary, Dims: map[string]string{}}
	for _, item := range dims {
		dim, err := r.ReadContextDim(item.Name)
		if err != nil {
			return nil, err
		}
		out.Dims[item.Name] = dim.Body
	}
	return out, nil
}

func readTextTree(dir string) (map[string]string, error) {
	out := map[string]string{}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return out, nil
	}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > 512*1024 {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
	return out, err
}

func writeTextTree(dir string, items map[string]string) error {
	for rel, body := range items {
		if err := validateRelativePath(rel); err != nil {
			return err
		}
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func validatePathSegment(v, label string) error {
	if strings.TrimSpace(v) == "" {
		return fmt.Errorf("%s 名称不能为空", label)
	}
	if strings.ContainsAny(v, `/\`) || v == "." || v == ".." {
		return fmt.Errorf("%s 名称非法: %q", label, v)
	}
	return nil
}

func validateRelativePath(rel string) error {
	if strings.TrimSpace(rel) == "" || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, "\\") {
		return fmt.Errorf("路径非法: %q", rel)
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return fmt.Errorf("路径非法: %q", rel)
	}
	return nil
}
