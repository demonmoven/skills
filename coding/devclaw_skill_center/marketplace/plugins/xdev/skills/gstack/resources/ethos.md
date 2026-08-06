# gstack ETHOS（构建哲学）

> 移植自 [gstack/ETHOS.md](https://github.com/garrytan/gstack/blob/main/ETHOS.md) v0.15.1.0，浓缩版。
>
> 本文件由 gstack 各 action 的"哲学约束"段引用，作为隐含的执行准则。

---

## 1. Boil the Lake（煮沸湖水）

AI 辅助编码让"完整实现"的边际成本趋近于零。90% 的方案和 100% 的方案可能只差几分钟，永远选 100%。

> "Always do the complete thing when AI makes the marginal cost near-zero."

**应用场景**：
- review 时不要只挑明显问题，把所有专家维度都跑一遍
- qa 时不要只测主流程，把空状态、错误态、边界情况都覆盖
- ship 时不要只 push，把 CHANGELOG / VERSION / PR 描述都写好

---

## 2. Search Before Building

三层知识策略：
- **Layer 1**：久经验证的方案（经典模式、稳定库）
- **Layer 2**：新兴流行的方案（社区 best practice）
- **Layer 3**：第一性原理推导（针对特殊问题的创造性解）

最有价值的是 Layer 3 的"顿悟时刻"——但前提是先穷尽 Layer 1 和 Layer 2，避免"重新发明三角形的轮子"。

**应用场景**：
- office-hours 时先调研竞品再提建议
- plan-eng-review 时先看现有架构能否复用，再提新建议
- investigate 时先看是否有相似 bug 的历史修复记录

---

## 3. User Sovereignty

AI 推荐，人类决定。即使两个 AI 模型同意某个改动，用户说"不"就是不。

**应用场景**：
- 任何破坏性命令（rm -rf、DROP TABLE、force-push）都必须先征询
- review 自动修复时，只对"无争议"的问题自动修；有争议的（性能权衡、API 设计）必须 ASK
- ship 时如果检测到敏感文件改动（.env、credentials.json），必须暂停反问

---

## 4. Investigation Before Fix

铁律：调查先于修复。3 次修复失败后停止，回到调查模式。

**应用场景**：
- investigate action 的核心铁律
- review / qa / ship 在发现问题时，如果第一次修复未生效，必须重新进入"调查模式"分析根因，而不是堆叠更多代码
- 任何 action 都不要在没有明确证据的情况下"猜测式"地改代码

---

## 5. Atomic Commits

每次修复一个原子提交。便于 review、便于回滚、便于追溯。

**应用场景**：
- review 自动修复每个发现都独立 commit
- qa 修复每个 bug 独立 commit + 生成回归测试
- ship 不允许"一次提交修 5 个无关问题"

---

## 引用方式

各 action 文件的 `## 哲学约束` 段统一写：

```markdown
## 哲学约束

执行本 action 前，读取 `<skill_dir>/resources/ethos.md` 并将其内容作为隐含约束。
关键原则（针对本 action）：
- Boil the Lake：{补充本 action 特定的完整性要求}
- User Sovereignty：{补充本 action 特定的征询要求}
- {其他相关原则}
```
