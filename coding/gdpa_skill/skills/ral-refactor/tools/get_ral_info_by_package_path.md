# Get RAL Info by Package Path

根据 package path 获取 RPC 服务的 PSM 信息。

## Input

- `package_path`: Go package 路径，例如 `code.byted.org/overpass/iesarch_devflow_api/rpc/iesarch_devflow_api`

## Output

- `psm`: 服务的 PSM 标识，格式为 `{product}.{subsystem}.{module}`

## Logic

### Step 1: 判断 Package 类型

检查 `package_path` 是否包含 `overpass`：
- 如果包含 `overpass`，则是 Overpass RPC Stub，继续执行 Step 2
- 如果不包含，返回错误提示：非 Overpass package，无法自动获取 PSM

### Step 2: 构造目标文件路径

从 package_path 推导出 `overpass_default.go` 文件路径：

```
package_path: code.byted.org/overpass/p_s_m/rpc/p_s_m
                              ↓
target_file:  code.byted.org/overpass/p_s_m/rpc/p_s_m/overpass_default.go
```

### Step 3: 使用 gopls 定位文件

使用 gopls-lsp 工具跳转到 overpass package 中的目标文件：

```bash
# 方法 1: 通过 go list 获取 package 本地路径
go list -f '{{.Dir}}' {package_path}

# 方法 2: 在 go.mod 缓存中查找
# 路径通常为: $GOPATH/pkg/mod/code.byted.org/overpass/p_s_m@version/rpc/p_s_m/overpass_default.go
```

### Step 4: 读取 BASIC_PSM

在 `overpass_default.go` 文件中查找 `BASIC_PSM` 常量：

```go
// 文件: overpass_default.go
// 查找类似以下的定义：
const BASIC_PSM = "product.subsystem.module"
// 或
var BASIC_PSM = "product.subsystem.module"
```

使用正则匹配提取 PSM 值：
```
pattern: (const|var)\s+BASIC_PSM\s*=\s*"([^"]+)"
group 2 即为 PSM 值
```

### Step 5: 返回结果

返回提取到的 PSM 值。

## Example

### Input
```
package_path: code.byted.org/overpass/iesarch_devflow_api/rpc/iesarch_devflow_api
```

### Execution
```bash
# 获取 package 本地路径
$ go list -f '{{.Dir}}' code.byted.org/overpass/iesarch_devflow_api/rpc/iesarch_devflow_api
/Users/xxx/go/pkg/mod/code.byted.org/overpass/iesarch_devflow_api@v1.2.3/rpc/iesarch_devflow_api

# 读取 overpass_default.go 文件
# 查找 BASIC_PSM 定义
const BASIC_PSM = "iesarch.devflow.api"
```

### Output
```
psm: iesarch.devflow.api
```

## Error Handling

| 错误场景 | 处理方式 |
|---------|---------|
| package_path 不包含 `overpass` | 返回错误：非 Overpass package，请手动指定 PSM |
| go list 执行失败 | 尝试使用 `go get` 下载 package 后重试 |
| 找不到 overpass_default.go | 返回错误：无法找到 overpass_default.go 文件 |
| 找不到 BASIC_PSM 定义 | 返回错误：无法在文件中找到 BASIC_PSM 定义 |

## Usage in Refactoring

在 RAL 重构过程中，使用此方法自动获取目标服务的 PSM：

```go
// 原代码
import "code.byted.org/overpass/iesarch_devflow_api/rpc/iesarch_devflow_api"

// 使用 get_ral_info_by_package_path 获取 PSM
// package_path = "code.byted.org/overpass/iesarch_devflow_api/rpc/iesarch_devflow_api"
// 输出: psm = "iesarch.devflow.api"

// 用于生成 RAL 配置
# conf/ral/services/rpc.yaml
iesarch_devflow_api:
  PSM: iesarch.devflow.api  # <- 从 BASIC_PSM 获取
  Protocol: rpc
```
