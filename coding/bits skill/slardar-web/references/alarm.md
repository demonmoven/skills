# Alarm — 完整参考

## Contents

- 查询操作: list, detail, history, history-detail, noise, similar
- 写操作（需二次确认）: ack, un-ack, history-status, history-status-batch, history-add-comment
- 参考值: 报警分类、噪音级别、触发记录状态、报警级别

## 查询操作

### alarm list — 报警规则列表

查询 bid 下的报警规则列表，支持多种筛选条件。

```bash
slardar-web-cli alarm list --bid <BID>
```

**参数**：

| 参数             | 必须 | 默认值 | 说明                                             |
| ---------------- | ---- | ------ | ------------------------------------------------ |
| `--bid`          | 是   | -      | 业务 ID                                          |
| `--name`         | 否   | -      | 按名称搜索，逗号分隔多个                         |
| `--receiver`     | 否   | -      | 按接收人搜索，逗号分隔多个                       |
| `--level`        | 否   | -      | 按级别筛选: P0,P1,P2，逗号分隔                   |
| `--category`     | 否   | -      | 按分类筛选，逗号分隔（分类表见下方「报警分类」） |
| `--tag`          | 否   | -      | 按标签筛选，逗号分隔                             |
| `--ids`          | 否   | -      | 按报警 ID 筛选，逗号分隔                         |
| `--enabled-only` | 否   | false  | 仅显示启用的规则                                 |
| `--page-num`     | 否   | 1      | 页码                                             |
| `--page-size`    | 否   | 20     | 每页条数                                         |
| `--region`       | 否   | cn     | 区域                                             |
| `--raw`          | 否   | false  | 输出原始 JSON                                    |

**常用场景**：

```bash
# 查看所有报警
slardar-web-cli alarm list --bid <BID> --page-size 999

# 只看 P0 级别
slardar-web-cli alarm list --bid <BID> --level P0

# 只看启用的 JS 错误报警
slardar-web-cli alarm list --bid <BID> --category js_error --enabled-only

# 按接收人搜索
slardar-web-cli alarm list --bid <BID> --receiver zhangsan

# 按名称搜索
slardar-web-cli alarm list --bid <BID> --name "首屏"
```

输出包含：统计摘要（级别分布、分类分布、标签分布、启用/关闭数量）和规则列表表格（状态、级别、名称、分类、标签、策略摘要、接收人、近30天触发次数、链接）。

---

### alarm detail — 报警配置详情

获取报警的完整配置信息，包括策略规则、阈值条件、通知方式、接收人等。对于引用自定义指标（UUID）的策略，会自动调用 metric_management/get 逐个反查指标完整口径。

```bash
slardar-web-cli alarm detail --id <ALARM_ID> --bid <BID>
```

**参数**：

| 参数       | 必须 | 默认值 | 说明          |
| ---------- | ---- | ------ | ------------- |
| `--id`     | 是   | -      | 报警 ID       |
| `--bid`    | 是   | -      | 业务 ID       |
| `--region` | 否   | cn     | 区域          |
| `--raw`    | 否   | false  | 输出原始 JSON |

输出包含：报警基本信息（ID、名称、级别、状态、平台、排除空值、防尖刺）、标签列表、升级通知策略、生效时段、时间信息（创建/更新/执行时间）、报警策略详情（指标、阈值、时间窗口、过滤条件）、飞书群通知。

---

### alarm history — 触发历史列表

查询报警的触发记录，包含每次触发的指标值、阈值对比、确认状态等。汇总表优先展示实际触发的策略（而非第一条策略），多策略场景有单独详情展开。

```bash
slardar-web-cli alarm history --id <ALARM_ID> --bid <BID>
```

**参数**：

| 参数            | 必须 | 默认值   | 说明                                                                        |
| --------------- | ---- | -------- | --------------------------------------------------------------------------- |
| `--id`          | 是   | -        | 报警 ID                                                                     |
| `--bid`         | 是   | -        | 业务 ID                                                                     |
| `--start-time`  | 否   | 6小时前  | 起始时间（Unix 秒级时间戳），**必须**通过 `slardar-web-cli time parse` 生成 |
| `--end-time`    | 否   | 当前时间 | 结束时间（Unix 秒级时间戳），**必须**通过 `slardar-web-cli time parse` 生成 |
| `--page-num`    | 否   | 1        | 页码                                                                        |
| `--page-size`   | 否   | 10       | 每页条数                                                                    |
| `--has-comment` | 否   | false    | 仅显示有评论的记录                                                          |
| `--agg-id`      | 否   | -        | 按聚合 ID 过滤                                                              |
| `--region`      | 否   | cn       | 区域                                                                        |
| `--raw`         | 否   | false    | 输出原始 JSON                                                               |

**常用场景**：

```bash
# 查看最近 6 小时的触发历史（默认）
slardar-web-cli alarm history --id <ID> --bid <BID>

# 查看最近 24 小时（先通过 time parse 获取时间戳）
# slardar-web-cli time parse "最近24小时" → START_TIME / END_TIME
slardar-web-cli alarm history --id <ID> --bid <BID> \
  --start-time <START_TIME> --end-time <END_TIME>

# 查看最近 7 天，每页 20 条（先通过 time parse 获取时间戳）
# slardar-web-cli time parse "最近7天" → START_TIME / END_TIME
slardar-web-cli alarm history --id <ID> --bid <BID> \
  --start-time <START_TIME> --end-time <END_TIME> --page-size 20
```

输出包含：报警触发记录表（触发时间、状态、确认状态、评论标记、聚合 ID、触发策略、当前值、阈值、样本量、用户数、时间窗口、关联 Issue、跳转链接）、多策略详情（如有）、优化策略命中情况。

---

### alarm history-detail — 单条触发详情

根据 history_id 查询某条触发历史的完整详情，包括报警基本信息、策略结果（每条策略的触发状态和指标值）、关联 Issue、策略链接等。

```bash
slardar-web-cli alarm history-detail --history-id <HISTORY_ID> --bid <BID>
```

**参数**：

| 参数           | 必须 | 默认值 | 说明          |
| -------------- | ---- | ------ | ------------- |
| `--history-id` | 是   | -      | 触发记录 ID   |
| `--bid`        | 是   | -      | 业务 ID       |
| `--region`     | 否   | cn     | 区域          |
| `--raw`        | 否   | false  | 输出原始 JSON |

**常用场景**：

```bash
# 查看某条触发记录的完整详情
slardar-web-cli alarm history-detail --history-id 55898990 --bid slardar_web

# 输出原始 JSON（用于后续归因分析等）
slardar-web-cli alarm history-detail --history-id 55898990 --bid slardar_web --raw
```

输出包含：基本信息表（History ID、报警 ID、报警名称、级别、触发时间、记录状态、确认状态、评论、聚合 ID）、关联 Issue（如有）、策略结果详情（每条策略的触发状态、当前值、阈值、样本量、用户数、窗口、优化命中）、策略链接。

---

### alarm noise — 噪音检测

对单条报警触发记录进行噪音分析，判断该次触发是否为噪音，并给出噪音评分、检测器结果和阈值优化建议。

支持两种调用方式：

- 指定 `--history-id`：直接检测某条触发记录
- 指定 `--alarm-id`：自动获取最近 7 天内最新一条触发记录后检测

```bash
# 方式一：指定 history-id
slardar-web-cli alarm noise --history-id <HISTORY_ID> --bid <BID>

# 方式二：指定 alarm-id（自动取最新触发记录）
slardar-web-cli alarm noise --alarm-id <ALARM_ID> --bid <BID>
```

**参数**：

| 参数           | 必须   | 默认值 | 说明                              |
| -------------- | ------ | ------ | --------------------------------- |
| `--history-id` | 二选一 | -      | 报警触发历史记录 ID               |
| `--alarm-id`   | 二选一 | -      | 报警规则 ID（自动取最新触发记录） |
| `--bid`        | 是     | -      | 业务 ID                           |
| `--region`     | 否     | cn     | 区域                              |
| `--raw`        | 否     | false  | 输出原始 JSON                     |

**噪音级别说明**见下方「噪音级别」参考表。

输出包含：噪音评分与级别、疑似噪音标识、各检测器结果与证据、阈值优化建议（如有）、推荐修复操作（如有）、汇总证据链。

**常用场景**：

```bash
# 检测某条触发记录是否为噪音
slardar-web-cli alarm noise --history-id 54209982 --bid slardar_web

# 直接用报警 ID 检测最新一次触发
slardar-web-cli alarm noise --alarm-id 4112280 --bid slardar_web
```

---

### alarm similar — 相似报警检测

根据一条报警触发记录，在历史数据中查找相似的报警触发（可跨规则），帮助发现关联问题、重复报警或同类故障。

支持两种调用方式：

- 指定 `--history-id`：直接以该触发记录为基准查找相似
- 指定 `--alarm-id`：自动获取最近 7 天内最新一条触发记录后查找

```bash
# 方式一：指定 history-id
slardar-web-cli alarm similar --history-id <HISTORY_ID> --bid <BID>

# 方式二：指定 alarm-id（自动取最新触发记录）
slardar-web-cli alarm similar --alarm-id <ALARM_ID> --bid <BID>
```

**参数**：

| 参数            | 必须   | 默认值 | 说明                                    |
| --------------- | ------ | ------ | --------------------------------------- |
| `--history-id`  | 二选一 | -      | 报警触发历史记录 ID                     |
| `--alarm-id`    | 二选一 | -      | 报警规则 ID（自动取最新触发记录）       |
| `--bid`         | 是     | -      | 业务 ID                                 |
| `--time-window` | 否     | 1440   | 搜索时间窗口（小时），默认 1440（60天） |
| `--min-score`   | 否     | 0.7    | 最低相似度阈值 0~1                      |
| `--limit`       | 否     | 400    | 最大返回数量                            |
| `--region`      | 否     | cn     | 区域                                    |
| `--raw`         | 否     | false  | 输出原始 JSON                           |

输出包含：匹配统计、元数据信息、规则分布表（多规则时）、相似报警列表（相似度、规则名/展示标题、分类、触发时间、触发值、分组信息、平台/区域/环境（存在差异时动态展示）、详情链接）、相似度分解（指标匹配/过滤条件/触发值/样本量/用户数各维度得分）。

**常用场景**：

```bash
# 查找与某条触发记录相似的报警
slardar-web-cli alarm similar --history-id 54209472 --bid slardar_web

# 用报警 ID 自动取最新触发记录后查找
slardar-web-cli alarm similar --alarm-id 4129721 --bid slardar_web

# 只要高相似度的结果
slardar-web-cli alarm similar --alarm-id 4129721 --bid slardar_web --min-score 0.95

# 缩小时间窗口到最近 7 天
slardar-web-cli alarm similar --alarm-id 4129721 --bid slardar_web --time-window 168
```

---

## 写操作（需二次确认）

> 所有写操作均需用户二次确认，命令执行前会展示变更内容。使用 `--yes` 可跳过交互式确认。

### alarm ack — 屏蔽报警

屏蔽报警（静默一段时间不发送通知），支持指定屏蔽时长和按条件屏蔽。

```bash
slardar-web-cli alarm ack --id <ALARM_ID> --bid <BID> --duration <SECONDS>
```

**参数**：

| 参数                | 必须 | 默认值 | 说明                                                                      |
| ------------------- | ---- | ------ | ------------------------------------------------------------------------- |
| `--id`              | 是   | -      | 报警规则 ID                                                               |
| `--bid`             | 是   | -      | 业务 ID                                                                   |
| `--duration`        | 是   | -      | 屏蔽时长（秒），如 1800=30分钟, 3600=1小时                                |
| `--extra-condition` | 否   | -      | 按条件屏蔽，JSON 格式: `{"属性名": ["值1", "值2"]}`，仅屏蔽符合条件的触发 |
| `--region`          | 否   | cn     | 区域                                                                      |
| `--yes`             | 否   | false  | 跳过确认                                                                  |
| `--raw`             | 否   | false  | 输出原始 JSON                                                             |

**常用场景**：

```bash
# 屏蔽 30 分钟
slardar-web-cli alarm ack --id 12345 --bid slardar_web --duration 1800 --yes

# 屏蔽 1 小时
slardar-web-cli alarm ack --id 12345 --bid slardar_web --duration 3600 --yes

# 按条件屏蔽（仅屏蔽 os=android 的触发）
slardar-web-cli alarm ack --id 12345 --bid slardar_web --duration 3600 \
  --extra-condition '{"os": ["android"]}' --yes
```

---

### alarm un-ack — 取消屏蔽

取消报警屏蔽，恢复正常通知。

```bash
slardar-web-cli alarm un-ack --id <ALARM_ID> --bid <BID>
```

**参数**：

| 参数       | 必须 | 默认值 | 说明                                                            |
| ---------- | ---- | ------ | --------------------------------------------------------------- |
| `--id`     | 是   | -      | 报警规则 ID                                                     |
| `--bid`    | 是   | -      | 业务 ID                                                         |
| `--ack-id` | 否   | -      | 指定要取消的屏蔽记录 ID（不传则自动获取活跃屏蔽记录并全部取消） |
| `--region` | 否   | cn     | 区域                                                            |
| `--yes`    | 否   | false  | 跳过确认                                                        |
| `--raw`    | 否   | false  | 输出原始 JSON                                                   |

**常用场景**：

```bash
# 取消屏蔽（自动获取活跃屏蔽记录）
slardar-web-cli alarm un-ack --id 12345 --bid slardar_web --yes

# 取消指定屏蔽记录
slardar-web-cli alarm un-ack --id 12345 --bid slardar_web --ack-id 67890 --yes
```

---

### alarm history-status — 修改单条触发记录状态

修改某条报警触发记录的状态标记，可用于将误报标记为"无效"、将周期性抖动标记为"正常波动"、将正在处理的问题标记为"处理中"等。

> `history-id` 需先通过 `alarm history --id <ALARM_ID> --bid <BID>` 获取。

```bash
slardar-web-cli alarm history-status --history-id <HISTORY_ID> --bid <BID> --status <0-4>
```

**参数**：

| 参数           | 必须 | 默认值 | 说明                                                     |
| -------------- | ---- | ------ | -------------------------------------------------------- |
| `--history-id` | 是   | -      | 触发记录 ID                                              |
| `--bid`        | 是   | -      | 业务 ID                                                  |
| `--status`     | 是   | -      | 目标状态：0=有效 / 1=无效 / 2=正常波动 / 3=处理中 / 4=未设置 |
| `--comment`    | 否   | -      | 备注说明，附加在状态变更上                               |
| `--yes`        | 否   | false  | 跳过确认                                                 |
| `--region`     | 否   | cn     | 区域                                                     |
| `--raw`        | 否   | false  | 输出原始 JSON                                            |

**常用场景**：

```bash
# 标记为"无效"（误报）
slardar-web-cli alarm history-status --history-id 54209982 --bid slardar_web --status 1 --yes

# 标记为"正常波动"
slardar-web-cli alarm history-status --history-id 54209982 --bid slardar_web --status 2 --yes

# 标记为"处理中"
slardar-web-cli alarm history-status --history-id 54209982 --bid slardar_web --status 3 --yes

# 附带备注
slardar-web-cli alarm history-status --history-id 54209982 --bid slardar_web \
  --status 1 --comment "周期性抖动，已确认非业务问题" --yes
```

状态值完整含义见下方「触发记录状态值」参考表。

---

### alarm history-status-batch — 批量修改触发记录状态

批量修改多条触发记录的状态，适用于集中清理误报或同类型抖动。

> 批量接口**仅支持 `--status 0 / 1 / 2`**，不支持 3（处理中）和 4（未设置）。

```bash
slardar-web-cli alarm history-status-batch --ids <ID_LIST> --bid <BID> --status <0|1|2>
```

**参数**：

| 参数       | 必须 | 默认值 | 说明                                          |
| ---------- | ---- | ------ | --------------------------------------------- |
| `--ids`    | 是   | -      | 触发记录 ID 列表，逗号分隔（如 `111,222,333`） |
| `--bid`    | 是   | -      | 业务 ID                                       |
| `--status` | 是   | -      | 目标状态：0=有效 / 1=无效 / 2=正常波动        |
| `--yes`    | 否   | false  | 跳过确认                                      |
| `--region` | 否   | cn     | 区域                                          |
| `--raw`    | 否   | false  | 输出原始 JSON                                 |

**常用场景**：

```bash
# 批量标记为"无效"
slardar-web-cli alarm history-status-batch \
  --ids 54209982,54209983,54209984 --bid slardar_web --status 1 --yes

# 批量标记为"正常波动"
slardar-web-cli alarm history-status-batch \
  --ids 54209982,54209983 --bid slardar_web --status 2 --yes
```

---

### alarm history-add-comment — 为触发记录添加评论

为某条触发记录追加评论备注，常用于记录人工处置过程、定位结论或与相关同学同步上下文。

> `history-id` 需先通过 `alarm history --id <ALARM_ID> --bid <BID>` 获取。

```bash
slardar-web-cli alarm history-add-comment --history-id <HISTORY_ID> --bid <BID> --content "<评论内容>"
```

**参数**：

| 参数           | 必须 | 默认值 | 说明          |
| -------------- | ---- | ------ | ------------- |
| `--history-id` | 是   | -      | 触发记录 ID   |
| `--bid`        | 是   | -      | 业务 ID       |
| `--content`    | 是   | -      | 评论内容      |
| `--yes`        | 否   | false  | 跳过确认      |
| `--region`     | 否   | cn     | 区域          |
| `--raw`        | 否   | false  | 输出原始 JSON |

**常用场景**：

```bash
# 添加处理备注
slardar-web-cli alarm history-add-comment \
  --history-id 55855649 --bid slardar_web --content "已定位为 CDN 抖动，无需处理" --yes

# 配合 history-status 使用，先标记状态再补充备注
slardar-web-cli alarm history-status --history-id 55855649 --bid slardar_web --status 1 --yes
slardar-web-cli alarm history-add-comment \
  --history-id 55855649 --bid slardar_web --content "自动打标为无效" --yes
```

---

## 参考值

### 报警分类（category）

| key            | 说明             |
| -------------- | ---------------- |
| pageview       | 用户分析         |
| action         | 行为指标         |
| performance    | 性能指标         |
| js_error       | Js错误指标       |
| resource       | 静态资源指标     |
| resource_error | 静态资源错误指标 |
| blank_screen   | 白屏指标         |
| http           | 请求指标         |
| http_error     | 请求错误指标     |
| custom         | 事件指标         |
| operation      | 经营数据         |
| image          | 图片资源         |
| composite      | 复合指标         |
| log            | 自定义日志       |

### 噪音级别

| 级别      | 含义                         |
| --------- | ---------------------------- |
| none      | 无噪音，报警正常             |
| suspected | 疑似噪音，建议关注           |
| confirmed | 确认噪音，建议调整阈值或规则 |

### 触发记录状态值

| 值  | 含义     | 单条支持 | 批量支持 |
| --- | -------- | -------- | -------- |
| 0   | 有效     | 是       | 是       |
| 1   | 无效     | 是       | 是       |
| 2   | 正常波动 | 是       | 是       |
| 3   | 处理中   | 是       | 否       |
| 4   | 未设置   | 是       | 否       |

### 报警级别

| 级别 | 说明       |
| ---- | ---------- |
| P0   | 最高优先级 |
| P1   | 中优先级   |
| P2   | 低优先级   |
