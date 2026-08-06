package prompt

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// TrustLevel marks how a model should treat a block. It is prompt metadata, not
// a security boundary; platform enforcement still lives in orchestrator/tools.
type TrustLevel string

const (
	TrustPlatform    TrustLevel = "platform"
	TrustUser        TrustLevel = "user"
	TrustAgentOutput TrustLevel = "agent_output"
	TrustMixed       TrustLevel = "mixed" // 块内混合了多种信任来源，需自行甄别
)

// ContextBlock is the smallest renderable unit in a prompt context pack.
type ContextBlock struct {
	Name           string     `json:"name"`
	Source         string     `json:"source,omitempty"`
	Trust          TrustLevel `json:"trust,omitempty"`
	Phase          string     `json:"phase,omitempty"`
	Role           string     `json:"role,omitempty"`
	Staleness      string     `json:"staleness,omitempty"`
	Priority       int        `json:"priority,omitempty"`
	UserControlled bool       `json:"user_controlled"`
	Content        string     `json:"content"`
}

// ContextPack is Gloop's prompt-side IR. It keeps source/trust metadata explicit
// before the pack is rendered into a agent-specific text shape.
type ContextPack struct {
	Kind   string         `json:"kind"`
	Blocks []ContextBlock `json:"blocks"`
}

type RenderOptions struct {
	MaxBlockChars int
	MaxTotalChars int
}

type ContextBlockSummary struct {
	Name           string     `json:"name"`
	Source         string     `json:"source,omitempty"`
	Trust          TrustLevel `json:"trust,omitempty"`
	Phase          string     `json:"phase,omitempty"`
	Role           string     `json:"role,omitempty"`
	Staleness      string     `json:"staleness,omitempty"`
	Priority       int        `json:"priority,omitempty"`
	UserControlled bool       `json:"user_controlled"`
	Chars          int        `json:"chars"`
	Truncated      bool       `json:"truncated,omitempty"`
	Preview        string     `json:"preview,omitempty"`
}

type ContextPackSummary struct {
	Kind          string                `json:"kind"`
	Blocks        []ContextBlockSummary `json:"blocks"`
	OriginalChars int                   `json:"original_chars"`
	RenderedChars int                   `json:"rendered_chars"`
	Truncated     bool                  `json:"truncated,omitempty"`
}

const (
	DefaultMaxBlockChars = 16000
	DefaultMaxPackChars  = 64000
	DefaultPreviewChars  = 240
	// FileContextDirName 是 .gloop/ 下存放文件式上下文的子目录名
	FileContextDirName = "context"
)

var DefaultRenderOptions = RenderOptions{
	MaxBlockChars: DefaultMaxBlockChars,
	MaxTotalChars: DefaultMaxPackChars,
}

func (p ContextPack) RenderXMLish() string {
	return p.RenderXMLishWithOptions(DefaultRenderOptions)
}

func (p ContextPack) RenderXMLishWithOptions(opt RenderOptions) string {
	return p.WithBudget(opt).renderXMLish()
}

// WithBudget 返回按预算裁剪后的 ContextPack。
// 裁剪后的 blocks content 是实际发给 agent 的版本，
// 用于持久化和展示时与 agent 实际收到的内容保持一致。
func (p ContextPack) WithBudget(opt RenderOptions) ContextPack {
	return p.withBudget(opt)
}

func (p ContextPack) Summary(opt RenderOptions) ContextPackSummary {
	trimmed := p.withBudget(opt)
	summary := ContextPackSummary{
		Kind:          p.Kind,
		OriginalChars: p.totalChars(),
		RenderedChars: trimmed.totalChars(),
		Truncated:     trimmed.totalChars() != p.totalChars(),
	}
	for i, block := range trimmed.Blocks {
		origChars := 0
		if i < len(p.Blocks) {
			origChars = charCount(p.Blocks[i].Content)
		}
		chars := charCount(block.Content)
		summary.Blocks = append(summary.Blocks, ContextBlockSummary{
			Name:           block.Name,
			Source:         block.Source,
			Trust:          block.Trust,
			Phase:          block.Phase,
			Role:           block.Role,
			Staleness:      block.Staleness,
			Priority:       block.Priority,
			UserControlled: block.UserControlled,
			Chars:          chars,
			Truncated:      chars != origChars,
			Preview:        truncateRunes(block.Content, DefaultPreviewChars),
		})
	}
	return summary
}

func (p ContextPack) renderXMLish() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("<gloop_context kind=\"%s\">\n", xmlAttr(p.Kind)))
	for _, block := range p.Blocks {
		b.WriteString("  <block")
		b.WriteString(fmt.Sprintf(" name=\"%s\"", xmlAttr(block.Name)))
		if block.Source != "" {
			b.WriteString(fmt.Sprintf(" source=\"%s\"", xmlAttr(block.Source)))
		}
		if block.Trust != "" {
			b.WriteString(fmt.Sprintf(" trust=\"%s\"", xmlAttr(string(block.Trust))))
		}
		if block.Phase != "" {
			b.WriteString(fmt.Sprintf(" phase=\"%s\"", xmlAttr(block.Phase)))
		}
		b.WriteString(fmt.Sprintf(" user_controlled=\"%s\"", boolAttr(block.UserControlled)))
		if block.Role != "" {
			b.WriteString(fmt.Sprintf(" role=\"%s\"", xmlAttr(block.Role)))
		}
		if block.Staleness != "" {
			b.WriteString(fmt.Sprintf(" staleness=\"%s\"", xmlAttr(block.Staleness)))
		}
		if block.Priority != 0 {
			b.WriteString(fmt.Sprintf(" priority=\"%d\"", block.Priority))
		}
		b.WriteString(">\n")
		b.WriteString(escapeText(block.Content))
		if !strings.HasSuffix(block.Content, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("  </block>\n")
	}
	b.WriteString("</gloop_context>")
	return b.String()
}

func (p ContextPack) withBudget(opt RenderOptions) ContextPack {
	maxBlock := opt.MaxBlockChars
	maxTotal := opt.MaxTotalChars
	if maxBlock < 0 {
		maxBlock = 0
	}
	if maxTotal < 0 {
		maxTotal = 0
	}
	out := ContextPack{Kind: p.Kind, Blocks: make([]ContextBlock, 0, len(p.Blocks))}
	remaining := maxTotal
	for _, block := range p.Blocks {
		next := block
		limit := maxBlock
		if remaining > 0 && (limit == 0 || remaining < limit) {
			limit = remaining
		}
		if limit > 0 {
			next.Content = truncateWithMarker(next.Content, limit)
		}
		out.Blocks = append(out.Blocks, next)
		if remaining > 0 {
			remaining -= charCount(next.Content)
			if remaining < 0 {
				remaining = 0
			}
		}
	}
	return out
}

func (p ContextPack) totalChars() int {
	total := 0
	for _, block := range p.Blocks {
		total += charCount(block.Content)
	}
	return total
}

// ==================== 能力优先模式（Capability-First Mode） ====================
//
// 核心思路：不给 agent 灌所有上下文，只给最核心的 + 一份"上下文目录"，
// agent 需要时主动查询具体 block 的内容。
//
// 设计目标：
//   - 减少首轮 prompt 的 token 开销
//   - 让 agent 只拉取它真正需要的信息
//   - 保留完整的可追溯性和信任标签

// CapabilityModeConfig 定义能力优先模式的配置。
type CapabilityModeConfig struct {
	// CoreBlocks 是始终直接注入的核心 block 名称列表
	CoreBlocks []string
	// CatalogBlockName 是"上下文目录"block 的名称
	CatalogBlockName string
}

// DefaultWarriorCoreBlocks 剑士侧默认始终注入的核心 blocks。
var DefaultWarriorCoreBlocks = []string{
	"phase_control_panel",
	"quest_user_intent",
	"execution_instruction",
}

// DefaultMageCoreBlocks 法师侧默认始终注入的核心 blocks。
var DefaultMageCoreBlocks = []string{
	"review_control_panel",
	"quest_user_intent",
	"warrior_artifact",
	"review_protocol",
}

// ToCapabilityMode 将完整 ContextPack 转换为能力优先模式。
// 只保留 coreBlocks 指定的核心 block，其余的在 catalog 中列出，
// 可通过 QueryBlock 按需查询。
func (p ContextPack) ToCapabilityMode(cfg CapabilityModeConfig) ContextPack {
	if cfg.CatalogBlockName == "" {
		cfg.CatalogBlockName = "context_catalog"
	}

	coreSet := make(map[string]bool)
	for _, name := range cfg.CoreBlocks {
		coreSet[name] = true
	}

	var coreBlocks []ContextBlock
	var queryable []ContextBlock

	for _, block := range p.Blocks {
		if coreSet[block.Name] {
			coreBlocks = append(coreBlocks, block)
		} else {
			queryable = append(queryable, block)
		}
	}

	// 构建目录 block
	if len(queryable) > 0 {
		catalogLines := []string{
			"以下上下文块可按需查询，调用方式：gloop context get <block_name>",
			"",
			"可用上下文列表：",
		}
		for _, b := range queryable {
			preview := truncateRunes(b.Content, 80)
			preview = strings.ReplaceAll(preview, "\n", " ")
			catalogLines = append(catalogLines,
				fmt.Sprintf("- **%s** (source=%s, trust=%s, role=%s, staleness=%s) — 预览：%s",
					b.Name, b.Source, b.Trust, b.Role, b.Staleness, preview))
		}
		catalogLines = append(catalogLines,
			"",
			"说明：",
			"- 需要某个上下文的完整内容时，主动查询即可",
			"- 只查你真正需要的，避免不必要的信息过载",
			"- 所有查询结果都带有 source/trust 标签，请注意甄别")

		coreBlocks = append(coreBlocks, ContextBlock{
			Name:    cfg.CatalogBlockName,
			Source:  "gloop",
			Trust:   TrustPlatform,
			Phase:   p.Blocks[0].Phase,
			Content: strings.Join(catalogLines, "\n"),
		})
	}

	return ContextPack{
		Kind:   p.Kind + "_capability_first",
		Blocks: coreBlocks,
	}
}

// QueryBlock 按名称查询 block 内容（用于能力优先模式下的主动获取）。
func (p ContextPack) QueryBlock(name string) (ContextBlock, bool) {
	for _, b := range p.Blocks {
		if b.Name == name {
			return b, true
		}
	}
	return ContextBlock{}, false
}

// QueryableBlocks 返回所有可查询的 block 名称列表（即不在核心集中的 blocks）。
func (p ContextPack) QueryableBlocks(coreNames []string) []string {
	coreSet := make(map[string]bool)
	for _, name := range coreNames {
		coreSet[name] = true
	}
	var result []string
	for _, b := range p.Blocks {
		if !coreSet[b.Name] {
			result = append(result, b.Name)
		}
	}
	return result
}

// ==================== 文件式上下文（File-Mode Context） ====================
//
// 设计思路：不把上下文直接注入 prompt，而是写成文件放在 .gloop/context/ 目录下，
// 只在 prompt 中放一份目录和使用说明。agent 需要时自己用 CLI / shell 命令读取。
//
// 核心假设：主动检索比被动注入更高效 ——
//   - agent 只看它需要的，注意力更集中
//   - 减少首轮 prompt token
//   - 保留完整可追溯性（source/trust 元数据写在文件头部）

// WriteFileContext 将 ContextPack 中的非核心 blocks 写成文件，放在 workDir/.gloop/context/ 下。
// coreBlocks 始终注入 prompt，其余 blocks 写成文件。
// 返回文件模式下的 ContextPack（只含 coreBlocks + 一个 context_files 目录说明块）。
func (p ContextPack) WriteFileContext(workDir string, gloopDirName string, coreBlocks []string) (ContextPack, error) {
	ctxDir := filepath.Join(workDir, gloopDirName, FileContextDirName)
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		return ContextPack{}, fmt.Errorf("create context dir: %w", err)
	}

	coreSet := make(map[string]bool)
	for _, name := range coreBlocks {
		coreSet[name] = true
	}

	var keptBlocks []ContextBlock
	var fileBlocks []ContextBlock
	for _, b := range p.Blocks {
		if coreSet[b.Name] || shouldInlinePrimaryInput(b) {
			keptBlocks = append(keptBlocks, b)
		} else {
			fileBlocks = append(fileBlocks, b)
		}
	}

	if len(fileBlocks) == 0 {
		return p, nil
	}

	// 写每个 block 为独立文件
	for _, b := range fileBlocks {
		filename := blockToFilename(b.Name)
		path := filepath.Join(ctxDir, filename)
		content := renderBlockFile(b)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return ContextPack{}, fmt.Errorf("write context file %s: %w", filename, err)
		}
	}

	// 构建目录说明 block
	catalogLines := []string{
		"以下上下文块已写成文件，位于 .gloop/context/ 目录下：",
		"",
		"可用上下文列表：",
	}
	for _, b := range fileBlocks {
		filename := blockToFilename(b.Name)
		preview := truncateRunes(b.Content, 80)
		preview = strings.ReplaceAll(preview, "\n", " ")
		catalogLines = append(catalogLines,
			fmt.Sprintf("- **%s** (source=%s, trust=%s, role=%s, staleness=%s) — 文件: %s — 预览：%s",
				b.Name, b.Source, b.Trust, b.Role, b.Staleness, filename, preview))
	}
	catalogLines = append(catalogLines,
		"",
		"读取方式：",
		"- 使用 shell 命令: cat .gloop/context/<filename>",
		"- 或使用 gloop context get <block_name>",
		"",
		"说明：",
		"- 需要某个上下文的完整内容时，主动读取对应文件即可",
		"- 只看你真正需要的，避免不必要的信息过载",
		"- 每个文件开头都有 source/trust 元数据，请注意甄别",
	)

	keptBlocks = append(keptBlocks, ContextBlock{
		Name:    "context_files",
		Source:  "gloop",
		Trust:   TrustPlatform,
		Phase:   p.Blocks[0].Phase,
		Content: strings.Join(catalogLines, "\n"),
	})

	return ContextPack{
		Kind:   p.Kind + "_file_mode",
		Blocks: keptBlocks,
	}, nil
}

func shouldInlinePrimaryInput(b ContextBlock) bool {
	if b.Role != "primary_input" {
		return false
	}
	switch b.Trust {
	case TrustUser, TrustAgentOutput:
		return true
	default:
		return false
	}
}

// blockToFilename 将 block 名称转为安全的文件名。
func blockToFilename(name string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '_' || r == '-':
			return r
		default:
			return '_'
		}
	}, name)
	return safe + ".md"
}

// renderBlockFile 渲染单个 block 为文件内容（带元数据头）。
func renderBlockFile(b ContextBlock) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", b.Name))
	sb.WriteString(fmt.Sprintf("source: %s\n", b.Source))
	sb.WriteString(fmt.Sprintf("trust: %s\n", b.Trust))
	if b.Phase != "" {
		sb.WriteString(fmt.Sprintf("phase: %s\n", b.Phase))
	}
	if b.Role != "" {
		sb.WriteString(fmt.Sprintf("role: %s\n", b.Role))
	}
	if b.Staleness != "" {
		sb.WriteString(fmt.Sprintf("staleness: %s\n", b.Staleness))
	}
	if b.Priority != 0 {
		sb.WriteString(fmt.Sprintf("priority: %d\n", b.Priority))
	}
	sb.WriteString(fmt.Sprintf("user_controlled: %v\n", b.UserControlled))
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s\n\n", b.Name))
	sb.WriteString(b.Content)
	if !strings.HasSuffix(b.Content, "\n") {
		sb.WriteString("\n")
	}
	return sb.String()
}

func charCount(s string) int {
	return utf8.RuneCountInString(s)
}

func truncateWithMarker(s string, limit int) string {
	if limit <= 0 || charCount(s) <= limit {
		return s
	}
	marker := "\n[...context block truncated by gloop budget...]\n"
	markerChars := charCount(marker)
	if limit <= markerChars {
		return truncateRunes(marker, limit)
	}
	return truncateRunes(s, limit-markerChars) + marker
}

func truncateRunes(s string, limit int) string {
	if limit <= 0 || charCount(s) <= limit {
		return s
	}
	runes := []rune(s)
	return string(runes[:limit])
}

func escapeText(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func xmlAttr(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	).Replace(s)
}

func boolAttr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
