package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"unicode/utf8"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/skills"
)

// runSkillCmd 是 `gloop skill` 子命令入口。
//
// 设计原则：skill 是 gloop 分发给 agent 的只读 instruction package。
// - 启动时 system prompt 只注入职业可用 skill manifest
// - `gloop skill list` 只返回 manifest；`gloop skill show <name>` 才读取正文
// - 用户不可修改、不可安装、不可卸载（冒险者自身的能力扩展走 CustomPrompt）
func runSkillCmd(_ context.Context, log *slog.Logger, args []string) int {
	_ = log
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop skill <list|show|validate>")
		return 2
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list":
		return runSkillList(rest)
	case "show":
		return runSkillShow(rest)
	case "validate":
		return runSkillValidate(rest)
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: skill %s\n", sub)
		fmt.Fprintln(os.Stderr, "用法: gloop skill <list|show|validate>")
		return 2
	}
}

func runSkillList(args []string) int {
	fs := flag.NewFlagSet("skill list", flag.ContinueOnError)
	classFlag := fs.String("class", "", "只看指定职业可用的技能（warrior | mage），默认全量")
	kindFlag := fs.String("kind", "", "只看指定种类的技能（capability | orchestration）")
	categoryFlag := fs.String("category", "", "只看指定分类的技能")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	reg := skills.Default()
	var list []skills.Skill

	if *classFlag != "" {
		var class model.AdventurerClass
		switch *classFlag {
		case "warrior", "Warrior", "剑士":
			class = model.ClassWarrior
		case "mage", "Mage", "法师":
			class = model.ClassMage
		default:
			fmt.Fprintf(os.Stderr, "未知职业: %s（可选 warrior / mage）\n", *classFlag)
			return 2
		}
		list = reg.ListForClass(class)
	} else {
		list = reg.List()
	}

	// 按 kind / category 过滤
	if *kindFlag != "" {
		filtered := make([]skills.Skill, 0, len(list))
		for _, sk := range list {
			if strings.EqualFold(sk.Kind, *kindFlag) {
				filtered = append(filtered, sk)
			}
		}
		list = filtered
	}
	if *categoryFlag != "" {
		filtered := make([]skills.Skill, 0, len(list))
		for _, sk := range list {
			if strings.EqualFold(sk.Category, *categoryFlag) {
				filtered = append(filtered, sk)
			}
		}
		list = filtered
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		for _, sk := range list {
			_ = enc.Encode(sk.Manifest())
		}
		return 0
	}

	if len(list) == 0 {
		fmt.Println("（无可用技能）")
		return 0
	}

	fmt.Printf("可用技能（%d 个）：\n\n", len(list))
	fmt.Printf("  %-20s  %-10s  %-14s  %-10s  %-8s  %s\n", "NAME", "VERSION", "KIND", "CATEGORY", "CLASSES", "DESCRIPTION")
	for _, sk := range list {
		ver := sk.Version
		if ver == "" {
			ver = "-"
		}
		kind := sk.Kind
		if kind == "" {
			kind = "-"
		}
		cat := sk.Category
		if cat == "" {
			cat = "-"
		}
		classes := classesTag(sk)
		desc := truncateUTF8(sk.Description, 40)
		fmt.Printf("  %-20s  %-10s  %-14s  %-10s  %-8s  %s\n", sk.Name, ver, kind, cat, classes, desc)
	}
	fmt.Println()
	fmt.Println("运行 `gloop skill show <name>` 查看详细说明。")
	return 0
}

func classesTag(sk skills.Skill) string {
	w := sk.WarriorAvailable
	m := sk.MageAvailable
	switch {
	case w && m:
		return "通用"
	case w:
		return "剑士"
	case m:
		return "法师"
	default:
		return "-"
	}
}

func runSkillShow(args []string) int {
	fs := flag.NewFlagSet("skill show", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "用法: gloop skill show <name>")
		return 2
	}
	name := fs.Arg(0)

	reg := skills.Default()
	sk, err := reg.Get(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "技能不存在: %s\n", name)
		names := []string{}
		for _, s := range reg.List() {
			names = append(names, s.Name)
		}
		fmt.Fprintf(os.Stderr, "可用技能: %s\n", joinStr(names, ", "))
		return 1
	}

	// 输出 Markdown 格式，方便 agent 解析（也方便人读）
	fmt.Print(sk.FormatMarkdown())
	return 0
}

// runSkillValidate 校验所有或指定 skill 的格式和内容是否符合规范。
//
// 用法：
//
//	gloop skill validate          # 校验所有内置 skill
//	gloop skill validate <name>   # 校验指定 skill
//
// 返回码：0 全部通过，1 有问题，2 参数错误
func runSkillValidate(args []string) int {
	fs := flag.NewFlagSet("skill validate", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	reg := skills.Default()

	var toValidate []skills.Skill
	if fs.NArg() > 0 {
		name := fs.Arg(0)
		sk, err := reg.Get(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "技能不存在: %s\n", name)
			return 2
		}
		toValidate = append(toValidate, *sk)
	} else {
		toValidate = reg.List()
	}

	// 收集所有校验问题
	type skillIssues struct {
		Name   string                   `json:"name"`
		Issues []skills.ValidationIssue `json:"issues"`
	}
	var results []skillIssues
	totalIssues := 0

	for _, sk := range toValidate {
		issues := sk.Validate()
		if len(issues) > 0 {
			results = append(results, skillIssues{
				Name:   sk.Name,
				Issues: issues,
			})
			totalIssues += len(issues)
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		if results == nil {
			results = []skillIssues{}
		}
		_ = enc.Encode(map[string]interface{}{
			"valid":   totalIssues == 0,
			"total":   len(toValidate),
			"invalid": len(results),
			"issues":  results,
		})
		if totalIssues > 0 {
			return 1
		}
		return 0
	}

	// 文本输出
	if totalIssues == 0 {
		if len(toValidate) == 1 {
			fmt.Printf("✓ %s: 通过校验\n", toValidate[0].Name)
		} else {
			fmt.Printf("✓ 全部 %d 个技能均通过校验\n", len(toValidate))
		}
		return 0
	}

	fmt.Printf("✗ 发现 %d 个校验问题（涉及 %d / %d 个技能）：\n\n",
		totalIssues, len(results), len(toValidate))
	for _, r := range results {
		fmt.Printf("  %s:\n", r.Name)
		for _, iss := range r.Issues {
			fmt.Printf("    - %s: %s\n", iss.Field, iss.Message)
		}
		fmt.Println()
	}
	fmt.Println("运行 `gloop skill show <name>` 查看技能详情。")
	return 1
}

func joinStr(ss []string, sep string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

// truncateUTF8 按 rune 个数截断 UTF-8 字符串，避免把多字节字符切一半。
func truncateUTF8(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	// 取前 maxRunes 个 rune，再追加 "..."
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
