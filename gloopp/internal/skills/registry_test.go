package skills

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Registry 查询测试 ====================

func TestDefaultRegistry_Loaded(t *testing.T) {
	reg := Default()
	list := reg.List()
	if len(list) != 11 {
		t.Fatalf("期望 11 个内置 skill，got %d", len(list))
	}

	// 名称应该按字典序稳定
	names := []string{}
	for _, sk := range list {
		names = append(names, sk.Name)
		if sk.Version == "" {
			t.Errorf("%s: version 不应该为空", sk.Name)
		}
		if sk.Description == "" {
			t.Errorf("%s: description 不应该为空", sk.Name)
		}
	}
	expectedOrder := []string{"gloop-code-exploration", "gloop-inbox-triage", "gloop-iteration-workflow", "gloop-note-keeping", "gloop-posting", "gloop-quest-execution", "gloop-quest-fanout", "gloop-quest-review", "gloop-self-awareness", "gloop-user-context", "gloop-user-notification"}
	for i, name := range expectedOrder {
		if names[i] != name {
			t.Errorf("排序不对：第 %d 位期望 %s，got %s", i, name, names[i])
		}
	}
}

func TestLoad_FromInjectedFS(t *testing.T) {
	fsys := fstest.MapFS{
		"sample_skill/SKILL.md": {
			Data: []byte(`---
name: sample_skill
version: 0.1.0
description: "sample skill for injected fs"
metadata:
  class: both
  category: test
  requires:
    bins: ["gloop"]
---
# sample_skill

## Trigger Examples

- sample

## CLI Contract

Run sample.

## Discipline

Keep it small.
`),
		},
	}
	reg, err := Load(fsys)
	if err != nil {
		t.Fatalf("Load injected fs failed: %v", err)
	}
	list := reg.List()
	if len(list) != 1 {
		t.Fatalf("expected one skill, got %d", len(list))
	}
	sk, err := reg.Get("sample_skill")
	if err != nil {
		t.Fatalf("Get sample_skill failed: %v", err)
	}
	if !sk.WarriorAvailable || !sk.MageAvailable {
		t.Fatalf("sample_skill should be available to both classes: %+v", sk.Manifest())
	}
	if sk.Category != "test" || sk.RequiresBins[0] != "gloop" {
		t.Fatalf("metadata not parsed from injected fs: %+v", sk)
	}
}

func TestRegistry_Get(t *testing.T) {
	reg := Default()

	sk, err := reg.Get("gloop-quest-execution")
	if err != nil {
		t.Fatalf("Get gloop-quest-execution 失败: %v", err)
	}
	if sk.Name != "gloop-quest-execution" {
		t.Errorf("名字不对，got %s", sk.Name)
	}
	if !sk.WarriorAvailable {
		t.Error("gloop-quest-execution 应该对剑士可用")
	}
	if sk.MageAvailable {
		t.Error("gloop-quest-execution 不应对法师可用")
	}
	if !strings.Contains(sk.Body, "## Discipline") {
		t.Error("正文应该包含 Discipline 章节")
	}
	if !strings.Contains(sk.Body, "Trigger Examples") {
		t.Error("正文应该包含 Trigger Examples 章节")
	}
	if sk.Category != "execution" {
		t.Errorf("category = %s，期望 execution", sk.Category)
	}

	reviewSkill, err := reg.Get("gloop-quest-review")
	if err != nil {
		t.Fatalf("Get gloop-quest-review 失败: %v", err)
	}
	for _, want := range []string{
		"gloop command list",
		"gloop command run <command_id>",
		"gloop command run go-test",
		"gloop skill show gloop-quest-review",
		"CLI 主路径",
		"不向某类 agent 暴露额外的 native platform tool",
		"ReviewReport",
		"EvidenceRef",
	} {
		if !strings.Contains(reviewSkill.Body, want) {
			t.Fatalf("gloop-quest-review should document mage CLI/skill path %q\n---\n%s", want, reviewSkill.Body)
		}
	}

	executionSkill, err := reg.Get("gloop-quest-execution")
	if err != nil {
		t.Fatalf("Get gloop-quest-execution 失败: %v", err)
	}
	if !strings.Contains(executionSkill.Body, "MakerReport") || !strings.Contains(executionSkill.Body, "Maker Delivery") {
		t.Fatalf("gloop-quest-execution should document MakerReport/Maker Delivery")
	}

	inboxSkill, err := reg.Get("gloop-inbox-triage")
	if err != nil {
		t.Fatalf("Get gloop-inbox-triage 失败: %v", err)
	}
	for _, want := range []string{"Human Exceptions", "Automation Candidates"} {
		if !strings.Contains(inboxSkill.Body, want) {
			t.Fatalf("gloop-inbox-triage should document %q", want)
		}
	}

	contextSkill, err := reg.Get("gloop-user-context")
	if err != nil {
		t.Fatalf("Get gloop-user-context 失败: %v", err)
	}
	for _, want := range []string{"Activity Snapshot", "Loop State Spine"} {
		if !strings.Contains(contextSkill.Body, want) {
			t.Fatalf("gloop-user-context should document %q", want)
		}
	}

	if _, err := reg.Get("nonexistent"); err == nil {
		t.Error("查不存在的 skill 应该报错")
	}
}

func TestSkillValidateRejectsStaleLoopStateTerm(t *testing.T) {
	sk := Skill{
		Name:        "sample",
		Version:     "0.1.0",
		Description: "sample",
		Kind:        KindCapability,
		Body: "# sample\n\n## Trigger Examples\n\nx\n\n## CLI Contract\n\nRead loop_state before work.\n\n## Discipline\n\nx\n",
	}
	issues := sk.Validate()
	found := false
	for _, issue := range issues {
		if issue.Field == "body.terminology" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected stale loop_state terminology issue, got %+v", issues)
	}

	sk.Body = strings.ReplaceAll(sk.Body, "loop_state", "loop_state_spine")
	for _, issue := range sk.Validate() {
		if issue.Field == "body.terminology" {
			t.Fatalf("loop_state_spine should be accepted, got %+v", sk.Validate())
		}
	}
}

func TestRegistry_ListForClass(t *testing.T) {
	reg := Default()

	warrior := reg.ListForClass(model.ClassWarrior)
	mage := reg.ListForClass(model.ClassMage)

	// 剑士：base + gloop-inbox-triage + gloop-user-notification + gloop-quest-fanout + gloop-code-exploration + gloop-iteration-workflow + gloop-posting = 10
	if len(warrior) != 10 {
		t.Errorf("剑士应该看到 10 个 skill，got %d", len(warrior))
	}
	// 法师：base + gloop-inbox-triage + gloop-user-notification + gloop-code-exploration + gloop-posting = 8
	if len(mage) != 8 {
		t.Errorf("法师应该看到 8 个 skill，got %d", len(mage))
	}

	has := func(list []Skill, name string) bool {
		for _, s := range list {
			if s.Name == name {
				return true
			}
		}
		return false
	}
	if !has(warrior, "gloop-quest-execution") {
		t.Error("剑士应该看到 gloop-quest-execution")
	}
	if has(warrior, "gloop-quest-review") {
		t.Error("剑士不该看到 gloop-quest-review")
	}
	if !has(mage, "gloop-quest-review") {
		t.Error("法师应该看到 gloop-quest-review")
	}
	if has(mage, "gloop-quest-execution") {
		t.Error("法师不该看到 gloop-quest-execution")
	}
}

func TestRegistry_ListManifests_DoesNotExposeBody(t *testing.T) {
	reg := Default()
	manifests := reg.ListManifestsForClass(model.ClassWarrior)
	if len(manifests) == 0 {
		t.Fatal("expected manifests")
	}
	for _, m := range manifests {
		if m.Name == "" || m.Description == "" {
			t.Fatalf("manifest should include metadata: %+v", m)
		}
		if strings.Contains(m.Description, "## CLI Contract") {
			t.Fatalf("manifest leaked body content: %+v", m)
		}
	}
}

func TestRegistry_BuildIndexLine(t *testing.T) {
	reg := Default()

	wLine := reg.BuildIndexLine(model.ClassWarrior)
	// 剑士应该有 gloop-quest-execution（作为 skill 条目名出现）
	if !strings.Contains(wLine, "`gloop-quest-execution`") {
		t.Errorf("剑士索引应该列出 gloop-quest-execution skill，got:\n%s", wLine)
	}
	// 剑士不应该把 gloop-quest-review 作为自己的 skill 条目列出来
	// （gloop-quest-execution 的 description 里文字提到 gloop-quest-review 不算）
	if strings.Contains(wLine, "`gloop-quest-review`") {
		t.Errorf("剑士索引不该列出 gloop-quest-review skill，got:\n%s", wLine)
	}
	// 索引提示通过 show 命令加载完整 skill
	if !strings.Contains(wLine, "skill show") {
		t.Errorf("索引应该提示 show 命令，got:\n%s", wLine)
	}
	if !strings.Contains(wLine, "## Agent Skills Manifest") {
		t.Errorf("索引应该有 manifest 标题，got:\n%s", wLine)
	}
	if !strings.Contains(wLine, "平台不会替你选择 skill") {
		t.Errorf("索引应该说明平台不做 skill 选择，got:\n%s", wLine)
	}
	// name 后必须跟 description（agent 靠 dense description 判断触发条件）
	if !strings.Contains(wLine, "委托执行") {
		t.Errorf("剑士索引里 gloop-quest-execution 必须带 description，got:\n%s", wLine)
	}

	mLine := reg.BuildIndexLine(model.ClassMage)
	if !strings.Contains(mLine, "`gloop-quest-review`") {
		t.Errorf("法师索引应该列出 gloop-quest-review skill，got:\n%s", mLine)
	}
}

// ==================== FormatMarkdown 测试 ====================

func TestSkill_FormatMarkdown(t *testing.T) {
	sk, err := Default().Get("gloop-quest-execution")
	if err != nil {
		t.Fatalf("查 skill 失败: %v", err)
	}
	md := sk.FormatMarkdown()
	if !strings.Contains(md, "version:") {
		t.Error("markdown 头部应该包含 version 元信息")
	}
	if !strings.Contains(md, "gloop-quest-execution") {
		t.Error("markdown 应该包含技能名")
	}
	if !strings.Contains(md, "Trigger Examples") {
		t.Error("markdown 正文应该包含 Trigger Examples 章节")
	}
	if !strings.Contains(md, "Discipline") {
		t.Error("markdown 正文应该包含 Discipline 章节")
	}
	// 应该包含 requires 依赖里的 gloop
	if !strings.Contains(md, "`gloop`") {
		t.Error("markdown 元信息应该包含 requires bins")
	}
}

// ==================== 正文里必备章节的完整性校验 ====================

func TestAllSkills_RequiredSections(t *testing.T) {
	reg := Default()
	for _, sk := range reg.List() {
		body := sk.Body
		if !strings.Contains(body, "# "+sk.Name) {
			t.Errorf("%s: 正文缺少 H1 标题「# %s」", sk.Name, sk.Name)
		}
		if !strings.Contains(body, "## Trigger Examples") {
			t.Errorf("%s: 正文缺少 Trigger Examples 章节", sk.Name)
		}
		if !strings.Contains(body, "## CLI Contract") {
			t.Errorf("%s: 正文缺少 CLI Contract 章节", sk.Name)
		}
		if !strings.Contains(body, "## Discipline") {
			t.Errorf("%s: 正文缺少 Discipline 章节", sk.Name)
		}
	}
}

// ==================== frontmatter 解析测试 ====================

func TestParseFrontmatter_OfficialFormat(t *testing.T) {
	var fm skillFrontmatter
	text := `name: douyin-style
version: 1.2.3
description: "一段带引号的描述，含逗号、特殊字符和反斜杠 \\"
metadata:
  class: warrior
  category: execution
  requires:
    bins: ["gloop", "bytedcli"]
    cliHelp: "gloop review --help"
`
	if err := parseFrontmatter(text, &fm); err != nil {
		t.Fatalf("parse 失败: %v", err)
	}
	if fm.Name != "douyin-style" {
		t.Errorf("name = %q", fm.Name)
	}
	if fm.Version != "1.2.3" {
		t.Errorf("version = %q", fm.Version)
	}
	if !strings.HasPrefix(fm.Description, "一段带引号") {
		t.Errorf("description = %q", fm.Description)
	}
	if fm.Metadata.Class != "warrior" {
		t.Errorf("metadata.class = %q", fm.Metadata.Class)
	}
	if fm.Metadata.Category != "execution" {
		t.Errorf("metadata.category = %q", fm.Metadata.Category)
	}
	if len(fm.Metadata.Requires.Bins) != 2 || fm.Metadata.Requires.Bins[0] != "gloop" {
		t.Errorf("requires.bins = %+v", fm.Metadata.Requires.Bins)
	}
	if !strings.Contains(fm.Metadata.Requires.CliHelp, "review") {
		t.Errorf("requires.cliHelp = %q", fm.Metadata.Requires.CliHelp)
	}
}

func TestParseFrontmatter_ClassBothAndBlockList(t *testing.T) {
	var fm skillFrontmatter
	text := `name: both-skill
version: 0.1.0
description: 通用型技能
metadata:
  class: both
  category: note
  requires:
    bins:
      - gloop
      - another-cli
    cliHelp: gloop note --help
`
	if err := parseFrontmatter(text, &fm); err != nil {
		t.Fatalf("parse 失败: %v", err)
	}
	if fm.Metadata.Class != "both" {
		t.Errorf("class = %q", fm.Metadata.Class)
	}
	if len(fm.Metadata.Requires.Bins) != 2 || fm.Metadata.Requires.Bins[1] != "another-cli" {
		t.Errorf("bins 不对: %+v", fm.Metadata.Requires.Bins)
	}
}

func TestParseFrontmatter_UnknownFieldsIgnored(t *testing.T) {
	var fm skillFrontmatter
	text := `name: foo
version: 1.0.0
description: bar
unknown_top: 42
metadata:
  class: both
  unknown_meta_field: whatever
  requires:
    bins: ["gloop"]
    cliHelp: "h"
    unknown_requires: ignored
`
	if err := parseFrontmatter(text, &fm); err != nil {
		t.Fatalf("未知字段不该报错: %v", err)
	}
	if fm.Name != "foo" {
		t.Error("name 不对")
	}
}

func TestParseFrontmatter_RelatedSkills(t *testing.T) {
	var fm skillFrontmatter
	text := `name: test-skill
version: 0.1.0
description: test
metadata:
  class: both
  category: test
  related_skills:
    - name: gloop-user-notification
      type: depends_on
      description: 通知的写作规范
    - name: gloop-self-awareness
      type: related
      description: quest 列表查询
    - name: gloop-code-exploration
      description: 代码探索
`
	if err := parseFrontmatter(text, &fm); err != nil {
		t.Fatalf("parse 失败: %v", err)
	}
	skills := fm.Metadata.RelatedSkills
	if len(skills) != 3 {
		t.Fatalf("期望 3 个 related_skills，got %d", len(skills))
	}

	// 第 1 个
	if skills[0].Name != "gloop-user-notification" {
		t.Errorf("第 1 个 name = %q，期望 gloop-user-notification", skills[0].Name)
	}
	if skills[0].Type != "depends_on" {
		t.Errorf("第 1 个 type = %q，期望 depends_on", skills[0].Type)
	}
	if skills[0].Description != "通知的写作规范" {
		t.Errorf("第 1 个 description = %q", skills[0].Description)
	}

	// 第 2 个
	if skills[1].Name != "gloop-self-awareness" {
		t.Errorf("第 2 个 name = %q", skills[1].Name)
	}
	if skills[1].Type != "related" {
		t.Errorf("第 2 个 type = %q", skills[1].Type)
	}

	// 第 3 个：省略 type，默认 related
	if skills[2].Name != "gloop-code-exploration" {
		t.Errorf("第 3 个 name = %q", skills[2].Name)
	}
	if skills[2].Type != "related" {
		t.Errorf("第 3 个 type 默认值应为 related，got %q", skills[2].Type)
	}
	if skills[2].Description != "代码探索" {
		t.Errorf("第 3 个 description = %q", skills[2].Description)
	}
}

func TestParseFrontmatter_RelatedSkills_SkipNameless(t *testing.T) {
	var fm skillFrontmatter
	text := `name: test-skill
version: 0.1.0
description: test
metadata:
  related_skills:
    - type: depends_on
      description: 没有 name，应该被跳过
    - name: valid-skill
      type: related
    - name:
      type: related
      description: name 为空字符串，应该被跳过
`
	if err := parseFrontmatter(text, &fm); err != nil {
		t.Fatalf("parse 失败: %v", err)
	}
	skills := fm.Metadata.RelatedSkills
	if len(skills) != 1 {
		t.Fatalf("期望只剩 1 个有效 related_skill，got %d", len(skills))
	}
	if skills[0].Name != "valid-skill" {
		t.Errorf("有效 skill name = %q", skills[0].Name)
	}
}

func TestParseFrontmatter_RelatedSkills_UnknownFieldsIgnored(t *testing.T) {
	var fm skillFrontmatter
	text := `name: test
version: 1.0.0
description: t
metadata:
  related_skills:
    - name: foo
      type: related
      description: bar
      unknown_field: should be ignored
      another: whatever
`
	if err := parseFrontmatter(text, &fm); err != nil {
		t.Fatalf("parse 失败: %v", err)
	}
	if len(fm.Metadata.RelatedSkills) != 1 {
		t.Fatalf("期望 1 个，got %d", len(fm.Metadata.RelatedSkills))
	}
	if fm.Metadata.RelatedSkills[0].Name != "foo" {
		t.Errorf("name = %q", fm.Metadata.RelatedSkills[0].Name)
	}
	if fm.Metadata.RelatedSkills[0].Description != "bar" {
		t.Errorf("description = %q", fm.Metadata.RelatedSkills[0].Description)
	}
}

func TestParseFrontmatter_RelatedSkills_Empty(t *testing.T) {
	var fm skillFrontmatter
	text := `name: test
version: 1.0.0
description: t
metadata:
  class: both
  related_skills:
`
	if err := parseFrontmatter(text, &fm); err != nil {
		t.Fatalf("空 related_skills 不该报错: %v", err)
	}
	if len(fm.Metadata.RelatedSkills) != 0 {
		t.Errorf("空列表应该为 0 个，got %d", len(fm.Metadata.RelatedSkills))
	}
}

func TestParseFrontmatter_RelatedSkills_MixedWithBins(t *testing.T) {
	// 确认对象列表和标量列表可以共存于同一个 metadata 下
	var fm skillFrontmatter
	text := `name: test
version: 1.0.0
description: t
metadata:
  class: both
  requires:
    bins:
      - gloop
      - bytedcli
  related_skills:
    - name: gloop-user-notification
      type: depends_on
    - name: gloop-self-awareness
      type: related
`
	if err := parseFrontmatter(text, &fm); err != nil {
		t.Fatalf("parse 失败: %v", err)
	}
	if len(fm.Metadata.Requires.Bins) != 2 || fm.Metadata.Requires.Bins[0] != "gloop" {
		t.Errorf("bins 不对: %+v", fm.Metadata.Requires.Bins)
	}
	if len(fm.Metadata.RelatedSkills) != 2 {
		t.Errorf("related_skills 数量不对: %d", len(fm.Metadata.RelatedSkills))
	}
	if fm.Metadata.RelatedSkills[0].Name != "gloop-user-notification" {
		t.Errorf("第 0 个 name 不对: %s", fm.Metadata.RelatedSkills[0].Name)
	}
}

func TestParseFlowArray(t *testing.T) {
	cases := map[string][]string{
		`["a", "b", "c"]`:        {"a", "b", "c"},
		`['a', 'b']`:             {"a", "b"},
		`[]`:                     {},
		`["x, with comma", 'y']`: {"x, with comma", "y"},
		`["only"]`:               {"only"},
	}
	for in, want := range cases {
		got := parseFlowArray(in)
		if len(got) != len(want) {
			t.Errorf("parse %s: len 期望 %d，got %d (%+v)", in, len(want), len(got), got)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("parse %s: 第 %d 个 期望 %q，got %q", in, i, want[i], got[i])
			}
		}
	}
}

// ==================== parseSkillMD 整体解析测试 ====================

func TestParseSkillMD_Valid(t *testing.T) {
	md := `---
name: foo
version: 1.0.0
description: 简短描述
metadata:
  class: both
  category: info
  requires:
    bins: ["gloop"]
    cliHelp: "gloop --help"
---

# foo

一段正文。

## Trigger Examples

随便写的示例。

## CLI Contract

` + "```" + `bash
gloop something
` + "```" + `

## Discipline

- 原则一
- 原则二
`
	sk, err := parseSkillMD([]byte(md))
	if err != nil {
		t.Fatalf("parse 失败: %v", err)
	}
	if sk.Name != "foo" {
		t.Errorf("name = %s", sk.Name)
	}
	if sk.Version != "1.0.0" {
		t.Errorf("version = %s", sk.Version)
	}
	if !sk.WarriorAvailable || !sk.MageAvailable {
		t.Error("both 应该对两个职业都可用")
	}
	if !strings.Contains(sk.Body, "## Discipline") {
		t.Errorf("body 应该保留 Discipline 章节: %s", sk.Body)
	}
}

func TestParseSkillMD_RelatedSkills(t *testing.T) {
	md := `---
name: test-skill
version: 0.1.0
description: 测试相关技能
metadata:
  class: both
  category: test
  related_skills:
    - name: gloop-user-notification
      type: depends_on
      description: 通知的写作规范
    - name: gloop-self-awareness
      type: related
---

# test-skill

## Trigger Examples

示例

## CLI Contract

noop

## Discipline

无
`
	sk, err := parseSkillMD([]byte(md))
	if err != nil {
		t.Fatalf("parse 失败: %v", err)
	}
	if len(sk.RelatedSkills) != 2 {
		t.Fatalf("期望 2 个 related_skills，got %d", len(sk.RelatedSkills))
	}
	if sk.RelatedSkills[0].Name != "gloop-user-notification" {
		t.Errorf("第 0 个 name = %q", sk.RelatedSkills[0].Name)
	}
	if sk.RelatedSkills[0].Type != "depends_on" {
		t.Errorf("第 0 个 type = %q", sk.RelatedSkills[0].Type)
	}
	if sk.RelatedSkills[0].Description != "通知的写作规范" {
		t.Errorf("第 0 个 description = %q", sk.RelatedSkills[0].Description)
	}

	// 验证 Manifest() 包含 related_skills
	m := sk.Manifest()
	if len(m.RelatedSkills) != 2 {
		t.Fatalf("Manifest 中 related_skills 数量不对: %d", len(m.RelatedSkills))
	}
	if m.RelatedSkills[0].Name != "gloop-user-notification" {
		t.Errorf("Manifest 中第 0 个 name = %q", m.RelatedSkills[0].Name)
	}

	// 验证 Manifest 是深拷贝
	sk.RelatedSkills[0].Name = "modified"
	if m.RelatedSkills[0].Name == "modified" {
		t.Error("Manifest 应该是深拷贝，不应该被原 skill 修改影响")
	}
}

func TestManifest_RelatedSkills_OmitEmpty(t *testing.T) {
	// 没有 related_skills 时，JSON 中不应出现该字段
	m := Manifest{
		Name:             "test",
		Version:          "1.0.0",
		Description:      "test",
		WarriorAvailable: true,
		MageAvailable:    true,
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}
	if strings.Contains(string(data), "related_skills") {
		t.Errorf("空 related_skills 应该被 omitempty，got: %s", string(data))
	}

	// 有相关技能时应该出现
	m.RelatedSkills = []RelatedSkill{
		{Name: "foo", Type: "related"},
	}
	data, err = json.Marshal(m)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}
	if !strings.Contains(string(data), "related_skills") {
		t.Errorf("有 related_skills 时 JSON 中应该出现，got: %s", string(data))
	}
	if !strings.Contains(string(data), `"name":"foo"`) {
		t.Errorf("JSON 中应该包含 name 字段，got: %s", string(data))
	}
}

func TestSkill_FormatMarkdown_RelatedSkills(t *testing.T) {
	sk := &Skill{
		Name:        "test-skill",
		Version:     "1.0.0",
		Description: "测试",
		RelatedSkills: []RelatedSkill{
			{Name: "gloop-user-notification", Type: "depends_on"},
			{Name: "gloop-self-awareness", Type: "related"},
		},
	}
	md := sk.FormatMarkdown()
	if !strings.Contains(md, "相关：") {
		t.Errorf("FormatMarkdown 应该包含相关技能标签，got:\n%s", md)
	}
	if !strings.Contains(md, "gloop-user-notification") {
		t.Errorf("FormatMarkdown 应该包含相关技能名称，got:\n%s", md)
	}
	if !strings.Contains(md, "gloop-self-awareness") {
		t.Errorf("FormatMarkdown 应该包含所有相关技能名称，got:\n%s", md)
	}
}

func TestParseSkillMD_NoFrontmatterStart(t *testing.T) {
	if _, err := parseSkillMD([]byte("name: foo\n---\n")); err == nil {
		t.Error("缺少起始标记应该报错")
	}
}

func TestParseSkillMD_NoFrontmatterEnd(t *testing.T) {
	if _, err := parseSkillMD([]byte("---\nname: foo\nbar")); err == nil {
		t.Error("缺少结束标记应该报错")
	}
}

// ==================== metadata.class 到职业权限的映射测试 ====================

func TestParseSkillMD_ClassMapping(t *testing.T) {
	cases := []struct {
		classVal     string
		wantW, wantM bool
	}{
		{"warrior", true, false},
		{"mage", false, true},
		{"both", true, true},
		{"", true, true},
		{"剑士", true, false},
		{"通用", true, true},
	}
	for _, c := range cases {
		md := "---\nname: t\nversion: 1\ndescription: d\nmetadata:\n  class: " + c.classVal + "\n  category: info\n  requires:\n    bins: [\"gloop\"]\n    cliHelp: h\n---\n\n# t\n"
		sk, err := parseSkillMD([]byte(md))
		if err != nil {
			t.Errorf("class=%q 解析失败: %v", c.classVal, err)
			continue
		}
		if sk.WarriorAvailable != c.wantW || sk.MageAvailable != c.wantM {
			t.Errorf("class=%q → W=%v M=%v，期望 W=%v M=%v",
				c.classVal, sk.WarriorAvailable, sk.MageAvailable, c.wantW, c.wantM)
		}
	}
}

func TestParseSkillMD_BadClass(t *testing.T) {
	md := "---\nname: t\nversion: 1\ndescription: d\nmetadata:\n  class: invalid_val\n  category: info\n  requires:\n    bins: [\"gloop\"]\n    cliHelp: h\n---\n\n# t\n"
	if _, err := parseSkillMD([]byte(md)); err == nil {
		t.Error("非法 class 值应该报错")
	}
}
