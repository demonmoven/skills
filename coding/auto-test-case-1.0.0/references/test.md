# 测试模块

## 常量
- RESULT_DIR: `${PROJECT_DIR}/.flou/test/`
- CDP_PORT: 9222

## 执行流程

### 1. 读取用例
- **方式1**：按照用户输入解析
- **方式2**：从 `$(RESULT_DIR)/<application>/explore/<model>_explore_result.md` 读取元素定位信息

### 2. 使用 agent-browser 执行测试

**执行步骤**：

1. **启动录制并注入环境变量（后台执行）**
   - 参考 `record-req.md`


2. **Navigate 到目标 URL**
   ```bash
   agent-browser navigate <url> --profile <path> --cdp 9222 --headed
   ```

3. **执行操作**
   - 始终连接到 --cdp 9222
   - 根据探索结果中的元素定位执行操作
   - click/type/hover 等操作
   - 每个操作失败时重试最多 2 次，不通过则需要优化用例描述

4. **结束录制**
   - 参考 `record-req.md`


### 3. 验证结果
- 检查页面状态
- 验证操作是否成功

### 4. 保存测试报告
- 保存到 `${RESULT_DIR}/<application>/test/${YYYY-MM-DD-HHMMSS}/test_report.md`

## 测试报告格式
```
## 基本信息
- **测试开始时间**: 2026-02-12 15:58:37
- **测试结束时间**: 2026-02-12 16:02:15
- **测试耗时**: 3分38秒
- **物料**: 3298166842599995 (UID)
- **日志文件**: ${RESULT_DIR}/<application>/test/${YYYY-MM-DD-HHMMSS}/request-logs.txt
## 结果
- **测试结果**: 通过/失败
- **失败表现说明**: 仅失败用例需要
```