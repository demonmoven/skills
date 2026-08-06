# 请求录制场景

录制 Web 项目请求，结合 Thrift IDL 分析生成接口文档。

## 执行流程

### 0. 参数检查

执行前需要确认以下参数，如果缺少则使用 AskUser 询问用户：
- **目标 URL** - 要录制的网页地址
- **接口过滤正则** - 用于过滤接口的模式
- **本地 IDL 路径** - Thrift IDL 文件路径，用于分析接口定义
- **项目英文名** - 用于生成文档目录

### 1. 启动录制

```bash
flou-cli test record-req <URL> \
   -r <RECORD_HOST> \
   --domain-headers "<DOMAIN>:x-tt-env=<PPE_ENV>,x-use-ppe=1" \
   --application <APPLICATION> \
   --output ${RESULT_DIR}/<application>/record-req/${YYYY-MM-DD-HHMMSS}/request-logs.txt \
   --format text \
   --timeout 60000 \
   --ignore-https-errors
```

参数说明：
- `<URL>` - **必填**，启动后自动导航的测试页面（位置参数）
- `-r, --record-host <host>` - 录制请求的目标域名（用于过滤录制范围，不传则录制所有域名）
- `--domain-headers <headers>` - 域名请求头配置，格式：`<DOMAIN>:header1=value1,header2=value2`
- `--application <name>` - 应用名称，用于 cookies 存储
- `--output <file>` - 输出文件路径（可选，默认 `.flou/test/<session_id>.log`）
- `--format <format>` - 输出格式：`text` 或 `json`
- `--timeout <ms>` - 请求超时时间，默认 30000ms
- `--ignore-https-errors` - 忽略 HTTPS 错误
- `-R, --record-response` - 是否录制响应体（可选）

**执行效果**：
- 命令立即退出，返回基本信息
- 浏览器在后台启动并自动打开目标页面
- 完成登录后自动保存 SSO cookies
- 浏览器保持运行，持续捕获请求

**输出示例**：
```
🚀 录制会话已在后台启动
测试页面: https://example.com

使用以下命令停止录制:
  flou-cli test stop-record -s <SESSION_ID>
```

### 2. 操作页面

在浏览器中操作页面，触发需要录制的请求

### 3. 停止录制并查看结果

```bash
flou-cli test stop-record -s <SESSION_ID>
```

参数说明：
- `-s, --session-id <sessionId>` - 录制会话ID（必填）

**执行效果**：
- 发送信号关闭浏览器
- 清理会话
- 输出日志文件路径

### 4. 查看活跃会话

```bash
flou-cli test list-sessions
```

**输出示例**：
```
SESSION-ID	URL	CREATED-AT
a1b2c3d4e5f6g7h8	https://example.com	2026-03-04 10:30:00
```

### 5. 分析 Thrift 文件

使用用户提供的本地 IDL 路径，分析接口定义：
- 获取 RPC 方法名、参数、返回值
- 提取接口说明注释

### 6. 生成接口文档

输出到 `${RESULT_DIR}/<application>/record-req/requests.md`，格式：

```markdown
# <application> 接口请求文档

> 平台地址: <URL>
> 捕获时间: <YYYY-MM-DD HH:MM:SS>

---

## 接口总览

### <模块名>

| HTTP URL | Method | RPC 方法 | 说明 | 页面触发路径 | 状态 |
|----------|--------|----------|------|-------------|------|
| `/api/xxx` | POST | MethodName | 接口说明 | 触发路径 | ✅/⚠️ 未录制/❌ 已下线 |
```

## 状态标记

- ✅ 已录制
- ⚠️ 未录制
- ❌ 已下线

## 接口详情与请求示例格式

每个接口需补充详细说明：

```markdown
#### <RPC方法名>

<接口说明>

**请求**
```http
POST /api/path
Content-Type: application/json
```

**请求示例**
```json
{
  "Param1": "value1",
  "Param2": "value2"
}
```

**响应示例**
```json
{
  "data": {},
  "baseResp": {
    "StatusCode": 0,
    "StatusMessage": ""
  }
}
```
```

## 注意事项

- 录制会话在后台运行，可以同时启动多个会话
- Session ID 用于标识和管理录制会话
- 浏览器关闭后会话自动失效
- 需要用户提供：目标 URL、Thrift 文件路径、项目名称
- 按模块分类整理接口
