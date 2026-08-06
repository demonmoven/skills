# Fixloop Universal v2 - 新仓库接入清单

## 概述

将 fixloop 接入一个全新的前端仓库，需要收集以下信息并填入 `fixloop-profile.yaml`。
本文档按**必须知道**和**可选了解**分类。

**v2 新特性**：基于 Playwright MCP 的浏览器自动化、自动依赖安装、时序观测、状态探索。

---

## 一、首次运行：自动安装

v2 的 `scripts/setup.sh` 会在首次运行时自动安装所有依赖：

| 依赖 | 用途 | 安装方式 |
|------|------|---------|
| `@playwright/mcp` | 浏览器自动化 | 全局 npm |
| Chromium 浏览器 | 浏览器运行时 | `npx playwright install`（`system_chrome` 模式可跳过） |
| `pixelmatch` / `pngjs` | 视觉回归对比 | skill 本地 npm |
| `gremlins.js` | 随机探索测试 | skill 本地 npm |
| `@axe-core/playwright` | 无障碍检查 | skill 本地 npm |
| `@metoto/playwright-trace-analyzer-mcp` | 时序分析 | skill 本地 npm |

**用户无需手动安装任何东西**。只要有 Node.js >= 18，运行 `/fixloop-universal run` 时会自动完成安装。

---

## 二、必须知道的信息

### 1. 项目基础信息

| 信息项 | Profile 字段 | 如何获取 | 示例 |
|--------|-------------|---------|------|
| 项目名称 | `project.name` | 人工提供 | `my-react-app` |
| 主应用相对路径 | `project.app_path` | 查看仓库目录结构 | `apps/web` 或 `.` |
| UI 框架 | `project.framework` | 查看 package.json | `react` |
| 构建工具 | `project.build_tool` | 查看配置文件 | `vite` |

### 2. 环境启动命令

| 信息项 | Profile 字段 | 如何获取 | 示例 |
|--------|-------------|---------|------|
| 依赖安装命令 | `setup.install_command` | README 或 package.json | `npm install` |
| Dev Server 启动命令 | `setup.dev_server.start_command` | package.json scripts | `npm run dev` |
| 端口注入方式 | `setup.dev_server.port_injection` | 查看构建工具文档 | `arg:--port` |

**端口注入方式参考**：

| 构建工具 | 通常方式 | 验证方法 |
|---------|---------|---------|
| Vite | `arg:--port` | `npx vite --port 8080` |
| Webpack | `arg:--port` | `npx webpack serve --port 8080` |
| Rsbuild | `config:rsbuild.config.ts:server.port` | 查看配置文件 |
| Next.js | `arg:-p` | `next dev -p 8080` |
| Nuxt | `env:PORT` | `PORT=8080 nuxt dev` |

### 3. 认证方式

| 策略 | 适用场景 | v2 实现 |
|------|---------|--------|
| `none` | 无需登录 | 直接导航 |
| `form_login` | 表单登录 | Playwright a11y tree 自动填写 |
| `cookie_inject` | SSO/OAuth | `--storage-state` 预加载 |
| `local_storage` | Token 认证 | `--storage-state` 预加载 |
| `custom` | 自定义脚本 | `browser_console_execute` |

**判断流程**：
```
打开应用首页
├─ 直接显示内容 → none
├─ 跳转到登录表单 → form_login
├─ 跳转到 SSO/OAuth → cookie_inject
└─ 白屏/401 → local_storage 或 cookie_inject
```

### 4. 测试用例

三种格式混用（按扩展名自动识别）：

| 格式 | 扩展名 | 适用场景 |
|------|--------|---------|
| 自然语言 | `.txt` | 快速冒烟测试 |
| 清单 | `.md` | 功能验收 |
| YAML | `.yaml` | 回归测试、CI |

---

## 三、v2 新增可选配置

### 5. 浏览器模式

| 模式 | 配置值 | 适用场景 |
|------|--------|---------|
| 系统 Chrome（默认） | `system_chrome` | 使用用户真实浏览器 |
| 连接已有 Chrome | `cdp_connect` | 带用户 profile/cookie |
| 独立 Chromium | `bundled` | 隔离环境 |
| 无头模式 | `headless` | CI/CD |

### 6. 时序观测

```yaml
observation:
  enabled: true       # 默认开启，录制每个动作的 DOM/网络/控制台
  trace_retention: 50 # 最多保留 50 个 session
```

### 7. 状态探索

```yaml
exploration:
  enabled: false      # 默认关闭，需显式开启
  budget: 10          # 每个测试后最多探索 10 次交互
  strategy: a11y_driven  # 或 boundary / chaos
  excluded_elements:
    - delete
    - logout
```

---

## 四、接入步骤

### 项目维护者（一次性）

```bash
# 1. 复制示例配置
cp <skill-path>/assets/profiles/react-vite.yaml fixloop-profile.yaml

# 2. 编辑 project / setup / auth 段
# 3. 提交到仓库
git add fixloop-profile.yaml && git commit
```

### 使用者（每次）

```bash
# 1. 写测试用例
mkdir -p e2e-cases/phase-1-smoke
echo "首页能正常加载" > e2e-cases/phase-1-smoke/smoke.txt

# 2. 运行（首次会自动安装依赖）
/fixloop-universal run
```

---

## 五、v1 → v2 迁移对照表

| v1 | v2 | 说明 |
|----|----|------|
| Chrome DevTools (chrome-in-chrome) | Playwright MCP (`@playwright/mcp`) | 无需 Chrome 扩展 |
| 截图 → AI 猜根因 | timeline.json → 因果链分析 | 时序证据替代猜测 |
| 仅执行预定义用例 | 预定义 + 状态探索 | 自动发现边界 bug |
| AI 看截图判断 | axe-core + pixelmatch + console + network | 工程化多信号断言 |
| `mcp__claude-in-chrome__*` | `browser_*` (Playwright MCP) | 工具名更简洁 |
| cookie 注入用 JS | `--storage-state` 预加载 | 更可靠 |
| 手动安装 Chrome 扩展 | `scripts/setup.sh` 自动安装 | 开箱即用 |

**Profile 完全兼容**：v1 的 profile 在 v2 中直接可用，新增字段均有默认值。

---

## 六、常见问题

### Q: 端口注入不生效怎么办？
A: 1) 查看构建工具文档确认支持方式 2) 检查配置文件有无硬编码端口 3) 使用 `config:<file>:<key>` 方式 4) 最后手段：`port_injection: none` + 固定 `port`

### Q: SSO 认证怎么绕过？
A: 1) 从浏览器登录后提取 cookie/token 2) 使用 `cookie_inject` 或 `local_storage` 策略 3) 凭据放环境变量（`{{FIXLOOP_SESSION_TOKEN}}`）。v2 中通过 `--storage-state` 预加载更可靠。

### Q: 没有 spec 文档能用 fixloop 吗？
A: 可以。跳过 `gen-tests`，直接手写测试用例放入 `cases_dir`。

### Q: 如何使用已有的 Chrome 登录状态？
A: 设置 `browser.mode: cdp_connect`，先手动启动 Chrome：`google-chrome --remote-debugging-port=9222`，然后在 profile 中配置 `browser.cdp_endpoint: "ws://localhost:9222"`。

### Q: 探索层会不会搞坏我的应用？
A: 探索层有安全约束：不点击删除/logout 类元素、导航离开后自动返回、崩溃时自动停止。通过 `exploration.excluded_elements` 可配置额外排除规则。

### Q: setup.sh 安装的依赖在哪？
A: 安装在 `<skill-dir>/scripts/node_modules/`，不会污染你的项目 node_modules。
