package fsstore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ==================== 用户上下文（context/ 目录） ====================
//
// 用户上下文是 agent 启动前能拿到的「全局背景知识」，从本地工作区、
// 飞书等数据源提炼而来，只存摘要不存原始数据。
//
// 结构：
//   context/
//     summary.md          总览摘要（agent 第一眼看到的入口）
//     meta.json           元信息（更新时间、各维度状态）
//     dim_<name>.md       各维度详情（如 dim_workspace.md, dim_lark_im.md）
//
// 设计原则：
// - 只存摘要，不存原始数据（隐私 + 体积）
// - agent 通过 "索引 + 按需加载" 方式使用，不一次性全量注入
// - 文件全部是 Markdown + JSON，用户可手改

const (
	ContextSummaryFile       = "summary.md"
	ContextMetaFile          = "meta.json"
	ContextKnowledgeMetaFile = "knowledge_meta.json"
	ContextDimPrefix         = "dim_"
)

// ContextMeta 是上下文的元信息。
type ContextMeta struct {
	UpdatedAtMs    int64            `json:"updated_at_ms"`
	GeneratedBy    string           `json:"generated_by,omitempty"` // automation id / manual
	Dimensions     []ContextDimInfo `json:"dimensions"`
	TotalSizeBytes int64            `json:"total_size_bytes,omitempty"`
}

// ContextDimInfo 是单个维度的元信息。
type ContextDimInfo struct {
	Name        string `json:"name"`        // 维度名，如 "workspace" / "lark_im"
	Title       string `json:"title"`       // 展示名，如 "本地工作区" / "飞书群聊"
	Description string `json:"description"` // 一句话说明
	UpdatedAtMs int64  `json:"updated_at_ms"`
	SizeBytes   int64  `json:"size_bytes"`
	Source      string `json:"source,omitempty"` // 数据来源说明
}

// ContextDim 是单个维度的完整内容（元信息 + Markdown 正文）。
type ContextDim struct {
	ContextDimInfo
	Body string `json:"body"` // Markdown 正文，详情 API 需要直接返回给前端预览。
}

// KnowledgeExportFile 是一次 knowledge bundle 导出的单个文件。
type KnowledgeExportFile struct {
	Path      string `json:"path"`
	Content   string `json:"content,omitempty"`
	SizeBytes int64  `json:"size_bytes"`
	Hash      string `json:"hash,omitempty"`
}

// KnowledgeExport 是可携带的 Gloop knowledge bundle。
type KnowledgeExport struct {
	Format          string                `json:"format"`
	Generated       int64                 `json:"generated_at_ms"`
	Files           []KnowledgeExportFile `json:"files"`
	FileCount       int                   `json:"file_count"`
	TotalSizeBytes  int64                 `json:"total_size_bytes"`
	DimensionCount  int                   `json:"dimension_count"`
	SummaryIncluded bool                  `json:"summary_included"`
	Diff            *KnowledgeExportDiff  `json:"diff,omitempty"`
}

// KnowledgeExportMeta 记录最近一次 knowledge bundle 导出。
type KnowledgeExportMeta struct {
	Format         string                `json:"format"`
	ExportedAtMs   int64                 `json:"exported_at_ms"`
	Filename       string                `json:"filename,omitempty"`
	FileCount      int                   `json:"file_count"`
	TotalSizeBytes int64                 `json:"total_size_bytes"`
	DimensionCount int                   `json:"dimension_count"`
	Files          []KnowledgeExportFile `json:"files,omitempty"`
}

type KnowledgeExportDiff struct {
	Added     []KnowledgeExportFile `json:"added"`
	Changed   []KnowledgeExportFile `json:"changed"`
	Removed   []KnowledgeExportFile `json:"removed"`
	Unchanged int                   `json:"unchanged"`
}

// ==================== 查询 API ====================

// GetContextMeta 读取上下文元信息。文件不存在时返回空 meta（不是错误）。
func (r *Root) GetContextMeta() (*ContextMeta, error) {
	path := r.Sub(SubdirContext, ContextMetaFile)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ContextMeta{Dimensions: []ContextDimInfo{}}, nil
		}
		return nil, fmt.Errorf("stat context meta 失败: %w", err)
	}
	meta, err := ReadJSON[ContextMeta](path)
	if err != nil {
		// 损坏了也不报错，返回空的
		return &ContextMeta{
			UpdatedAtMs: info.ModTime().UnixMilli(),
			Dimensions:  []ContextDimInfo{},
		}, nil
	}
	if meta.Dimensions == nil {
		meta.Dimensions = []ContextDimInfo{}
	}
	// 补全总大小
	if meta.TotalSizeBytes == 0 {
		for _, d := range meta.Dimensions {
			meta.TotalSizeBytes += d.SizeBytes
		}
	}
	return meta, nil
}

// ListContextDims 列出所有上下文维度（只返回元信息，不含正文）。
func (r *Root) ListContextDims() ([]ContextDimInfo, error) {
	dir := r.Sub(SubdirContext)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ContextDimInfo{}, nil
		}
		return nil, fmt.Errorf("读取 context 目录失败: %w", err)
	}
	var out []ContextDimInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, ContextDimPrefix) || !strings.HasSuffix(name, ".md") {
			continue
		}
		dimName := strings.TrimSuffix(strings.TrimPrefix(name, ContextDimPrefix), ".md")
		info, statErr := e.Info()
		if statErr != nil {
			continue
		}
		out = append(out, ContextDimInfo{
			Name:        dimName,
			Title:       dimTitleFor(dimName),
			Description: dimDescriptionFor(dimName),
			UpdatedAtMs: info.ModTime().UnixMilli(),
			SizeBytes:   info.Size(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ReadContextDim 读取单个维度的完整内容。
func (r *Root) ReadContextDim(name string) (*ContextDim, error) {
	if name == "" {
		return nil, fmt.Errorf("维度名不能为空")
	}
	// 防目录穿越
	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return nil, fmt.Errorf("维度名非法: %q", name)
	}
	filename := ContextDimPrefix + name + ".md"
	path := r.Sub(SubdirContext, filename)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("上下文维度不存在: %s", name)
		}
		return nil, fmt.Errorf("读取上下文维度失败 %s: %w", name, err)
	}
	info, _ := os.Stat(path)
	var size int64
	var updatedAt int64
	if info != nil {
		size = info.Size()
		updatedAt = info.ModTime().UnixMilli()
	}
	return &ContextDim{
		ContextDimInfo: ContextDimInfo{
			Name:        name,
			Title:       dimTitleFor(name),
			Description: dimDescriptionFor(name),
			UpdatedAtMs: updatedAt,
			SizeBytes:   size,
		},
		Body: string(raw),
	}, nil
}

func (r *Root) RenameContextDim(oldName, newName string) error {
	if strings.TrimSpace(oldName) == "" || strings.TrimSpace(newName) == "" {
		return fmt.Errorf("维度名不能为空")
	}
	if strings.ContainsAny(oldName, `/\`) || oldName == "." || oldName == ".." {
		return fmt.Errorf("维度名非法: %q", oldName)
	}
	if strings.ContainsAny(newName, `/\`) || newName == "." || newName == ".." {
		return fmt.Errorf("维度名非法: %q", newName)
	}
	if oldName == newName {
		return nil
	}
	dir := r.Sub(SubdirContext)
	oldPath := filepath.Join(dir, ContextDimPrefix+oldName+".md")
	newPath := filepath.Join(dir, ContextDimPrefix+newName+".md")
	if _, err := os.Stat(oldPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("读取旧上下文维度失败 %s: %w", oldName, err)
	}
	if _, err := os.Stat(newPath); err == nil {
		legacyPath := filepath.Join(dir, ContextDimPrefix+oldName+".legacy.md")
		if _, statErr := os.Stat(legacyPath); statErr == nil {
			r.refreshContextMeta()
			return nil
		} else if !os.IsNotExist(statErr) {
			return fmt.Errorf("读取旧上下文维度备份失败 %s: %w", oldName, statErr)
		}
		if err := os.Rename(oldPath, legacyPath); err != nil {
			return fmt.Errorf("保留旧上下文维度失败 %s: %w", oldName, err)
		}
		r.refreshContextMeta()
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("读取新上下文维度失败 %s: %w", newName, err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("重命名上下文维度失败 %s -> %s: %w", oldName, newName, err)
	}
	r.refreshContextMeta()
	return nil
}

// ReadContextSummary 读取总览摘要。
func (r *Root) ReadContextSummary() (string, error) {
	path := r.Sub(SubdirContext, ContextSummaryFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("读取上下文摘要失败: %w", err)
	}
	return string(raw), nil
}

// BuildKnowledgeExport 以 OKF-style 目录格式导出 Gloop 用户上下文。
func (r *Root) BuildKnowledgeExport() (*KnowledgeExport, error) {
	dims, err := r.ListContextDims()
	if err != nil {
		return nil, err
	}
	summary, err := r.ReadContextSummary()
	if err != nil {
		return nil, err
	}
	meta, _ := r.GetContextMeta()
	now := time.Now().UnixMilli()
	out := &KnowledgeExport{
		Format:         "gloop.okf.v0",
		Generated:      now,
		Files:          []KnowledgeExportFile{},
		DimensionCount: len(dims),
	}
	out.addKnowledgeFile(KnowledgeExportFile{
		Path: "index.md",
		Content: renderOKFDocument(map[string]string{
			"type":        "gloop_context_bundle",
			"title":       "Gloop Knowledge Bundle",
			"description": "Portable Gloop context bundle for humans and agents.",
			"resource":    "gloop://context",
			"tags":        "gloop,context,knowledge",
			"timestamp":   formatOKFTimestamp(now),
		}, renderKnowledgeIndex(summary, dims, meta)),
	})
	if strings.TrimSpace(summary) != "" {
		out.SummaryIncluded = true
		out.addKnowledgeFile(KnowledgeExportFile{
			Path: "summary.md",
			Content: renderOKFDocument(map[string]string{
				"type":        "gloop_context_summary",
				"title":       "Gloop Context Summary",
				"description": "Cross-quest summary maintained by Gloop context refresh.",
				"resource":    "gloop://context/summary",
				"tags":        "gloop,context,summary",
				"timestamp":   formatOKFTimestamp(contextUpdatedAt(meta, now)),
			}, strings.TrimSpace(summary)+"\n"),
		})
	}
	for _, d := range dims {
		dim, readErr := r.ReadContextDim(d.Name)
		if readErr != nil {
			continue
		}
		out.addKnowledgeFile(KnowledgeExportFile{
			Path: filepath.ToSlash(filepath.Join("dimensions", safeKnowledgeFilename(dim.Name)+".md")),
			Content: renderOKFDocument(map[string]string{
				"type":        "gloop_context_dimension",
				"title":       dim.Title,
				"description": dim.Description,
				"resource":    "gloop://context/dims/" + dim.Name,
				"tags":        "gloop,context," + dim.Name,
				"timestamp":   formatOKFTimestamp(dim.UpdatedAtMs),
			}, strings.TrimSpace(dim.Body)+"\n"),
		})
	}
	return out, nil
}

func (k *KnowledgeExport) addKnowledgeFile(file KnowledgeExportFile) {
	if file.SizeBytes == 0 && file.Content != "" {
		file.SizeBytes = int64(len([]byte(file.Content)))
	}
	if file.Hash == "" && file.Content != "" {
		sum := sha256.Sum256([]byte(file.Content))
		file.Hash = hex.EncodeToString(sum[:])
	}
	k.Files = append(k.Files, file)
	k.FileCount = len(k.Files)
	k.TotalSizeBytes += file.SizeBytes
}

// KnowledgeExportPreview 返回不含正文的导出预览。
func (r *Root) KnowledgeExportPreview() (*KnowledgeExport, *KnowledgeExportMeta, error) {
	bundle, err := r.BuildKnowledgeExport()
	if err != nil {
		return nil, nil, err
	}
	preview := *bundle
	preview.Files = make([]KnowledgeExportFile, 0, len(bundle.Files))
	for _, f := range bundle.Files {
		preview.Files = append(preview.Files, KnowledgeExportFile{
			Path:      f.Path,
			SizeBytes: f.SizeBytes,
			Hash:      f.Hash,
		})
	}
	last, _ := r.ReadKnowledgeExportMeta()
	if last != nil {
		preview.Diff = diffKnowledgeFiles(last.Files, preview.Files)
	}
	return &preview, last, nil
}

func diffKnowledgeFiles(previous []KnowledgeExportFile, current []KnowledgeExportFile) *KnowledgeExportDiff {
	prevByPath := make(map[string]KnowledgeExportFile, len(previous))
	for _, f := range previous {
		prevByPath[f.Path] = f
	}
	curByPath := make(map[string]KnowledgeExportFile, len(current))
	out := &KnowledgeExportDiff{
		Added:   []KnowledgeExportFile{},
		Changed: []KnowledgeExportFile{},
		Removed: []KnowledgeExportFile{},
	}
	for _, f := range current {
		curByPath[f.Path] = f
		prev, ok := prevByPath[f.Path]
		if !ok {
			out.Added = append(out.Added, f)
			continue
		}
		if prev.SizeBytes != f.SizeBytes || prev.Hash != f.Hash {
			out.Changed = append(out.Changed, f)
			continue
		}
		out.Unchanged++
	}
	for _, f := range previous {
		if _, ok := curByPath[f.Path]; !ok {
			out.Removed = append(out.Removed, f)
		}
	}
	sort.Slice(out.Added, func(i, j int) bool { return out.Added[i].Path < out.Added[j].Path })
	sort.Slice(out.Changed, func(i, j int) bool { return out.Changed[i].Path < out.Changed[j].Path })
	sort.Slice(out.Removed, func(i, j int) bool { return out.Removed[i].Path < out.Removed[j].Path })
	return out
}

// WriteKnowledgeExport writes a knowledge bundle directory to outDir.
func (r *Root) WriteKnowledgeExport(outDir string) (*KnowledgeExport, error) {
	if strings.TrimSpace(outDir) == "" {
		return nil, fmt.Errorf("导出目录不能为空")
	}
	bundle, err := r.BuildKnowledgeExport()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建导出目录失败: %w", err)
	}
	for _, f := range bundle.Files {
		if err := validateRelativePath(f.Path); err != nil {
			return nil, err
		}
		path := filepath.Join(outDir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("创建导出子目录失败: %w", err)
		}
		if err := os.WriteFile(path, []byte(f.Content), 0o644); err != nil {
			return nil, fmt.Errorf("写入 knowledge 文件失败 %s: %w", f.Path, err)
		}
	}
	_ = r.WriteKnowledgeExportMeta("directory:"+filepath.Clean(outDir), bundle)
	return bundle, nil
}

// ReadKnowledgeExportMeta 读取最近一次 knowledge bundle 导出的元信息。
// 若从未导出过返回 (nil, nil)。
func (r *Root) ReadKnowledgeExportMeta() (*KnowledgeExportMeta, error) {
	path := r.Sub(SubdirContext, ContextKnowledgeMetaFile)
	meta, err := ReadJSON[KnowledgeExportMeta](path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return meta, nil
}

// WriteKnowledgeExportMeta 记录一次 knowledge bundle 导出的元信息。
// filename 表示导出位置（通常是 "directory:<path>" 或文件名）。
func (r *Root) WriteKnowledgeExportMeta(filename string, bundle *KnowledgeExport) error {
	if bundle == nil {
		return fmt.Errorf("knowledge bundle 不能为空")
	}
	files := make([]KnowledgeExportFile, 0, len(bundle.Files))
	for _, f := range bundle.Files {
		files = append(files, KnowledgeExportFile{
			Path:      f.Path,
			SizeBytes: f.SizeBytes,
			Hash:      f.Hash,
		})
	}
	meta := &KnowledgeExportMeta{
		Format:         bundle.Format,
		ExportedAtMs:   time.Now().UnixMilli(),
		Filename:       filename,
		FileCount:      bundle.FileCount,
		TotalSizeBytes: bundle.TotalSizeBytes,
		DimensionCount: bundle.DimensionCount,
		Files:          files,
	}
	return WriteJSON(r.Sub(SubdirContext, ContextKnowledgeMetaFile), meta)
}

// ==================== 写入 API ====================

// WriteContextDim 写入（创建或覆盖）一个维度的 Markdown 正文。
func (r *Root) WriteContextDim(name, body string) (*ContextDimInfo, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("维度名不能为空")
	}
	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return nil, fmt.Errorf("维度名非法: %q", name)
	}
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("维度内容不能为空")
	}
	dir := r.Sub(SubdirContext)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建 context 目录失败: %w", err)
	}
	filename := ContextDimPrefix + name + ".md"
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return nil, fmt.Errorf("写入上下文维度失败 %s: %w", name, err)
	}
	info, _ := os.Stat(path)
	var size int64
	var updatedAt int64
	if info != nil {
		size = info.Size()
		updatedAt = info.ModTime().UnixMilli()
	}
	// 同步刷新 meta
	r.refreshContextMeta()
	return &ContextDimInfo{
		Name:        name,
		Title:       dimTitleFor(name),
		Description: dimDescriptionFor(name),
		UpdatedAtMs: updatedAt,
		SizeBytes:   size,
	}, nil
}

// WriteContextSummary 写入总览摘要。
func (r *Root) WriteContextSummary(body string) error {
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("摘要内容不能为空")
	}
	dir := r.Sub(SubdirContext)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建 context 目录失败: %w", err)
	}
	path := filepath.Join(dir, ContextSummaryFile)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("写入上下文摘要失败: %w", err)
	}
	r.refreshContextMeta()
	return nil
}

// SetContextGeneratedBy 标记本次上下文是由谁生成的（写进 meta）。
func (r *Root) SetContextGeneratedBy(by string) error {
	meta, err := r.GetContextMeta()
	if err != nil {
		return err
	}
	meta.GeneratedBy = by
	meta.UpdatedAtMs = time.Now().UnixMilli()
	return r.saveContextMeta(meta)
}

// ==================== 内部工具 ====================

func (r *Root) saveContextMeta(meta *ContextMeta) error {
	if meta == nil {
		return fmt.Errorf("meta 不能为空")
	}
	dir := r.Sub(SubdirContext)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建 context 目录失败: %w", err)
	}
	path := filepath.Join(dir, ContextMetaFile)
	return WriteJSON(path, meta)
}

// refreshContextMeta 从磁盘文件重新计算 meta 并保存。
func (r *Root) refreshContextMeta() {
	dims, err := r.ListContextDims()
	if err != nil {
		return
	}
	meta := &ContextMeta{
		UpdatedAtMs: time.Now().UnixMilli(),
		Dimensions:  dims,
	}
	// 保留 generated_by
	old, err := r.GetContextMeta()
	if err == nil && old != nil {
		meta.GeneratedBy = old.GeneratedBy
	}
	var total int64
	for _, d := range dims {
		total += d.SizeBytes
	}
	meta.TotalSizeBytes = total
	_ = r.saveContextMeta(meta)
}

// dimTitleFor 给出维度的默认展示名。
func dimTitleFor(name string) string {
	switch name {
	case "identity":
		return "用户身份"
	case "workspace":
		return "本地工作区"
	case "tooling":
		return "工具与环境"
	case "lark_im":
		return "飞书聊天记录"
	case "lark_doc":
		return "飞书文档"
	case "lark_calendar":
		return "工作节奏"
	case "gloop_history":
		return "工作模式"
	case "codebase_map":
		return "代码地图"
	case "activity_snapshot":
		return "活动快照"
	default:
		return name
	}
}

// dimDescriptionFor 给出维度的默认描述。
func dimDescriptionFor(name string) string {
	switch name {
	case "identity":
		return "用户身份、角色、团队归属与权限边界"
	case "workspace":
		return "从本地 Git 仓库和项目文件提炼的项目上下文"
	case "tooling":
		return "可用工具链、服务地址、部署环境与基础设施"
	case "lark_im":
		return "从飞书聊天记录（群聊/私聊/话题群）提炼的近期讨论主题和决策"
	case "lark_doc":
		return "从飞书云文档提炼的相关文档摘要"
	case "lark_calendar":
		return "从飞书日历提炼的工作节奏、会议模式与协作规律"
	case "gloop_history":
		return "从历史委托中提炼的个人工作模式偏好（任务拆解风格、验证习惯、验收口径）"
	case "codebase_map":
		return "项目代码结构地图：目录树、核心模块、关键接口与架构模式，供快速定位代码"
	case "activity_snapshot":
		return "Gloop 全局活动快照：开放委托、阻塞项与近期完成情况"
	default:
		return "用户上下文维度"
	}
}

func renderKnowledgeIndex(summary string, dims []ContextDimInfo, meta *ContextMeta) string {
	var b strings.Builder
	b.WriteString("# Gloop Knowledge Bundle\n\n")
	b.WriteString("This bundle exports Gloop context as markdown files with YAML frontmatter. It is intended to be readable by humans and parseable by agents.\n\n")
	if meta != nil && meta.GeneratedBy != "" {
		b.WriteString("- Generated by: `" + meta.GeneratedBy + "`\n")
	}
	if meta != nil && meta.UpdatedAtMs > 0 {
		b.WriteString("- Context updated: " + formatOKFTimestamp(meta.UpdatedAtMs) + "\n")
	}
	b.WriteString("- Dimensions: ")
	b.WriteString(fmt.Sprintf("%d\n\n", len(dims)))
	if strings.TrimSpace(summary) != "" {
		b.WriteString("## Summary\n\n")
		b.WriteString(strings.TrimSpace(summary))
		b.WriteString("\n\n")
	}
	b.WriteString("## Dimensions\n\n")
	if len(dims) == 0 {
		b.WriteString("_No context dimensions exported._\n")
		return b.String()
	}
	for _, d := range dims {
		b.WriteString("- [")
		b.WriteString(d.Title)
		b.WriteString("](dimensions/")
		b.WriteString(safeKnowledgeFilename(d.Name))
		b.WriteString(".md) - `")
		b.WriteString(d.Name)
		b.WriteString("`")
		if d.Description != "" {
			b.WriteString(": ")
			b.WriteString(d.Description)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func renderOKFDocument(frontmatter map[string]string, body string) string {
	keys := []string{"type", "title", "description", "resource", "tags", "timestamp"}
	var b strings.Builder
	b.WriteString("---\n")
	for _, k := range keys {
		v := frontmatter[k]
		if strings.TrimSpace(v) == "" {
			continue
		}
		b.WriteString(k)
		b.WriteString(": ")
		if k == "tags" {
			b.WriteString("[")
			parts := strings.Split(v, ",")
			wrote := false
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				if wrote {
					b.WriteString(", ")
				}
				b.WriteString(yamlQuote(p))
				wrote = true
			}
			b.WriteString("]\n")
			continue
		}
		b.WriteString(yamlQuote(v))
		b.WriteString("\n")
	}
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimSpace(body))
	b.WriteString("\n")
	return b.String()
}

func yamlQuote(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return `"` + v + `"`
}

func formatOKFTimestamp(ms int64) string {
	if ms <= 0 {
		ms = time.Now().UnixMilli()
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}

func contextUpdatedAt(meta *ContextMeta, fallback int64) int64 {
	if meta != nil && meta.UpdatedAtMs > 0 {
		return meta.UpdatedAtMs
	}
	return fallback
}

func safeKnowledgeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "untitled"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "._-")
	if out == "" {
		return "untitled"
	}
	return out
}
