# i18n key 命名规则（v0.5.5 Slice 1）

## 命名格式

```
namespace.section.element.action/state
```

- 全小写，点号分层，camelCase 用于多词（如 `feed.composer.placeholder`）。
- 不用中文当 key，不用页面行号当 key，不把整句动态拼接拆成半句。

## namespace 定义

| namespace | 用途 | 示例 |
|-----------|------|------|
| `nav` | 侧边栏/顶栏导航 | `nav.workbench`, `nav.settings` |
| `brand` | 品牌文案 | `brand.tagline` |
| `common` | 跨页面复用的按钮/状态 | `common.action.cancel`, `common.state.loading` |
| `topbar` | 顶栏专属文案 | `topbar.search.placeholder`, `topbar.view.feed` |
| `feed` | Feed 流专属文案 | `feed.composer.placeholder`, `feed.empty.noActivity` |
| `quest` | 委托相关 | `quest.workflow.direct.title` |
| `world` | gloop 世界观词（保留中文，不翻译） | `world.warrior`, `world.mage` |
| `term` | 技术/流程术语（保留英文，不翻译） | `term.checker`, `term.fanout` |

## world / term 词汇定调

**world（保留中文）**：冒险者、剑士、法师、委托、公会
**term（保留英文）**：Agent、Checker、Fanout、Evidence、Policy、Ledger、Runtime、Loop、Delegation、Delivery、Exception

未来加 en locale 时，`world.*` 和 `term.*` 基本原样复用，不翻译。

## 动态文案规则

必须用参数，不许字符串拼接：

```ts
// 正确
t('feed.reply.hiddenCount', { count })
t('nav.version', { version })

// 错误
'展开 ' + count + ' 条回复'
```

## 渐进迁移

- Provider 包了但不强制，未迁移组件继续硬编码不影响渲染。
- 同一页面半边 `t()`、半边硬编码是过渡态，slice 内完成该页面全部迁移。
- 不迁移代码注释、测试 fixture、日志、后端返回字符串，只迁移用户可见 UI 文案。
