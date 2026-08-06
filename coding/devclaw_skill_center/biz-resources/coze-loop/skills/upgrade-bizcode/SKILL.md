---
name: upgrade-bizcode
description: 在本项目中需要新增或修改业务错误码时使用。
---
# 业务错误码 Skill

当在本项目中需要新增或修改业务错误码时，使用此 Skill 完成错误码定义、代码生成和国际化配置流程。

## 重要原则

- **统一管理**：所有业务错误码在 `backend/script/errorx` 目录下定义
- **自动生成**：不要手动编写业务错误 Go 代码，必须通过脚本生成
- **国际化支持**：配置多语言错误文案，默认英文，需额外配置其他语言

## 错误码格式

错误码为 9 位数字，结构如下：
```
1  2  3  4  5  6  7  8  9
_________________________
|   |  |  |  |  |  |  |  |
|app|    biz    |  sub_code |
‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
```

- `app`: 产品代码 (cozeloop = 6)
- `biz`: 业务域代码（在 metadata.yaml 中定义）
- `sub_code`: 业务域内的子错误码

## 错误码配置文件

在 `backend/script/errorx/` 目录下：
- `metadata.yaml` - 应用和业务域配置
- `common.yaml` - 通用错误码（跨所有业务域共享）
- `{biz}.yaml` - 业务特定错误码（如 evaluation.yaml、prompt.yaml）

## 执行步骤

当用户请求新增或修改业务错误码时，**必须**按以下步骤执行：

### 1. 定义错误码

**新增通用错误码：**
- 编辑 `backend/script/errorx/common.yaml`
- 添加错误码定义：
  ```yaml
  error_code:
    - name: CommonNoPermission
      code: 101
      message: no access permission
      no_affect_stability: true
  ```

**新增业务特定错误码：**
- 编辑对应的 `backend/script/errorx/{biz}.yaml`
- 添加错误码定义：
  ```yaml
  error_code:
    - name: BalanceInsufficient
      code: 3001
      message: user balance is insufficient
      description: user balance is insufficient
      no_affect_stability: true
  ```

**新增业务域：**
- 在 `backend/script/errorx/metadata.yaml` 中添加业务域
- 创建对应的 `{biz}.yaml` 文件

### 2. 生成错误码 Go 代码

- **必须**使用 Bash 工具执行：`.claude/skills/upgrade-bizcode/scripts/run.sh {biz}`
- 其中 `{biz}` 为业务域名称（如 evaluation、prompt）
- **等待**脚本完成，不要中断
- **检查**输出中是否有错误信息
- 如果失败，**报告**错误并停止

生成的代码位于：`backend/module/{biz}/pkg/errno/`

### 3. 配置国际化文案

在以下目录配置错误码对应的国际化文案：
```json
{
  "Docker": "release/deployment/docker-compose/conf/locales/",
  "Kubernetes": "release/deployment/helm-chart/umbrella/conf/locales/"
}
```

**配置说明：**
- 每个文件对应不同的语言（如 `en-US.yaml`、`zh-CN.yaml`）
- 错误码默认文案为英文，可以不在 `en-US.yaml` 中配置
- 其他语言需要配置，格式：
  ```yaml
  "600500101": "无访问权限"
  "600500202": "参数无效"
  ```
- Key 为 9 位错误码，Value 为对应语言的文案
- 确保在 **2 个位置**（Docker + Kubernetes）都配置

## 使用错误码

在代码中返回业务错误：
```go
return errorx.NewByCode(errno.CommonNoPermissionCode)
```

## 使用示例

- 为某业务域新增特定错误码
- 新增通用错误码
- 为错误码添加中文翻译
- 创建新的业务域及其错误码

## 注意事项

- 不要手动编写错误码 Go 代码，必须通过脚本生成
- 错误码在业务域内必须唯一
- 配置国际化文案时同时修改 Docker 和 Kubernetes 的配置
- 英文为默认文案，其他语言需显式配置