# 探索模块

## 常量
- RESULT_DIR: `${PROJECT_DIR}/.flou/test/`

## 测试标准
- 每个步骤重试次数 <= 2 次能够完成用例元素定位

## 执行流程

1. 询问用户应用名称和模块
   - 使用 AskUserQuestion 获取 application 和 model

2. 启动录制并注入环境变量（后台执行）
   - 参考 `record-req.md`

3. 使用 agent-browser 命令打开网页
   - 始终连接到 --cdp 9222
   - navigate 到目标 URL

4. 获取页面快照
   - snapshot -i 获取所有可交互元素

5. 分析元素定位
   - 优先使用 text 标记
   - 避免使用 class 标记（可能重复）
   - 不要使用 placeholder 定位（不同用户看到的可能不同）

6. 结束录制
   - 参考 `record-req.md`

7. 保存结果
   - 创建 RESULT_DIR 目录
   - 保存到 `${RESULT_DIR}/<application>/explore/<model>_explore_result.md`
   - 格式要求：
     - 完成路径简介
     - 从项目主页触发的完整操作路径

## 示例

### 登录模块

**完成路径简介**: 首页-登录

**完整操作路径**:
1. navigate 到 https://example.com
2. click text="登录"
3. type text="用户名" value="test@example.com"
4. type text="密码" value="password123"
5. click text="提交"
