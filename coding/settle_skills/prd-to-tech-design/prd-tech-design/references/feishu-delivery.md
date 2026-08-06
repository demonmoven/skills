# 飞书交付

技术方案 Markdown 草稿完成后使用本参考。该 skill 的最终交付物是飞书/Lark 文档。

## 创建规则

默认创建飞书文档，除非用户明确说不需要。

推荐标题：

```text
<PRD 标题或项目名>技术方案
```

先将草稿写入临时 Markdown 文件，再从该文件创建文档：

```bash
lark-cli docs +create --title "<title>" --markdown @/path/to/draft.md
```

当需要访问用户空间或源 PRD 上下文时使用 `--as user`。本地 keychain 访问受沙箱限制时，使用授权方式执行。

## 内容要求

- 保留 `tech-design-template.md` 中的标准技术方案章节。
- Markdown 草稿中保留 Mermaid 图。如果用户明确要求原生飞书画板，再使用 `lark-doc` 和画板相关 skill 插入并填充图表。
- 文档中必须包含简洁证据引用，尤其是账户流、结算明细/汇总边界、IDL、常量、BP/XML、代码路径。
- 存在假设时必须包含 `待确认事项` 章节。
- 不要把终端日志或原始 JSON 粘贴到文档中。

## 失败处理

如果 `lark-cli docs +create` 失败：

- `keychain Get failed: keychain not initialized`：如果可用，使用授权方式在沙箱外重试；如果仍失败，报告 keychain 阻塞并保留本地 Markdown 草稿。
- `need_user_authorization`：通过 `lark-cli auth login --scope "<scope>"` 申请最小缺失 scope，授权后重试。
- 目标文件夹/文档空间 permission denied：报告权限阻塞并保留本地 Markdown 草稿。

遇到 keychain 失败时，不要反复用同一个沙箱命令重试。最终回复必须包含以下之一：

- 已创建的飞书文档 URL/token；或
- 阻塞原因 + 本地 Markdown 草稿路径。

## 最终回复格式

创建成功时：

```text
已生成飞书技术方案文档：<url>
本次主要覆盖：...
待确认：...
```

创建失败时：

```text
飞书文档创建被阻塞：<reason>
已保留本地 Markdown 草稿：<path>
```
