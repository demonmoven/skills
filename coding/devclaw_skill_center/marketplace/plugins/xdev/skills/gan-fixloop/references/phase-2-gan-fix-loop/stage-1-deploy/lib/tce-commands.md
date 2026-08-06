# TCE 命令参考

## 命令发现

bytedcli TCE 相关命令的完整用法，请参考已安装的 bytedcli skill 文件：

- **调用方式和全局参数**（`-j`、`--site` 等）：读取 `$BYTEDCLI_SKILLS_DIR/bytedance-tools/references/invocation.md`
- **TCE 子命令详情**：读取 `$BYTEDCLI_SKILLS_DIR/bytedance-tce/SKILL.md`

执行任何命令前，先通过 `--help` 确认当前版本的参数：

```bash
bytedcli tce --help
bytedcli tce <subcommand> --help
```

## gan-fixloop 常用 TCE 操作

| 操作 | 用途 |
|------|------|
| 实例查询 | 查询指定 PSM + 泳道下的实例列表，判断泳道状态 |
| 泳道部署 | 部署服务到指定泳道（create 或 upgrade） |
| 环境级联 | 查看 PSM 在不同环境/分区下的级联关系 |
| 服务搜索 | 按关键字搜索 TCE 上的服务，确认 PSM 是否存在 |
