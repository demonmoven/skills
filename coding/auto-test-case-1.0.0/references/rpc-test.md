# RPC接口测试模块

## 功能说明
使用gdpa-cli的bam-query工具完成RPC接口测试，支持参数传递、结果验证和批量执行。

## 前置条件
1. 已安装gdpa-cli
  - 安装命令：`bash -c "$(curl -fssL https://gdp.bytedance.net/open_api/tool/gdpa_cli/install.sh)"`
  - 介绍（https://bytedance.larkoffice.com/wiki/VKm8wGX2diqZz7kldNgcoe98nBh）
  - 执行登录：`gdpa-cli login`

2. **前置环境确认**
   - 创建Session：`gdpa-cli create-session`
   - 在后续命令中必须使用 `--session-id` 参数传递session_id
   - env, 确认用户需要测试 env 环境
   - vregion 使用 "China-North"

## 使用步骤
### 1. 准备测试参数
创建JSON格式的参数文件（例如：rpc_params.json），包含接口所需的参数。


### 2. 校准参数
询问用户是否需要校准 idl 文件，如果提供，优化参数字段类型、格式、拼写。
如：类型不匹配，进行修正

### 3. 执行测试命令
```bash
# 步骤1: 创建Session
SESSION_ID=$(gdpa-cli create-session | grep -o 'sess_[^ ]*')

# 步骤2: 执行RPC请求（必须使用session_id）
gdpa-cli run bam-query --session-id $SESSION_ID --input '{
  "action": "rpc",
  "psm": "<service_name>",
  "func_name": "<method_name>",
  "request": "<json_params>",
  "vregion": "China-BOE"
}'

# 示例
gdpa-cli run bam-query --session-id $SESSION_ID --input '{
  "action": "rpc",
  "psm": "user.service.v1.User",
  "func_name": "GetUser",
  "request": "{\"user_id\":123}",
  "vregion": "China-North",
  "env": "prod"
}'
```

### 4. 结果分析
- 成功：返回接口响应数据
- 失败：显示错误信息（如参数错误、服务不可达等）

## 注意事项
- 确保服务名和方法名与RPC接口定义一致
- 参数文件需符合JSON格式规范
- 复杂场景可结合脚本实现批量测试