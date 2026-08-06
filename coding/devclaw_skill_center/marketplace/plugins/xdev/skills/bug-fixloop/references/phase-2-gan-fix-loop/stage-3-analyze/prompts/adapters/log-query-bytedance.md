# Stage 3 / Log Query / Adapter: bytedance

> PROFILE=bytedance-tce 的日志查询 adapter。**通过 bytedcli log + LogID + PSM 过滤** 拉取服务端日志。
>
> 由 `failure-analysis.md` 的「步骤 2.5: 通过 LogID 查询服务日志」段调用。

## 适用场景

PROFILE=bytedance-tce 时，failure-analysis.md 的 Judge 角色在分析失败用例时,如果失败用例的 `log_ids` 非空,通过本 adapter 拉服务端日志获取额外的调试上下文。

## 输入

- `LOG_IDS`：来自 failed_cases.jsonl 的 `log_ids` 数组（每个失败用例最多取 3 个 LogID，多个失败用例共享同一个 LogID 时只查一次）
- `PSM`：被测服务的 PSM
- `BYTEDCLI_SITE`：默认 `boe`
- `BYTEDCLI_SKILLS_DIR`：用于读 bytedance-log/SKILL.md 学命令

## 流程

### Step 1: Read bytedance-log SKILL

```bash
cat $BYTEDCLI_SKILLS_DIR/bytedance-log/SKILL.md
```

了解日志查询命令的具体用法。然后 `--help` 确认当前版本参数。

### Step 2: 对每个 LogID 执行查询

```bash
# 伪代码：bytedcli 命令名以实际 SKILL.md 为准
bytedcli --json --site $BYTEDCLI_SITE log query \
  --logid <LOGID> \
  --psm $PSM \
  > /tmp/logid_raw_<LOGID>.log
```

### Step 3: 轮询重试（日志可能延迟写入）

最多 10 次，渐进退避：
- 第 1 次失败 → 等 10 秒
- 第 2 次失败 → 等 15 秒
- 第 3 次失败 → 等 20 秒
- ...
- 第 10 次失败 → 放弃，仅记录"日志未及时写入"

总最大等待 ~4 分钟。

### Step 4: 过滤错误关键字

```bash
grep -iE "error|panic|fatal|fail|exception|nil pointer|timeout|refused|404|500|502|503" \
  /tmp/logid_raw_<LOGID>.log -C 3 > /tmp/logid_filtered_<LOGID>.log

if [ ! -s /tmp/logid_filtered_<LOGID>.log ]; then
  head -200 /tmp/logid_raw_<LOGID>.log > /tmp/logid_filtered_<LOGID>.log
fi
```

### Step 5: 把过滤后的日志返回给 Judge

Judge 在分析每个失败用例时，读取对应的 `/tmp/logid_filtered_<LOGID>.log`，作为额外证据：
- panic 堆栈
- error 级别日志
- 请求处理流程
- 非预期返回值

## 跨外部 skill 依赖

- `$BYTEDCLI_SKILLS_DIR/bytedance-log/SKILL.md`

## 处理规则

- 每个失败用例最多查询 3 个 LogID
- 多个失败用例共享同一个 LogID 时只查一次
- 没有 log_ids 的用例跳过
- 超时后继续分析，不阻塞整体流程
- 过滤后的日志按 LogID 索引存储

## 失败处理

- bytedcli log 命令报错（认证失败、网络问题）→ warn,继续分析（仅基于 stdout）
- 所有 LogID 都查不到 → warn,继续分析
- 部分 LogID 查到 → 继续，使用查到的部分
