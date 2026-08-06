# gloop 移动端修复 + 默认冒险者 Spec v0.2.3

> 版本：v0.2.3
> 状态：已落地
> 前置：v0.2.2 前端 HOTL 深度适配后，移动端 review 发现 pipeline 回归 + 窄屏密度 + 断点碎片化；用户要求补默认剑士/法师

---

## 一、缺口与方案

### 1.1 pipeline 767px 回归 bug（必须修）
v0.2.2 把 pipeline 从 5 段改 3 段，但 styles.css:2958 `@media(max-width:767px)` 还硬编码 `repeat(5, minmax(56px, 1fr))`，3 个 step 塞 5 列有空隙。
**方案**：改 `repeat(3, 1fr)`。

### 1.2 窄屏影响日志信息密度过高
每条 5 元素（徽章+摘要+文件数+effect+时间），720px 以下 padding 才 5px 7px，挤。
**方案**：720px 以下只留徽章+摘要+文件数，effect 和时间 `display:none`；汇总头 effect 标签窄屏隐藏。

### 1.3 移动端隐藏 board 视图
board 横向 kanban 小屏体验弱，移动端主力是 focus。
**方案**：768px 以下隐藏 board 切换按钮，只留 focus+list。

### 1.4 断点收敛（根因治理）
96 个 @media，11 种断点（480/720/760/767/768/820/860/920/1000/1024/1080/1100）。pipeline 回归就是 720 改了 767 没改。
**方案**：收敛到 3 个：480/768/1024。把 720/760/767→768，820/860→768（小屏）/1024（中屏），920/1000/1080/1100→1024。

### 1.5 默认剑士/默认法师（新功能）
每次创建 quest 都要手动选 warrior/mage，重复操作。
**方案**：
- GlobalConfig 加 `DefaultWarriorID`/`DefaultMageID`
- GlobalConfigPatch 加对应指针字段
- Settings 页加配置入口
- CreateQuestSheet 打开时如果 config 有默认值且用户没手动选，预填默认
- createQuest 后端：req 字段为空时填 config 默认值

## 二、落地清单

| 文件 | 改动 |
|------|------|
| `web/src/styles.css` | pipeline 767px repeat(5)→repeat(3)；窄屏影响日志精简；board 切换隐藏；断点收敛 |
| `web/src/pages/FocusView.tsx` | 无（pipeline 已是 3 段，CSS 修即可）|
| `web/src/components/AppTopbar.tsx` | board 切换按钮加 mobile-only 隐藏类 |
| `internal/fsstore/config.go` | GlobalConfig + Patch 加 DefaultWarriorID/DefaultMageID |
| `internal/server/api_quests.go` | createQuest: warrior/mage 空时填 config 默认 |
| `web/src/components/CreateQuestSheet.tsx` | 打开时 fetch settings 预填默认 warrior/mage |
| `web/src/pages/Settings.tsx` | 加默认剑士/法师配置 UI |

## 三、验证
- go test ./... 全绿；go build + tsc + vite build 通过
- 版本：0.2.2 → 0.2.3
- 断点收敛结果：96 个 @media 从 21 种断点值收敛到 5 种（max-width: 480/768/1024 + min-width: 768/1024）
