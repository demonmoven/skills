# Test Case Format Comparison

以下是同一个测试场景用三种格式的写法对比。

## 测试场景："用户能在聊天页面发送一条消息并看到回复"

---

### Format 1: Natural Language (自然语言) — `chat-send.txt`

```
用户能在聊天页面发送一条消息并看到AI回复
```

就这一句话。Test Runner AI 会自动：
- 推断目标 URL（`/session` 或首页）
- 找到输入框，输入一段测试文本
- 找到发送按钮并点击
- 等待 AI 回复出现
- 判断测试是否通过

**优点**：门槛最低，任何人都能写
**缺点**：AI 的理解可能与预期不同，不够精确

---

### Format 2: Checklist (清单) — `chat-send.md`

```markdown
## 发送消息并收到回复

> 前置条件：用户已登录，在 /session 页面

- [ ] 在聊天输入框中输入"你好，请介绍一下你自己"
- [ ] 点击发送按钮（或按 Enter）
- [ ] 等待 AI 回复出现（最多 30 秒）
- [!] 回复内容非空，且显示在消息列表中
- [!] 消息列表中同时可见用户消息和 AI 回复
```

**优点**：比自然语言更精确，比 YAML 更易读
**缺点**：需要了解 checklist 语法

---

### Format 3: YAML (结构化) — `chat-send.yaml`

```yaml
web:
  url: http://localhost:8090/session

tasks:
  - name: "发送消息并收到回复"
    flow:
      - aiWaitFor: "聊天输入框可见"
        timeout: 15000

      - aiAct: "在聊天输入框中输入'你好，请介绍一下你自己'"

      - aiAct: "点击发送按钮"

      - aiWaitFor: "AI回复消息出现在消息列表中"
        timeout: 30000

      - aiAssert: "消息列表中同时包含用户发送的消息和AI的回复，回复内容非空"

      - recordToReport: "消息发送测试"
        content: "验证了用户能发送消息并收到AI回复"
```

**优点**：最精确，支持超时/多步骤/元数据
**缺点**：写起来最繁琐

---

## 推荐用法

| 场景 | 推荐格式 | 原因 |
|------|---------|------|
| 快速验证/冒烟测试 | 自然语言 | 5 秒写完一条 |
| 功能验收测试 | Checklist | 可读性好，团队容易 review |
| 回归测试套件 | YAML | 精确可重复 |
| PRD 直接转测试 | 自然语言 | 从需求文档直接复制粘贴 |
| CI/CD 集成 | YAML | 结构化便于解析统计 |

## 混合使用

一个 `e2e-cases/` 目录可以同时包含三种格式：

```
e2e-cases/
├── phase-1-smoke/
│   ├── homepage-loads.txt              # 自然语言
│   └── basic-navigation.txt            # 自然语言
├── phase-2-chat/
│   ├── chat-features.md                # Checklist
│   └── phase-2-case-1-send-msg.yaml   # YAML
└── phase-10-us1-full-flow/
    └── complete-user-journey.md        # Checklist
```

fixloop 会根据文件扩展名自动识别格式。
