# 3.2 Post Task Completion

> **本节目标**：理解"任务完成后"还需要做什么——自动化验收和知识复利。

---

## 概念

**Post Task Completion** 关注的是代码"写完"之后的两件事：

1. **验收**：代码能不能跑？UI 对不对？（E2E / UI 还原）
2. **复利**：这次踩的坑能不能防止下次再踩？（Compound 复利工程）

```text
Pre Task Completion ──→ Post Task Completion
 (需求→调研→方案→拆解→编码落地)     (E2E / UI 验收 + 知识复利)
                                          │
                                          ├── E2E
                                          │   "代码跑起来了吗？"
                                          │
                                          ├── UI 还原
                                          │   "UI 对吗？"
                                          │
                                          └── Compound 复利工程
                                              "这次的经验怎么防止下次再踩？"
```

---

## 为什么 Post 也很重要

Agent 的代码生成速度很快，但有两个常见问题：

1. **"代码落地 ≠ 功能正确"**：Pre Task Completion 交付的代码可能通过了单测，但端到端集成和 UI 还原仍需验证
2. **"交付了 ≠ 学到了"**：这次踩的坑如果不沉淀，下次还会踩

Post Task Completion 就是解决这两个问题的。

---

## 子章节

- 3.2.1. [E2E](032a-e2e.md)
- 3.2.2. [UI 还原](032b-ui.md)
- 3.2.3. [Compound（复利工程）](032c-compound.md)

---

[上一节：Pre Task Completion](031-pre-task-completion.md) | [返回上级：研发范式](030-dev-paradigm.md)
