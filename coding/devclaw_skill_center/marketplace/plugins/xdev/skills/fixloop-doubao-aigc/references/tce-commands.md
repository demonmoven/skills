# TCE 查询命令参考

fixloop 的**部署/升级走 `bytedcli env`**（见 `deploy-procedure.md`）。`bytedcli tce` 仅用于 pod 级观察和辅助查询。

## 用到的 TCE 子命令

| 用途 | 命令示例 |
|------|---------|
| 列出指定 lane 下的 pod（判断 Running / 拿 pod name / 看 IDC 分布） | `bytedcli --json tce list-instance --psm <psm> --env <lane> --tce-site prod --page-size 20` |
| 环境级联（按 PSM 查 partition / env / lane_list） | `bytedcli --json tce env-cascader --psm <psm>` |
| 拿 cluster-id（env service upgrade-tce 需要） | `bytedcli --json tce list-service-clusters --service-id <sid> --tce-site prod` |
| Pod webshell（排查启动错误、查日志） | `bytedcli tce webshell open --psm <psm> --env <lane> --first` + `tce webshell exec --session-id <sid> --command '<cmd>'` |

## 关键点

- TCE 命令用 `--tce-site prod`（或 `--site prod`，等价）；BOE 调试用 `--tce-site boe`
- `list-instance` 返回 `data.pods[]`，不是 `items[]` 或 `instances[]` —— 看 JSON 再写解析
- `env-cascader` 的响应是 `data.data[].env_list[].lane_list[]` 三层嵌套（`data` 出现两次，外层是 envelope，内层是 partition 列表）

完整 TCE 子命令清单见 `$BYTEDCLI_SKILLS_DIR/bytedance-tce/SKILL.md`，执行前 `--help` 确认当前版本参数。

**禁用**：`bytedcli tce deploy-lane` —— 它不注册环境平台，会让染色请求被 BFF 拒绝。部署/升级一律走 `bytedcli env service deploy-tce` / `upgrade-tce`。
