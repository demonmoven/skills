# IDL 检测报告

## 概览

| 项目 | 值 |
|-----|-----|
| 项目名称 | {project} |
| PSM | {psm} |
| 服务类型 | {http/rpc} |
| IDL 类型 | {thrift/proto} |
| 检测范围 | {idl_dir} |
| 检测时间 | {check_time} |
| 变更范围 | {commit_range} |

### 检测结果统计

| 检测阶段 | Errors | Warnings | Info |
|---------|--------|----------|------|
| 规范检测 (P1) | {p1_errors} | {p1_warnings} | {p1_info} |
| 本地一致性 (P2) | {p2_errors} | {p2_warnings} | - |
| **合计** | **{total_errors}** | **{total_warnings}** | **{total_info}** |

**阻断问题**: {blocker_count} 个（必须修复后才能继续）

---

## 一、项目上下文

| 配置项 | 值 |
|-------|-----|
| IDL 类型 | {idl_type} |
| 服务类型 | {service_type} |
| 生成命令 | {gen_command} |
| IDL 目录 | {idl_dir} |
| 生成目录 | {gen_dir} |
| Service IDL | {service_idl} |
| Handler 目录 | {handler_dir} |

---

## 二、IDL 变更 Diff

### Service 变更

| 变更类型 | 方法名 | 参数类型 | 返回类型 | 路由注解 |
|---------|-------|---------|---------|---------|
| + 新增 | {MethodName} | {RequestType} | {ResponseType} | {route} |
| ~ 修改 | {MethodName} | {change_desc} | - | {route} |

### Struct 变更

| Struct 名称 | 字段编号 | 变更类型 | 变更详情 |
|------------|---------|---------|---------|
| {StructName} | {field_id} | + 新增 | {field_type} {field_name} |
| {StructName} | {field_id} | ~ 类型变更 | {old_type} → {new_type} ⚠️ 不兼容 |

### Enum 变更

| Enum 名称 | 变更类型 | 枚举项 | 值 |
|----------|---------|-------|-----|
| {EnumName} | + 新增 | {name} | {value} |

### 注解变更

| Struct.字段 | 旧注解 | 新注解 |
|------------|-------|-------|
| {StructName}.{FieldName} | {old_annotation} | {new_annotation} |

### 兼容性风险

| 风险级别 | 问题描述 | 影响分析 |
|---------|---------|---------|
| ⚠️ 高风险 | {desc} | {impact} |

### 变更文件

```
{changed_files}
```

---

## 三、规范检测结果 (P1)

### 开源规范 (Layer 1)

| 状态 | 规则 ID | 检测项 | 文件:行号 | 描述 | 修复建议 |
|-----|--------|-------|----------|------|---------|
| ✅ | OS-T001 | 字段编号重复检查 | - | 通过 | - |
| ❌ | OS-T005 | Service 方法参数数量 | {file}:{line} | {desc} | {fix_suggestion} |

### 字节体系规范 (Layer 2)

| 状态 | 规则 ID | 检测项 | 文件:行号 | 描述 | 修复建议 |
|-----|--------|-------|----------|------|---------|
| ❌ | BD-T001 | Response 缺少 BaseResp | {file}:{line} | {desc} | 添加 `255: optional base.BaseResp BaseResp` |
| ❌ | BD-T006 | HTTP Method 缺少路由注解 | {file}:{line} | {desc} | 添加路由注解 |
| ⚠️ | BD-T003 | Request 缺少 Base 字段 | {file}:{line} | {desc} | 添加 `255: optional base.Base Base` |

### 业务规范 (Layer 3)

| 状态 | 规则 ID | 检测项 | 文件:行号 | 描述 |
|-----|--------|-------|----------|------|
| ⚠️ | BZ-001 | 方法命名不规范 | {file}:{line} | {desc} |
| ⚠️ | BZ-003 | Enum 缺少 Undefined = 0 | {file}:{line} | {desc} |
| ℹ️ | BZ-013 | 孤立 struct | {file}:{line} | {desc} |

### 兼容性检测

| 状态 | 问题类型 | 位置 | 描述 | 影响 |
|-----|---------|------|------|------|
| ❌ | 字段类型变更 | {location} | {desc} | {impact} |
| ⚠️ | 字段删除未保留编号 | {location} | {desc} | 建议添加 reserved |

**规范检测统计**: {error_count} Errors, {warn_count} Warnings, {info_count} Info

---

## 四、本地一致性检测结果 (P2)

### P2.1 IDL ↔ 生成代码一致性

| 状态 | IDL 文件 | 生成目录 | Diff 摘要 | 修复建议 |
|-----|---------|---------|----------|---------|
| ❌ | {idl_file} | {gen_dir} | {diff_summary} | 执行 `{gen_command}` |

### P2.2 IDL 接口 ↔ 业务代码一致性

| 状态 | IDL 方法 | Route 状态 | Handler 状态 | 修复建议 |
|-----|---------|-----------|-------------|---------|
| ❌ | {method_name} | 未注册 | 未实现 | 补充路由注册和 Handler |
| ⚠️ | - | {route} (多余) | - | 确认是否为废弃接口 |

### P2.3 Git 变更一致性

| 状态 | 规则 | 变更文件 | 检测结果 | 修复建议 |
|-----|------|---------|---------|---------|
| ❌ | G1: IDL 变更但生成代码未更新 | {idl_files} | {gen_status} | 执行代码生成命令 |

**本地一致性统计**: {error_count} Errors, {warn_count} Warnings

---

## 五、问题分析与修复建议 (P3-P4)

### 问题 {n}: {issue_title}

**优先级**: {priority}（{P0 阻断性 / P1 严重 / P2 一般}）

#### 问题描述

{issue_desc}

#### 影响范围

```
├─ Service 方法: {method}
├─ HTTP 路由: {route}
├─ Handler: {handler}
├─ 下游调用方:
│   ├─ {caller_1}
│   └─ {caller_2}
└─ 兼容性: {compatibility}
```

#### 修复方案

| 方案 | 描述 | 优点 | 缺点 |
|-----|------|------|------|
| A (推荐) | {option_a} | {pro_a} | {con_a} |
| B | {option_b} | {pro_b} | {con_b} |

**关联问题**: {related_issues}

**自动修复**: ✅ 可自动修复 / ❌ 需人工确认

```
{fix_content}
```

---

## 六、检测总结

### 待处理事项

| 优先级 | 事项 | 状态 |
|-------|------|------|
| 🔴 必须修复 | {must_fix_1} | □ |
| 🔴 必须修复 | {must_fix_2} | □ |
| 🟡 建议修复 | {should_fix_1} | □ |
| 🟢 可选修复 | {optional_fix} | □ |

### 后续步骤

1. 修复所有阻断性问题（P0）
2. 重新执行检测验证修复结果
3. 处理建议修复项（可选）
4. 提交代码或创建 MR
