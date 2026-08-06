package prompt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"

	builtinprompts "code.byted.org/lihuanyu.0w0/gloop/prompts"
)

// TemplateStore 加载 prompt 模板。
//
// 设计为两层叠加：
//   - base: 内建默认模板（embed 进二进制），保证缺失时总能降级
//   - overlay: 用户目录下的覆盖模板（~/.gloop/prompts/），用户可手改、重启生效
//
// 查找顺序：先 overlay，再 base。overlay 缺的文件自动 fallback 到 base。
type TemplateStore struct {
	base    fs.FS
	overlay fs.FS // 可为 nil

	mu             sync.Mutex
	cache          map[string]string
	hitBase        map[string]bool // 命中 base 的 key（overlay 里没这个文件）
	baseHashCached string          // base 层整体哈希缓存，计算过一次后复用
}

var defaultStore *TemplateStore
var defaultStoreOnce sync.Once

// DefaultTemplateStore 返回进程级单例，仅使用内建模板。
// 调用方如果需要用户目录覆盖，自己构造带 overlay 的 store 并通过 WithTemplateStore 注入。
func DefaultTemplateStore() *TemplateStore {
	defaultStoreOnce.Do(func() {
		defaultStore = NewTemplateStore(builtinprompts.FS(), nil)
	})
	return defaultStore
}

// NewTemplateStore 构造一个模板存储。
// base 必须非空；overlay 可为 nil（表示只使用 base）。
func NewTemplateStore(base fs.FS, overlay fs.FS) *TemplateStore {
	if base == nil {
		panic("TemplateStore base FS is nil")
	}
	return &TemplateStore{
		base:    base,
		overlay: overlay,
		cache:   map[string]string{},
		hitBase: map[string]bool{},
	}
}

// Get 读取指定模板内容。name 是相对路径，例如 "warrior_role.md" 或 "blocks/design_note.md"。
// 内容会去掉首尾空白行，但保留中间的空行和格式。
func (s *TemplateStore) Get(name string) (string, error) {
	s.mu.Lock()
	if cached, ok := s.cache[name]; ok {
		s.mu.Unlock()
		return cached, nil
	}
	s.mu.Unlock()

	// 先查 overlay
	content, err := s.readFile(s.overlay, name)
	if err == nil {
		result := strings.TrimSpace(content) + "\n"
		s.mu.Lock()
		s.cache[name] = result
		s.mu.Unlock()
		return result, nil
	}

	// 再查 base
	content, err = s.readFile(s.base, name)
	if err != nil {
		return "", fmt.Errorf("模板 %q 不存在（base + overlay 都没找到）: %w", name, err)
	}
	result := strings.TrimSpace(content) + "\n"
	s.mu.Lock()
	s.cache[name] = result
	s.hitBase[name] = true
	s.mu.Unlock()
	return result, nil
}

// IsOverridden 返回模板是否被 overlay 覆盖（即用户目录里有同名文件）。
// 主要用于测试和调试输出。
func (s *TemplateStore) IsOverridden(name string) bool {
	// 触发一次加载确保缓存
	if _, err := s.Get(name); err != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.hitBase[name]
}

// PromptVersionInfo 是 prompt 模板的版本快照，用于 quest 归因。
//
// 每个 quest 创建时记录当时的 prompt 版本，后续排查问题时可以追溯
// "这个 quest 是用哪套 prompt 跑的"，避免 prompt 改动后历史 quest 不可复盘。
type PromptVersionInfo struct {
	// BaseHash 是内建模板（base 层）的整体内容哈希。
	// 同一份二进制的内建模板总是相同，不同版本的二进制哈希不同。
	BaseHash string `json:"base_hash"`

	// Overrides 是被用户目录（overlay 层）覆盖的模板文件列表。
	// 列表已排序，方便直接对比。
	Overrides []string `json:"overrides,omitempty"`

	// OverrideHashes 是被覆盖模板各自的内容哈希，key = 模板路径。
	// 用于细粒度对比用户改了哪些模板、改动幅度多大。
	// 仅当 Overrides 非空时有值。
	OverrideHashes map[string]string `json:"override_hashes,omitempty"`
}

// Short 返回简洁的版本描述（base hash 前 8 位 + override 数量）。
func (v PromptVersionInfo) Short() string {
	base := v.BaseHash
	if len(base) > 8 {
		base = base[:8]
	}
	if len(v.Overrides) == 0 {
		return fmt.Sprintf("base-%s", base)
	}
	return fmt.Sprintf("base-%s+%doverrides", base, len(v.Overrides))
}

// Version 计算当前模板存储的版本快照。
//
// base 层哈希在同一份二进制内是稳定的，可缓存；overlay 层因为用户可能
// 在运行中修改文件，每次调用都会重新扫描（但内容走缓存，开销可接受）。
func (s *TemplateStore) Version() PromptVersionInfo {
	// base hash 是稳定的，缓存一下
	s.mu.Lock()
	baseHash := s.baseHashCached
	s.mu.Unlock()

	if baseHash == "" {
		baseHash = s.computeBaseHash()
		s.mu.Lock()
		s.baseHashCached = baseHash
		s.mu.Unlock()
	}

	info := PromptVersionInfo{BaseHash: baseHash}

	// overlay 层：列出所有 .md 文件并计算各自哈希
	if s.overlay != nil {
		overrides, hashes := s.computeOverlayInfo()
		if len(overrides) > 0 {
			info.Overrides = overrides
			info.OverrideHashes = hashes
		}
	}

	return info
}

// computeBaseHash 计算 base 层所有模板的整体哈希。
// 按文件名排序后拼接内容再 sha256，确保确定性。
func (s *TemplateStore) computeBaseHash() string {
	files, err := s.ListBase()
	if err != nil {
		return "unknown"
	}
	sort.Strings(files)

	h := sha256.New()
	for _, f := range files {
		content, err := s.Get(f)
		if err != nil {
			continue
		}
		h.Write([]byte(f))
		h.Write([]byte{0})
		h.Write([]byte(content))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// computeOverlayInfo 返回 overlay 层覆盖的模板列表（排序）和各文件哈希。
func (s *TemplateStore) computeOverlayInfo() ([]string, map[string]string) {
	var files []string
	hashes := map[string]string{}

	err := fs.WalkDir(s.overlay, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		// 只有同时在 base 里存在的才算 override（纯新增的自定义模板单独管理）
		if _, err := s.readFile(s.base, path); err != nil {
			return nil
		}
		files = append(files, path)

		// 用缓存里的内容算哈希（或直接读文件）
		content, err := s.Get(path)
		if err == nil {
			sum := sha256.Sum256([]byte(content))
			hashes[path] = hex.EncodeToString(sum[:])
		}
		return nil
	})
	if err != nil {
		return files, hashes
	}
	sort.Strings(files)
	return files, hashes
}

// ListBase 返回 base 层所有可用的模板文件路径（相对于 base 根）。
// 用于调试 / 文档生成。
func (s *TemplateStore) ListBase() ([]string, error) {
	var names []string
	err := fs.WalkDir(s.base, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".md") {
			names = append(names, path)
		}
		return nil
	})
	return names, err
}

// TemplateInfo 是单个 prompt 模板的元信息（list 用）。
type TemplateInfo struct {
	Name         string `json:"name"`          // 模板路径，如 "warrior_role.md" 或 "blocks/intensity_quick.md"
	Source       string `json:"source"`        // "builtin" 或 "user"
	SizeBytes    int    `json:"size_bytes"`    // 内容字节数
	Preview      string `json:"preview"`       // 内容预览（前 N 个字符，不含首尾空白）
	IsOverridden bool   `json:"is_overridden"` // 是否被用户目录覆盖
}

// ListTemplates 返回所有可用模板的元信息列表。
// 列表包含 base 层全部模板 + overlay 层新增的模板。
// 结果按 name 字典序排序。
func (s *TemplateStore) ListTemplates() ([]TemplateInfo, error) {
	// 收集 base 层模板
	baseFiles, err := s.ListBase()
	if err != nil {
		return nil, fmt.Errorf("列出 base 模板失败: %w", err)
	}
	names := map[string]bool{}
	for _, f := range baseFiles {
		names[f] = true
	}

	// 收集 overlay 层模板（可能包含 base 没有的新增文件）
	if s.overlay != nil {
		err := fs.WalkDir(s.overlay, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			names[path] = true
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("列出 overlay 模板失败: %w", err)
		}
	}

	// 转成有序列表
	var sorted []string
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)

	out := make([]TemplateInfo, 0, len(sorted))
	for _, name := range sorted {
		content, err := s.Get(name)
		if err != nil {
			continue
		}
		overridden := s.IsOverridden(name)
		source := "builtin"
		if overridden {
			source = "user"
		} else if _, err := s.readFile(s.base, name); err != nil {
			// base 里没有、overlay 里加的
			source = "user"
		}
		preview := strings.TrimSpace(content)
		if len(preview) > 80 {
			preview = preview[:80] + "…"
		}
		out = append(out, TemplateInfo{
			Name:         name,
			Source:       source,
			SizeBytes:    len(content),
			Preview:      preview,
			IsOverridden: overridden,
		})
	}
	return out, nil
}

// GetTemplate 返回单个模板的完整内容。
// name 是模板路径，如 "warrior_role.md"。
func (s *TemplateStore) GetTemplate(name string) (content string, info TemplateInfo, err error) {
	content, err = s.Get(name)
	if err != nil {
		return
	}
	overridden := s.IsOverridden(name)
	source := "builtin"
	if overridden {
		source = "user"
	}
	info = TemplateInfo{
		Name:         name,
		Source:       source,
		SizeBytes:    len(content),
		Preview:      "",
		IsOverridden: overridden,
	}
	return
}

func (s *TemplateStore) readFile(fsys fs.FS, name string) (string, error) {
	if fsys == nil {
		return "", fmt.Errorf("fs is nil")
	}
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// renderSimple 做极简变量替换：{{key}} → value。
// 故意不引入完整模板引擎，保持 prompt 模板简单、可预测。
// 如果变量不存在，原样保留（方便调试）。
func renderSimple(tmpl string, vars map[string]string) string {
	if len(vars) == 0 {
		return tmpl
	}
	out := tmpl
	for k, v := range vars {
		placeholder := "{{" + k + "}}"
		out = strings.ReplaceAll(out, placeholder, v)
	}
	return out
}
