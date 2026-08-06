# 2.6 命名空间 / 包管理

> **本节目标**：理解 Agent 能力如何通过 Plugin 和 Marketplace 进行组织、分发和复用。

---

## 概念

当团队积累了大量 Skills 和工具后，就需要解决**组织和分发**的问题：

- 谁的 Skill 放在哪里？（命名空间）
- 怎么安装到项目中？（包管理）
- 怎么发现好用的 Skill？（Marketplace）

这和传统软件工程中的 npm / pip / cargo 包管理是同一个问题，只不过管理的对象从"代码库"变成了"Agent 能力包"。

---

## 子章节

- 2.6.1. [Plugin / Marketplace](026a-plugin-marketplace.md)

---

## 为什么需要命名空间

```text
Without namespace:
  .claude/skills/
    speckit/          ← 谁写的？哪个版本？
    deploy/           ← 和另一个团队的 deploy 冲突了
    fixloop/          ← 更新了吗？

With namespace (Plugin):
  .claude/skills/
    xdev/             ← xdev plugin 的命名空间
      speckit/
      deploy/
      fixloop/
    bytedcli/         ← bytedcli plugin 的命名空间
      tce/
      log/
```

---

[上一节：流程控制](025-flow-control.md) | [返回上级：Agent 技术栈](020-agent-tech-stack.md)
