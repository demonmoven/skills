# Design From Directory Command

基于已有参考实现目录，补齐或重建 `component-design.md`。

## 重要规则

1. 开始前必须读取 `.claude/docs/constitution.md`，遵守本工程技术规范。
2. 必须读取真实代码文件后再写文档，禁止凭空生成组件说明。
3. 组件文档规范统一遵循 [../references/component-design-guidelines.md](../references/component-design-guidelines.md)。

## 调用方式

```bash
x-ui-workflow /design-from-directory <reference_dir> [output_path]
```

参数说明：
- `reference_dir`：已有参考实现目录
- `output_path`：可选输出路径；默认写到 `<reference_dir>/component-design.md`

## 执行流程

## Step 1: 校验目录与输出路径

1. 确认 `reference_dir` 存在。
2. 若未提供 `output_path`，默认输出到：

```text
<reference_dir>/component-design.md
```

3. 若输出目录不存在，先创建最小必要目录。

## Step 2: 读取真实实现文件

按以下优先级收集代码：

1. `components/*/implement.tsx`
2. `page/implement.tsx`
3. 若上述文件不存在，退化为读取目录下与组件实现相关的 `.tsx` / `.ts` / `.jsx` / `.js` 文件

同时可补充读取以下文件（如果存在）：
- `summary.md`
- `ui.md`
- `ui.implement.xml`

信息冲突时，以参考实现代码为准。

## Step 3: 生成 `component-design.md`

根据真实代码和补充文档，生成组件设计文档，并写入 `output_path`。

生成要求：
- 组件名、路径、层级必须与实际代码一致
- 若目录同时包含页面与业务组件，二者都要覆盖
- ASCII 结构图应反映真实布局，不要写成抽象概念图
- 职责描述聚焦 UI 组织与复用边界，不重复业务逻辑实现细节

## Step 4: 返回最终结果

1. 读取最终生成的 `component-design.md`
2. 返回输出路径
3. 将完整 Markdown 内容作为回复主体返回
