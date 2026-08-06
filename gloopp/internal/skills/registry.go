package skills

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	builtinskills "code.byted.org/lihuanyu.0w0/gloop/skills"
)

// ==================== 技能体系（方向 2：渐进式披露 + 官方内置只读） ====================
//
// Skill 是 agent-side instruction package。gloop 只负责分发和索引，不解释 skill 语义，
// 不根据 skill 替 agent 选择工具，也不参与业务决策。
//
// 格式严格对齐 douyin-cli 官方 agent skill 规范：
//   - 每个 skill 一个子目录，入口文件是 SKILL.md
//   - 头部是 YAML frontmatter（`---` 分隔），包含 name/version/description/metadata
//   - 正文是 Markdown，至少包含 `# <name>`、`## Trigger Examples`、`## CLI Contract`、
//     `## Discipline` 四个章节
//
// 设计原则：
// - 随二进制 embed 发行，只由 gloop 官方维护，用户不可修改/安装/卸载
//   （冒险者自身的能力扩展走 Adventurer.CustomPrompt，与平台职责无关）
// - 启动时只注入职业可用 skill manifest，正文通过 `gloop skill show` 按需拉取

// skillFrontmatter 对齐 douyin-cli 官方 SKILL.md 的 YAML 头部结构。
// gloop 私有扩展放在 metadata 下。
type skillFrontmatter struct {
	Name        string
	Version     string
	Description string
	Metadata    skillMeta
}

type skillMeta struct {
	Requires skillMetaRequires
	// gloop 私有扩展：职业权限（warrior | mage | both | 空=both）、分类、种类
	Class         string
	Category      string
	Kind          string // atomic | orchestration | informational | utility
	RelatedSkills []RelatedSkill
}

type skillMetaRequires struct {
	Bins    []string
	CliHelp string
}

// RelatedSkill describes a relationship to another skill.
type RelatedSkill struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// Skill 一个技能包（元数据 + Markdown 正文）。
type Skill struct {
	Name             string
	Version          string
	Description      string // frontmatter 里的一段高密度描述（含触发场景+反例）
	Category         string
	Kind             string // atomic | orchestration | informational | utility
	RequiresBins     []string
	RequiresCliHelp  string
	WarriorAvailable bool
	MageAvailable    bool
	RelatedSkills    []RelatedSkill
	Body             string // 正文（去掉 frontmatter 后的完整 Markdown）
}

// Manifest 是可安全暴露到 prompt / list API 的 skill 元数据。
// 它刻意不包含 Body，保证 progressive disclosure 只能通过 show 显式加载正文。
type Manifest struct {
	Name             string         `json:"name"`
	Version          string         `json:"version"`
	Description      string         `json:"description"`
	Category         string         `json:"category,omitempty"`
	Kind             string         `json:"kind,omitempty"`
	RequiresBins     []string       `json:"requires_bins,omitempty"`
	RequiresCliHelp  string         `json:"requires_cli_help,omitempty"`
	WarriorAvailable bool           `json:"warrior_available"`
	MageAvailable    bool           `json:"mage_available"`
	RelatedSkills    []RelatedSkill `json:"related_skills,omitempty"`
}

func (sk Skill) Manifest() Manifest {
	return Manifest{
		Name:             sk.Name,
		Version:          sk.Version,
		Description:      sk.Description,
		Category:         sk.Category,
		Kind:             sk.Kind,
		RequiresBins:     append([]string(nil), sk.RequiresBins...),
		RequiresCliHelp:  sk.RequiresCliHelp,
		WarriorAvailable: sk.WarriorAvailable,
		MageAvailable:    sk.MageAvailable,
		RelatedSkills:    append([]RelatedSkill(nil), sk.RelatedSkills...),
	}
}

// Registry 技能注册表：启动时从 fs.FS 一次性解析。
type Registry struct {
	byName map[string]*Skill
	list   []Skill // 按 name 字典序稳定
}

var defaultRegistry = LoadBuiltin()

// Default 返回全局单例的内置技能注册表（CLI / 测试等不构造 Engine 的场景用）。
func Default() *Registry { return defaultRegistry }

// LoadBuiltin 返回一份全新的内置技能注册表（内容与 Default() 等价，实例独立）。
// DIP 原则：Engine 通过构造函数显式持有 *Registry，不依赖包级 defaultRegistry 全局状态。
func LoadBuiltin() *Registry { return MustLoad(builtinskills.FS()) }

// MustLoad 从给定 fs.FS 加载 */SKILL.md；解析失败启动期直接 panic。
func MustLoad(fsys fs.FS) *Registry {
	reg, err := Load(fsys)
	if err != nil {
		panic(fmt.Sprintf("加载 skills 失败: %v", err))
	}
	return reg
}

// Load 从给定 fs.FS 加载 */SKILL.md。
// 每个 skill 是 fs 根目录下的一个子目录（<name>/SKILL.md），目录名必须与 frontmatter.name 一致。
func Load(fsys fs.FS) (*Registry, error) {
	if fsys == nil {
		return nil, fmt.Errorf("skill fs is nil")
	}
	byName := map[string]*Skill{}
	var list []Skill

	// fs 根路径就是 skill 资产目录，匹配模式 */SKILL.md 得到所有 skill 文件。
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "SKILL.md" {
			return nil
		}
		// path = "<name>/SKILL.md"，skill 名即父目录名
		dirName := filepath.Base(filepath.Dir(path))
		raw, rErr := fs.ReadFile(fsys, path)
		if rErr != nil {
			return fmt.Errorf("读取 skill %s 失败: %w", path, rErr)
		}
		sk, pErr := parseSkillMD(raw)
		if pErr != nil {
			return fmt.Errorf("解析 skill %s 失败: %w", path, pErr)
		}
		if sk.Name == "" {
			return fmt.Errorf("skill %s 缺少 name 字段", path)
		}
		if dirName != sk.Name {
			return fmt.Errorf("skill 目录名 %q 与 frontmatter.name %q 不一致", dirName, sk.Name)
		}
		if _, dup := byName[sk.Name]; dup {
			return fmt.Errorf("skill 名称重复: %s", sk.Name)
		}
		byName[sk.Name] = sk
		list = append(list, *sk)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return &Registry{byName: byName, list: list}, nil
}

// parseSkillMD 切分 frontmatter + 正文，再分别解析。
func parseSkillMD(raw []byte) (*Skill, error) {
	s := string(raw)
	if !strings.HasPrefix(s, "---") {
		return nil, fmt.Errorf("缺少 YAML frontmatter 起始标记")
	}
	// 跳 "---" + 换行
	rest := s[3:]
	if strings.HasPrefix(rest, "\r") {
		rest = rest[1:]
	}
	if !strings.HasPrefix(rest, "\n") {
		return nil, fmt.Errorf("frontmatter 起始格式错误")
	}
	rest = rest[1:]

	endIdx := strings.Index(rest, "\n---")
	if endIdx < 0 {
		return nil, fmt.Errorf("缺少 YAML frontmatter 结束标记")
	}
	fmText := rest[:endIdx]
	body := rest[endIdx+len("\n---"):]
	// 去掉结束标记后紧跟的换行
	if strings.HasPrefix(body, "\r") {
		body = body[1:]
	}
	if strings.HasPrefix(body, "\n") {
		body = body[1:]
	}

	var fm skillFrontmatter
	if err := parseFrontmatter(fmText, &fm); err != nil {
		return nil, fmt.Errorf("frontmatter 解析失败: %w", err)
	}

	// metadata.class → 职业权限
	var wAvail, mAvail bool
	switch strings.ToLower(strings.TrimSpace(fm.Metadata.Class)) {
	case "warrior", "剑士":
		wAvail = true
	case "mage", "法师":
		mAvail = true
	case "both", "通用", "all", "":
		wAvail, mAvail = true, true
	default:
		return nil, fmt.Errorf("未知 metadata.class 值: %q", fm.Metadata.Class)
	}

	return &Skill{
		Name:             strings.TrimSpace(fm.Name),
		Version:          strings.TrimSpace(fm.Version),
		Description:      strings.TrimSpace(fm.Description),
		Category:         strings.TrimSpace(fm.Metadata.Category),
		Kind:             strings.TrimSpace(fm.Metadata.Kind),
		RequiresBins:     fm.Metadata.Requires.Bins,
		RequiresCliHelp:  strings.TrimSpace(fm.Metadata.Requires.CliHelp),
		WarriorAvailable: wAvail,
		MageAvailable:    mAvail,
		RelatedSkills:    fm.Metadata.RelatedSkills,
		Body:             strings.TrimSpace(body),
	}, nil
}

// ==================== 查询 API ====================

// List 返回所有内置技能。
func (r *Registry) List() []Skill {
	out := make([]Skill, len(r.list))
	copy(out, r.list)
	return out
}

// ListManifests 返回所有 skill 的元数据视图，不包含正文。
func (r *Registry) ListManifests() []Manifest {
	list := r.List()
	out := make([]Manifest, 0, len(list))
	for _, sk := range list {
		out = append(out, sk.Manifest())
	}
	return out
}

// ListForClass 返回指定职业可用的技能。
func (r *Registry) ListForClass(class model.AdventurerClass) []Skill {
	var out []Skill
	for _, sk := range r.list {
		switch class {
		case model.ClassWarrior:
			if sk.WarriorAvailable {
				out = append(out, sk)
			}
		case model.ClassMage:
			if sk.MageAvailable {
				out = append(out, sk)
			}
		}
	}
	return out
}

// ListManifestsForClass 返回指定职业可用 skill 的元数据视图，不包含正文。
func (r *Registry) ListManifestsForClass(class model.AdventurerClass) []Manifest {
	list := r.ListForClass(class)
	out := make([]Manifest, 0, len(list))
	for _, sk := range list {
		out = append(out, sk.Manifest())
	}
	return out
}

// Get 按名字查找技能。
func (r *Registry) Get(name string) (*Skill, error) {
	sk, ok := r.byName[name]
	if !ok {
		return nil, fmt.Errorf("技能不存在: %s", name)
	}
	return sk, nil
}

// BuildIndexLine 返回职业可用 skills 的 agent-side manifest，用于启动时注入 system prompt。
// 对齐 Anthropic progressive disclosure：只暴露 name + description，正文按需加载。
// 平台不使用该索引做路由或语义判断；它只给 agent 自己选择。
func (r *Registry) BuildIndexLine(class model.AdventurerClass) string {
	return r.BuildIndexLineWithExtras(class, nil)
}

// BuildIndexLineWithExtras 在职业默认技能基础上，追加指定名称的额外技能。
// extraSkillNames 中不存在的技能会被静默忽略（避免配置错误导致启动失败）。
// 额外技能总是显示在列表末尾，并用 [自定义] 标记。
func (r *Registry) BuildIndexLineWithExtras(class model.AdventurerClass, extraSkillNames []string) string {
	avail := r.ListForClass(class)
	if len(avail) == 0 && len(extraSkillNames) == 0 {
		return ""
	}

	// 收集额外技能（去重，跳过已在职业默认列表中的）
	seen := map[string]bool{}
	for _, sk := range avail {
		seen[sk.Name] = true
	}
	var extras []Skill
	for _, name := range extraSkillNames {
		if seen[name] {
			continue
		}
		if sk, err := r.Get(name); err == nil {
			extras = append(extras, *sk)
			seen[name] = true
		}
	}

	if len(avail) == 0 && len(extras) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Agent Skills Manifest\n\n")
	sb.WriteString("以下 skill 是 agent-side instruction package。你自己判断是否需要加载；平台不会替你选择 skill，也不会根据 skill 内容推断工具调用。需要详细指南时运行 `gloop skill show <name>`。\n\n")
	for _, sk := range avail {
		desc := sk.Description
		// 超长 description 截断（frontmatter 里的 description 本来就该写紧凑）
		if utf8.RuneCountInString(desc) > 180 {
			desc = truncateRunes(desc, 180)
		}
		sb.WriteString(fmt.Sprintf("- `%s`: %s\n", sk.Name, desc))
	}
	for _, sk := range extras {
		desc := sk.Description
		if utf8.RuneCountInString(desc) > 180 {
			desc = truncateRunes(desc, 180)
		}
		sb.WriteString(fmt.Sprintf("- `%s` [自定义]: %s\n", sk.Name, desc))
	}
	return sb.String()
}

func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// ==================== 输出 ====================

// FormatMarkdown 把技能格式化为 Markdown 文档（供 `gloop skill show` 输出用）。
// 官方 skill 格式的正文已经是完整 Markdown，这里只在最前面补一个简短的元信息块，
// 方便 agent 一眼看到版本和依赖；正文原样透传。
func (sk *Skill) FormatMarkdown() string {
	var sb strings.Builder

	// 元信息摘要（不是 frontmatter，直接渲染）
	if sk.Version != "" {
		sb.WriteString(fmt.Sprintf("> **version:** %s  ", sk.Version))
	}
	if sk.Kind != "" {
		sb.WriteString(fmt.Sprintf("**kind:** %s  ", sk.Kind))
	}
	if sk.Category != "" {
		sb.WriteString(fmt.Sprintf("**分类：** %s  ", sk.Category))
	}
	if len(sk.RequiresBins) > 0 {
		sb.WriteString("**依赖：** ")
		for i, b := range sk.RequiresBins {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("`" + b + "`")
		}
		sb.WriteString("  ")
	}
	if len(sk.RelatedSkills) > 0 {
		sb.WriteString("**相关：** ")
		names := make([]string, 0, len(sk.RelatedSkills))
		for _, rs := range sk.RelatedSkills {
			names = append(names, rs.Name)
		}
		sb.WriteString(strings.Join(names, ", "))
	}
	if sb.Len() > 0 {
		sb.WriteString("\n\n")
	}

	if sk.Description != "" {
		sb.WriteString(sk.Description)
		sb.WriteString("\n\n")
		sb.WriteString("---\n\n")
	}

	if sk.Body != "" {
		sb.WriteString(sk.Body)
		sb.WriteString("\n")
	}

	return sb.String()
}

// ==================== frontmatter 解析器 ====================
//
// 够用的极简 YAML 子集，零外部依赖。覆盖官方 agent skill 格式实际用到的所有语法：
//
//   - key: scalar                   # 标量（string/bool），可选双/单引号
//   - key:                          # 嵌套对象：下一行缩进更深的 key/val 对子级
//       subkey: scalar
//       sublist:
//         - item1
//         - item2
//   - key: ["a", "b", "c"]          # 流式数组（字符串列表，元素可带引号）
//   - 注释：整行 `# ...` 或行尾 ` # ...`
//
// 不支持：流式对象 {a: b}、内联复合、锚点别名、多行块标量（|/>, 需要时再加）。

type parseState struct {
	path           []string         // 当前对象路径，比如 ["metadata","requires"]
	indentAt       map[int][]string // 缩进深度 → 对应层级的 path 快照
	currentKey     string           // 当前正在收集列表项的 key（上一行 key: 后无值）
	pending        []string
	pendingObjects []map[string]string // 对象列表：已完成的对象
	curListItem    map[string]string   // 对象列表：正在构建的当前对象
	listItemIndent int                 // 对象列表项（- 开头）的缩进深度
	fm             *skillFrontmatter
}

func parseFrontmatter(text string, fm *skillFrontmatter) error {
	st := &parseState{
		indentAt: map[int][]string{0: {}},
		fm:       fm,
	}
	lines := strings.Split(text, "\n")

	for lineIdx, raw := range lines {
		lineNum := lineIdx + 1
		// 去尾注释
		if i := strings.Index(raw, " #"); i >= 0 {
			raw = raw[:i]
		} else if strings.HasPrefix(strings.TrimLeft(raw, " \t"), "#") {
			continue
		}
		line := strings.TrimRight(raw, "\r ")
		if strings.TrimSpace(line) == "" {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		content := strings.TrimLeft(line, " ")

		// 列表项？
		if indent > 0 && (strings.HasPrefix(content, "- ") || strings.HasPrefix(content, "-")) {
			if st.currentKey == "" {
				return fmt.Errorf("第 %d 行：列表项不归属任何 key", lineNum)
			}
			var rest string
			if strings.HasPrefix(content, "- ") {
				rest = strings.TrimSpace(content[2:])
			} else {
				rest = strings.TrimSpace(content[1:])
			}

			// 判断是对象列表项还是标量列表项：包含 ":" 且不是冒号结尾（标量值本身带冒号的情况极少，
			// 若有歧义优先按对象处理——相关技能这类字段不会有纯冒号值）
			if colon := strings.Index(rest, ":"); colon >= 0 {
				// 对象列表项：先把上一个对象收尾
				if st.curListItem != nil {
					st.pendingObjects = append(st.pendingObjects, st.curListItem)
				}
				key := strings.TrimSpace(rest[:colon])
				val := strings.TrimSpace(rest[colon+1:])
				st.curListItem = map[string]string{key: stripQuotes(val)}
				st.listItemIndent = indent
				continue
			}
			// 标量列表项
			st.pending = append(st.pending, stripQuotes(rest))
			continue
		}

		// 如果正在构建对象列表项，且当前行缩进比列表项更深，
		// 则是当前对象的一个字段，直接写入 curListItem。
		if st.curListItem != nil && indent > st.listItemIndent {
			colon := strings.Index(content, ":")
			if colon < 0 {
				// 不是 key:value，跳过（按当前解析器哲学，不报错）
				continue
			}
			key := strings.TrimSpace(content[:colon])
			val := strings.TrimSpace(content[colon+1:])
			if key != "" {
				st.curListItem[key] = stripQuotes(val)
			}
			continue
		}

		// 缩进变化时：刷掉待处理列表，并调整当前路径栈
		if err := st.flushList(lineNum); err != nil {
			return err
		}
		// 路径回退：找到缩进等于当前 indent 的层级
		path, ok := st.indentAt[indent]
		if !ok {
			return fmt.Errorf("第 %d 行：缩进深度 %d 与之前层级不匹配", lineNum, indent)
		}
		st.path = append([]string(nil), path...)

		colon := strings.Index(content, ":")
		if colon < 0 {
			return fmt.Errorf("第 %d 行：缺少 ':'", lineNum)
		}
		key := strings.TrimSpace(content[:colon])
		val := strings.TrimSpace(content[colon+1:])
		if key == "" {
			return fmt.Errorf("第 %d 行：空 key", lineNum)
		}

		if val == "" {
			// 两种情况：嵌套对象 / 后续跟块列表 → 都先记为 currentKey，
			// 下一行缩进更深时判定：若为 "- " 开头则是列表，否则是嵌套对象。
			st.currentKey = key
			continue
		}

		// 立即有值：标量或流式数组
		fullKey := append(append([]string(nil), st.path...), key)
		if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
			items := parseFlowArray(val)
			setField(fm, fullKey, "", items, lineNum)
			st.currentKey = ""
			continue
		}
		setField(fm, fullKey, stripQuotes(val), nil, lineNum)
		st.currentKey = ""
	}

	// 文件结束时再刷一次列表
	if err := st.flushList(len(lines)); err != nil {
		return err
	}
	return nil
}

// flushList 在缩进变化或文件结束时，把 currentKey 待处理的列表落下来。
// 标量列表和对象列表都在这里处理。
// 如果 currentKey 指向的子路径里还从来没写过任何子字段（说明 key: 之后没有列表，
// 只有嵌套对象或空对象），就把 path 下推一级。
func (st *parseState) flushList(lineNum int) error {
	if st.currentKey == "" {
		return nil
	}
	// 对象列表
	if st.curListItem != nil || len(st.pendingObjects) > 0 {
		// 把最后一个对象收尾
		if st.curListItem != nil {
			st.pendingObjects = append(st.pendingObjects, st.curListItem)
			st.curListItem = nil
		}
		fullKey := append(append([]string(nil), st.path...), st.currentKey)
		setObjectListField(st.fm, fullKey, st.pendingObjects, lineNum)
		st.pendingObjects = nil
		st.listItemIndent = 0
		st.currentKey = ""
		return nil
	}
	if len(st.pending) > 0 {
		// 标量块列表
		fullKey := append(append([]string(nil), st.path...), st.currentKey)
		setField(st.fm, fullKey, "", st.pending, lineNum)
		st.pending = nil
		st.currentKey = ""
		return nil
	}
	// 无 pending → currentKey 是嵌套对象的入口：把当前路径下推一级，并记住当前缩进深度（即下一行的缩进）。
	// 约定：嵌套对象入口的下一行缩进深度 = 当前行缩进深度 + 2（保持和官方格式一致）。
	// 这里我们在 parseFrontmatter 主循环中通过 indentAt 匹配，下一层的缩进深度就是
	// "当前 key 那行的 indent + 该 key 下第一行子字段实际出现的 indent"，
	// 匹配逻辑交给主循环的 indentAt[indent] 查找。
	st.path = append(append([]string(nil), st.path...), st.currentKey)
	// 把路径写入一个"占位缩进"：下一级只要缩进 > 当前 indent 都算。
	// 我们没有下一级确切缩进值，所以用主循环的"找到或报错"策略。
	// 解决办法：等主循环遇到更深的缩进行时，自动把那个 indent 登记到 indentAt，
	// 所以我们需要：在主循环里"路径回退"失败时，如果路径只是下推了一级的 currentKey，
	// 就动态登记 indentAt。

	// 为简单起见，这里直接：允许下一级任意缩进深度（> 当前层 indent）都匹配为该路径。
	// 改动主循环比较复杂，改用策略：遇到新的缩进深度时，如果它在 path 栈顶当前的下一层候选
	// 范围内，就自动登记。我们在这里把 "path 候选" 记到另一个 map。
	// 更简单：因为我们的嵌套层级最多 3 层，且官方格式是严格每级 +2 空格，直接根据路径栈深度
	// 推断 indent=2*len(path) 就够。
	// 所以直接登记 indentAt。
	st.indentAt[2*len(st.path)] = append([]string(nil), st.path...)
	st.currentKey = ""
	return nil
}

// parseFlowArray 解析 YAML 流式字符串数组：`["a", 'b', c]` → ["a","b","c"]。
// 假设外层方括号已经由调用方校验。
func parseFlowArray(val string) []string {
	inner := strings.TrimSpace(val[1 : len(val)-1])
	if inner == "" {
		return nil
	}
	// 按逗号切，注意不要切引号内部的逗号。
	var items []string
	var cur strings.Builder
	inSingle, inDouble := false, false
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		switch {
		case c == '"' && !inSingle:
			inDouble = !inDouble
			cur.WriteByte(c)
		case c == '\'' && !inDouble:
			inSingle = !inSingle
			cur.WriteByte(c)
		case c == ',' && !inSingle && !inDouble:
			items = append(items, stripQuotes(strings.TrimSpace(cur.String())))
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if s := strings.TrimSpace(cur.String()); s != "" {
		items = append(items, stripQuotes(s))
	}
	return items
}

// setField 把一个值（标量或列表）写入 frontmatter 指定路径。
// path 例如 ["name"] / ["metadata", "requires", "bins"]。
// 未知路径静默忽略，保持向前兼容。
func setField(fm *skillFrontmatter, path []string, scalar string, list []string, lineNum int) {
	_ = lineNum
	if len(path) == 0 {
		return
	}
	join := strings.ToLower(strings.Join(path, "."))

	if list != nil {
		switch join {
		case "metadata.requires.bins":
			fm.Metadata.Requires.Bins = list
		}
		return
	}
	switch join {
	case "name":
		fm.Name = scalar
	case "version":
		fm.Version = scalar
	case "description":
		fm.Description = scalar
	case "metadata.requires.chelp":
		fm.Metadata.Requires.CliHelp = scalar
	case "metadata.requires.clihelp":
		fm.Metadata.Requires.CliHelp = scalar
	case "metadata.class":
		fm.Metadata.Class = scalar
	case "metadata.category":
		fm.Metadata.Category = scalar
	case "metadata.kind":
		fm.Metadata.Kind = scalar
	}
}

// setObjectListField 把对象列表写入 frontmatter 指定路径。
// 未知路径静默忽略，保持向前兼容。
func setObjectListField(fm *skillFrontmatter, path []string, objects []map[string]string, lineNum int) {
	_ = lineNum
	if len(path) == 0 {
		return
	}
	join := strings.ToLower(strings.Join(path, "."))

	switch join {
	case "metadata.related_skills":
		var skills []RelatedSkill
		for _, obj := range objects {
			name := strings.TrimSpace(obj["name"])
			if name == "" {
				// 没有 name 的条目静默跳过
				continue
			}
			typ := strings.TrimSpace(obj["type"])
			if typ == "" {
				typ = "related"
			}
			skills = append(skills, RelatedSkill{
				Name:        name,
				Type:        typ,
				Description: strings.TrimSpace(obj["description"]),
			})
		}
		fm.Metadata.RelatedSkills = skills
	}
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
