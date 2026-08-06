# 场景模板映射索引 (Scene Template Mapping Index)

> **维护说明**：本文件由人工维护，用于定义"用户意图 → 平台模板"的映射关系。
> 当 Skill 识别到用户需求匹配某个场景时，直接使用对应的 `template_id` 通过
> `CreatePipelineFromTemplateSync` RPC 创建流水线，无需从零编排 DSL。
>
> **新增场景**时，请按下方格式追加条目，并确保 `template_id` 和 `space_id` 已在 Bits 平台上创建。

---

## 格式说明

每个场景条目包含以下字段：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `scene` | String | ✅ | 场景名称，简短描述该模板覆盖的典型用途 |
| `template_id` | UInt64 | ✅ | Bits 平台上的模板 ID |
| `space_id` | UInt64 | ✅ | 模板所属的空间 ID |
| `keywords` | List[String] | ✅ | 匹配关键词列表，用于从用户自然语言意图中识别场景 |
| `description` | String | ✅ | 场景的详细描述，帮助模型理解何时应该匹配该模板 |
| `required_vars` | Table | ⚠️ | 必须由用户提供或模型推断的模板变量（见下方表格格式） |
| `optional_vars` | Table | ❌ | 可选模板变量，有合理默认值 |

---

## ⚠️ 变量值填充规则（强制）

**在调用 `CreatePipelineFromTemplateSync` 时，`template_vars` 中每个变量的 `value` 必须是从用户输入中提取的真实值，绝对不能使用本文档中 `description` 列的说明文字作为值。**

### ❌ 错误示范（把描述当作值传递）

```json
{
  "template_vars": [
    {"name": "custom.template.psm", "value": {"text": "服务 PSM，三段式形如 a.b.c"}},
    {"name": "custom.template.scm_name", "value": {"text": "PSM 对应的 SCM 仓库，形如 a/b/c"}}
  ]
}
```

### ✅ 正确示范（使用用户提供的真实值）

```json
{
  "template_vars": [
    {"name": "custom.template.psm", "value": {"text": "atom.demo.bits_development_tce"}},
    {"name": "custom.template.scm_name", "value": {"text": "bits/nexus_flow/api"}}
  ]
}
```

### 规则总结

1. `description` 列仅用于理解变量的含义，**绝不可作为变量值**
2. `example` 列展示了值的格式示例，真实值必须从**用户输入中提取**
3. 如果用户未提供某个 `required_vars` 的值，**必须追问用户**，不得自行编造或使用描述文字填充
4. 变量的 `name` 必须加 `custom.template.` 前缀（如 `custom.template.psm`）

---

## CLI 调用示例

匹配到场景模板后，通过以下 CLI 命令创建流水线：

```bash
# 示例：为 atom.demo.bits_development_tce 创建 TCE BOE 基准部署流水线
bits_pipeline_cli call \
  --env cn \
  --username zhangsan \
  --rpc CreatePipelineFromTemplateSync \
  --path-param template_id=1126124772610 \
  --body-json '{
    "pipeline_name": {"value": "TCE-BOE-prod-deploy", "lang": "zh", "texts": {"zh": "TCE-BOE-prod-deploy"}},
    "space_id": 1097554434818,
    "template_vars": [
      {"name": "custom.template.psm", "value": {"text": "atom.demo.bits_development_tce"}},
      {"name": "custom.template.scm_name", "value": {"text": "bits/nexus_flow"}}
    ]
  }'
```

**VarAssignEntry.value 类型映射**：

| 用户输入类型 | JSON 字段 | 示例 |
|-------------|----------|------|
| 字符串 | `"text": "value"` | `{"name": "custom.template.psm", "value": {"text": "atom.demo.bits_development_tce"}}` |
| 布尔值 | `"boolean": true/false` | `{"name": "custom.template.enable_lint", "value": {"boolean": true}}` |
| 数字 | `"number": 123` | `{"name": "custom.template.timeout", "value": {"number": 300}}` |
| JSON 数组 | `"json_array": "[...]"` | `{"name": "custom.template.psms", "value": {"json_array": "[\"psm1\",\"psm2\"]"}}` |
| JSON 对象 | `"json_object": "{...}"` | `{"name": "custom.template.extra", "value": {"json_object": "{\"k\":\"v\"}"}}` |

---

## 场景列表

### 1. CN - TCE BOE 基准（prod）部署模板

- **scene**: cn-tce-boe-prod-deploy
- **template_id**: `1126124772610`
- **space_id**: `1097554434818`
- **keywords**: `["TCE", "BOE", "基准", "prod", "部署模板", "CN"]`
- **description**: 适用于 CN - TCE BOE 基准（prod）部署场景。

**required_vars**:

| name | type | description | example |
|------|------|-------------|---------|
| `psm` | text | 服务 PSM，三段式格式 | `atom.demo.bits_development_tce` |
| `scm_name` | text | PSM 对应的 SCM 仓库路径 | `bits/nexus_flow` |

---

### 2. CN - TCE BOE 泳道部署模板

- **scene**: cn-tce-boe-swimlane-deploy
- **template_id**: `1127882138882`
- **space_id**: `1097554434818`
- **keywords**: `["TCE", "BOE", "泳道", "部署模板", "CN"]`
- **description**: 适用于 CN - TCE BOE 泳道部署场景。

**required_vars**:

| name | type | description | example |
|------|------|-------------|---------|
| `psm` | text | 服务 PSM，三段式格式 | `atom.demo.bits_development_tce` |
| `scm_name` | text | PSM 对应的 SCM 仓库路径 | `bits/nexus_flow` |

---

### 3. CN - TCE PPE 泳道部署模板

- **scene**: cn-tce-ppe-swimlane-deploy
- **template_id**: `1103878448642`
- **space_id**: `1097554434818`
- **keywords**: `["TCE", "PPE", "泳道", "部署模板", "CN"]`
- **description**: 适用于 CN - TCE PPE 泳道部署场景。

**required_vars**:

| name | type | description | example |
|------|------|-------------|---------|
| `psm` | text | 服务 PSM，三段式格式 | `atom.demo.bits_development_tce` |
| `scm_name` | text | PSM 对应的 SCM 仓库路径 | `bits/nexus_flow` |

---

### 4. CN - TCE 线上部署模板

- **scene**: cn-tce-online-deploy
- **template_id**: `1127983207938`
- **space_id**: `1097554434818`
- **keywords**: `["TCE", "线上", "部署模板", "CN"]`
- **description**: 适用于 CN - TCE 线上部署场景。

**required_vars**:

| name | type | description | example |
|------|------|-------------|---------|
| `psm` | text | 服务 PSM，三段式格式 | `atom.demo.bits_development_tce` |
| `scm_name` | text | PSM 对应的 SCM 仓库路径 | `bits/nexus_flow` |

---

### 5. CN - FaaS BOE 泳道部署模板

- **scene**: cn-faas-boe-swimlane-deploy
- **template_id**: `1126850312194`
- **space_id**: `1097554434818`
- **keywords**: `["FaaS", "BOE", "泳道", "部署模板", "CN"]`
- **description**: 适用于 CN - FaaS BOE 泳道部署场景。

**required_vars**:

| name | type | description | example |
|------|------|-------------|---------|
| `psm` | text | 服务 PSM，三段式格式 | `data.faas.my_function` |
| `scm_name` | text | PSM 对应的 SCM 仓库路径 | `faas/my_function` |

---

### 6. CN - FaaS PPE 泳道部署模板

- **scene**: cn-faas-ppe-swimlane-deploy
- **template_id**: `1126451919106`
- **space_id**: `1097554434818`
- **keywords**: `["FaaS", "PPE", "泳道", "部署模板", "CN"]`
- **description**: 适用于 CN - FaaS PPE 泳道部署场景。

**required_vars**:

| name | type | description | example |
|------|------|-------------|---------|
| `psm` | text | 服务 PSM，三段式格式 | `data.faas.my_function` |
| `scm_name` | text | PSM 对应的 SCM 仓库路径 | `faas/my_function` |

---

### 7. CN - FaaS 线上灰度部署模板

- **scene**: cn-faas-gray-deploy
- **template_id**: `1127882139650`
- **space_id**: `1097554434818`
- **keywords**: `["FaaS", "线上", "灰度", "部署模板", "CN"]`
- **description**: 适用于 CN - FaaS 线上灰度部署场景。

**required_vars**:

| name | type | description | example |
|------|------|-------------|---------|
| `psm` | text | 服务 PSM，三段式格式 | `data.faas.my_function` |
| `scm_name` | text | PSM 对应的 SCM 仓库路径 | `faas/my_function` |

---

### 8. CN - FaaS BOE 基准（prod）灰度部署模板

- **scene**: cn-faas-boe-prod-gray-deploy
- **template_id**: `1125973181442`
- **space_id**: `1097554434818`
- **keywords**: `["FaaS", "BOE", "基准", "prod", "灰度", "部署模板", "CN"]`
- **description**: 适用于 CN - FaaS BOE 基准（prod）灰度部署场景。

**required_vars**:

| name | type | description | example |
|------|------|-------------|---------|
| `psm` | text | 服务 PSM，三段式格式 | `data.faas.my_function` |
| `scm_name` | text | PSM 对应的 SCM 仓库路径 | `faas/my_function` |

---

<!-- 
  === 添加新场景模板 ===
  
  复制以下模板并填写对应信息：

  ### N. 场景名称

  - **scene**: scene-slug
  - **template_id**: `_TODO_`
  - **space_id**: `_TODO_`
  - **keywords**: `["关键词1", "关键词2"]`
  - **description**: 场景描述。

  **required_vars**:

  | name | type | description | example |
  |------|------|-------------|---------|
  | `var_name` | text | 变量说明 | `示例值` |

  **optional_vars**:

  | name | type | description | default | example |
  |------|------|-------------|---------|---------|
  | `var_name` | text | 变量说明 | 默认值 | `示例值` |

  ---
-->
