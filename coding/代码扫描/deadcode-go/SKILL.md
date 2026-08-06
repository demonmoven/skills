---
name: deadcode-go
description: 用 deadcode（golang.org/x/tools/cmd/deadcode）扫描 Go 项目不可达函数/方法并协助清理；用于“清理死代码/无用代码”等需求。安装与删除前需用户确认；无 main 入口的 lib 仓库需先确认是否适用（仅 -test 或指定入口）。
version: 1.0.1
tags: [go]
---

# Go Deadcode Cleaner

按以下步骤使用 `deadcode` 扫描并在用户确认后删除无用代码。

## 1. 判断仓库类型（是否有入口）

1) 检查是否存在 `main` 包（可执行入口）：

```bash
go list -f '{{if eq .Name "main"}}{{.ImportPath}}{{"\n"}}{{end}}' ./...
```

2) 若没有任何输出，视为 **无 main 入口（lib/SDK 类型）**，必须先问用户：

- “该仓库没有 `main` 入口。`deadcode` 基于入口可达性分析，可能无法代表真实‘无用代码’。是否仍要使用 `deadcode`？如果继续：优先仅分析测试（`-test`），还是你能提供/指定一个入口包进行分析？”

只有用户明确同意后才能继续。

## 2. 检查并安装工具（安装前必须确认）

1) 检查 `deadcode` 是否可用：

```bash
command -v deadcode || true
```

2) 若未安装，必须先询问用户是否允许安装；用户同意后再执行：

```bash
go install golang.org/x/tools/cmd/deadcode@latest
```

若用户的 `PATH` 未包含 `GOBIN`/`GOPATH/bin`，用 `$(go env GOPATH)/bin/deadcode` 运行。

## 3. 与用户确认扫描配置

在运行前，必须让用户确认这些参数（给出推荐默认值）：

- 扫描范围：默认 `./...`（或用户指定的 `./cmd/...`、`<module>/...` 等）
- 是否包含测试：推荐默认开启 `-test`
- 构建 tags：如项目依赖 tags，询问用户是否需要 `-tags=...`
- 是否包含生成文件：默认不包含；如确需覆盖生成文件，再加 `-generated`

## 4. 扫描并生成“候选删除清单”

执行扫描（以用户确认的范围/flags 为准）：

```bash
deadcode -test <packages...>
```

需要机器可读输出时用：

```bash
deadcode -test -json <packages...>
```

把结果整理为候选清单，并展示给用户，至少包含：

- `package path`
- `symbol name`（func / method）
- `position`（file:line:col）

同时提示用户：deadcode 报告的是“从入口不可达”，并不等于绝对可删（例如不同构建配置、接口约束、`//go:linkname` 等场景）。

## 5. 删除前必须二次确认（删除范围确认）

在修改任何代码之前，必须向用户明确确认：

- “确认要删除哪些包/哪些符号？”
- “是否按批次删除并每批做一次编译/测试验证？”

只有用户确认删除范围后才能开始删除。

## 6. 实施删除与验证

1) 按用户确认范围进行删除（优先小批量）。
2) 每批删除后做最小验证：优先运行仓库自带构建脚本；否则运行 `go test`/`go build`（仅在用户允许或用户要求时）。
3) 若出现编译/测试失败：回退该批次或缩小范围，并把失败原因与依赖关系反馈给用户。

## 7. 解释“为什么某函数不是 dead”（可选）

当用户想知道某函数为何未被报告为 dead，使用：

```bash
deadcode -test -whylive=<pkg.FuncOrType.Method> <packages...>
```