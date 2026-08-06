# API Test References Index

本文档是 `api-test` 的 reference 导航入口。除 `SKILL.md` 外，Agent 读取其他 reference 前，应先通过本文件定位对应材料，避免在多文件之间跳错流程或忽略强制检查点。

## 读取顺序

| 场景 | 必读文件 | 按需文件 | 完成信号 |
|------|----------|----------|----------|
| 执行一次接口测试 | `flows/test-flow.md` | `refs/cli-reference.md`、`support/runtime-support.md` | 已输出测试结果摘要 |
| ENV/泳道/环境测试 | `flows/env-precheck-flow.md` | `support/runtime-support.md`、`refs/data-reference.md` | 已产出 `<ENV>`、`<CLUSTER>`、`<VDC>` |
| 失败结果诊断 | `flows/doctor-flow.md` | `refs/doctor-reference.md`、`support/runtime-support.md` | 已输出根因分类、证据摘要、置信度与下一步建议 |
| 查询命令模板 | `refs/cli-reference.md` | `support/runtime-support.md` | 已获得可执行命令模板 |
| 查询区域/环境规则 | `refs/data-reference.md` | 无 | 已推导 `<ZONE>`、`<JWT-VRegion>`、`<VDC>` 或环境命名规则 |
| 安装/鉴权/通用错误 | `support/runtime-support.md` | 无 | 工具可用或错误已按规则处理 |

## 目录职责

| 目录 | 职责 | 约束 |
|------|------|------|
| `flows/` | 必须执行的流程步骤 | 只写执行顺序、检查点和决策规则 |
| `refs/` | 查表型参考与命令模板 | 只写命令、映射、枚举、模板，不写主流程 |
| `support/` | 运行支撑与异常兜底 | 只写安装、更新、鉴权、站点切换、通用错误处理 |

## 强制规则

- ENV/泳道/环境测试必须先完成 `flows/env-precheck-flow.md`，未产出 `<CLUSTER>` 与 `<VDC>` 时禁止进入测试请求发送。
- `flows/test-flow.md` 负责接口选择、参数生成、发送前确认、结果总结；命令细节只引用 `refs/cli-reference.md`。
- `flows/doctor-flow.md` 负责诊断主流程；根因枚举、报告结构和模板只引用 `refs/doctor-reference.md`。
- CLI 安装、JWT、登录鉴权、权限错误统一查 `support/runtime-support.md`，禁止在其他文件重复维护。
