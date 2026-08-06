# 工作流初始化

> **入口判定**：用户说"创建工作流"、"初始化 bits flow"、"新建 spec 工作流"、"flow init"、"初始化工作流"等。

在当前项目中初始化 BitsCLI 工作流目录，并引导用户创建第一个 change。

---

## Step 1：初始化 BitsCLI

在用户当前项目根目录下执行：

```bash
bitscli flow init
```

命令会交互式完成 schema 选择、AI 工具配置等，成功后生成 `flux/` 目录（含 `config.yaml`、`schemas/`、`specs/` 等）。

**成功标志**：命令输出包含 `Initialized BitsCLI` 或类似确认信息，且项目根目录出现 `flux/` 目录。

**失败处理**：若命令报错，按 SKILL.md 公共错误处理规则向用户展示错误信息。

---

## Step 2：引导创建 Change

初始化成功后，向用户展示：

```
✅ BitsCLI 工作流已初始化完成，项目下已生成 flux/ 目录。

接下来可以使用 /flux-flow-new 创建一个新的 change，进入需求澄清和 spec 编写流程。
```

本步骤到此结束，后续由 `/flux-flow-new` skill 接管。
