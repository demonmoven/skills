# 筛选操作符参考

Metric query 的 `--filter` 和 Log query 的 `--filter` 共享以下操作符。

## filter 格式

JSON 数组，每个元素包含 `op`、`filter_name`、`values` 三个字段，多个条件之间是 **AND** 关系：

```json
[
  {"op": "<操作符>", "filter_name": "<维度名>", "values": ["<值1>", "<值2>"]},
  {"op": "<操作符>", "filter_name": "<维度名>", "values": ["<值>"]}
]
```

## 操作符列表

| 操作符 | op 值 | 说明 | 示例 |
|--------|-------|------|------|
| `=` | `eq` | 等于 | `{"op": "eq", "values": ["Chrome"]}` |
| `!=` | `neq` | 不等于 | `{"op": "neq", "values": ["IE"]}` |
| `in` | `in` | 包含在列表中 | `{"op": "in", "values": ["Chrome", "Firefox"]}` |
| `not in` | `not_in` | 不在列表中 | `{"op": "not_in", "values": ["IE", "Edge"]}` |
| `>` | `gt` | 大于 | `{"op": "gt", "values": ["1000"]}` |
| `>=` | `gte` | 大于等于 | `{"op": "gte", "values": ["500"]}` |
| `<` | `lt` | 小于 | `{"op": "lt", "values": ["5000"]}` |
| `<=` | `lte` | 小于等于 | `{"op": "lte", "values": ["3000"]}` |
| `regex` | `regex` | 正则匹配 | `{"op": "regex", "values": ["^/api/.*"]}` |
| `not_regex` | `not_regex` | 正则不匹配 | `{"op": "not_regex", "values": ["^/test/.*"]}` |
| `like` | `lk` | 模糊匹配（支持 `%` 通配符） | `{"op": "lk", "values": ["%login%"]}` |
| `not like` | `nlk` | 不模糊匹配 | `{"op": "nlk", "values": ["%debug%"]}` |
| `has` | `has` | 包含 | `{"op": "has", "values": ["error"]}` |
| `null` | `null` | 为空 | `{"op": "null", "values": []}` |
| `not_null` | `not_null` | 不为空 | `{"op": "not_null", "values": []}` |
| `is_empty` | `is_empty` | 为空字符串 | `{"op": "is_empty", "values": []}` |
| `includes` | `includes` | 包含（子串匹配或数组元素匹配） | `{"op": "includes", "values": ["/flex/query"]}` |
| `not_includes` | `not_includes` | 不包含（子串或数组元素不匹配） | `{"op": "not_includes", "values": ["internal"]}` |
| `starts_with` | `starts_with` | 以指定前缀开头 | `{"op": "starts_with", "values": ["/api/"]}` |
| `ends_with` | `ends_with` | 以指定后缀结尾 | `{"op": "ends_with", "values": [".html"]}` |

## 使用示例

### 示例 1：按浏览器和操作系统筛选

筛选 Chrome 浏览器 **且** Mac 系统的数据：

```json
[
  {"op": "in", "filter_name": "browser_brand", "values": ["Chrome"]},
  {"op": "eq", "filter_name": "os", "values": ["Mac"]}
]
```

### 示例 2：排除特定页面路径

排除以 `/test/` 开头的页面：

```json
[
  {"op": "not_regex", "filter_name": "page_path", "values": ["^/test/.*"]}
]
```

### 示例 3：筛选非空用户且指定地域

user_id 不为空 **且** 地域为北京或上海：

```json
[
  {"op": "not_null", "filter_name": "user_id", "values": []},
  {"op": "in", "filter_name": "region", "values": ["北京", "上海"]}
]
```

### 示例 4：模糊匹配 URL

筛选 URL 中包含 `login` 的请求：

```json
[
  {"op": "lk", "filter_name": "url", "values": ["%login%"]}
]
```
