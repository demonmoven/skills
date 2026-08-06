# 字节跳动 Go 编码规范审查要点

---

## 1. 命名与风格

### 包（Package）
- 包名必须是小写的单个名词，且与目录名一致
- **禁止**使用下划线或大写字母
```go
// 正确
package user
package prompt

// 错误
package User       // 大写
package user_info  // 下划线
```

### 文件名
- 确认文件名遵循 `lower_snake_case.go` 格式
```
user_handler.go  ✅
userHandler.go   ❌
```

### 类型、函数、变量、常量
- 使用驼峰命名法（`camelCase` 或 `PascalCase`）
```go
// 正确
type UserInfo struct{}
func GetUserByID() {}
var userCount int

// 错误
type user_info struct{}
func get_user() {}
```

### 接收器（Receiver）
- 接收器命名应简短（通常是类型名的 1-2 个字母缩写），且在整个类型的方法中保持一致
```go
// 正确
func (u *User) GetName() string {}
func (u *User) SetName(name string) {}

// 错误：不一致
func (u *User) GetName() string {}
func (user *User) SetName(name string) {}
```

### 缩略词
- 通用缩略词（如 `ID`, `URL`, `API`, `JSON`）在命名中保持全大写
```go
userID    string  // 正确
userId    string  // 错误
apiURL    string  // 正确
apiUrl    string  // 错误
```

---

## 2. 包导入

### 分组与排序
- `import` 语句按照 **标准库、第三方库、内部库** 的顺序分组，每组内按字典序排序
```go
import (
    // 标准库
    "context"
    "fmt"
    // 第三方库
    "github.com/gin-gonic/gin"
    // 内部库
    "code.byted.org/project/internal/user"
)
```

### 别名与点导入
- **禁止**随意使用导入别名，除非为解决命名冲突
- **严格禁止**使用 `.` 进行点导入

---

## 3. 注释

- 所有导出的符号（函数、类型、变量、常量）都**必须**有完整的注释，注释应以符号名开头
- 在能清晰准确表达的前提下，**优先使用英文**注释
- 复杂逻辑应有注释说明"为什么"，而非仅"做了什么"

---

## 4. 函数与参数

- `context.Context` 如果存在，**必须**是第一个参数
- 函数入参**建议不超过 5 个**，超过应使用结构体封装
- 优先通过**值传递**而非指针传递（除非性能瓶颈或需修改状态）
- 包含 `sync.Mutex` 等同步原语的结构体**必须**使用指针接收器
- **不要**为 `map`、`func` 或 `chan` 类型使用指针接收器

---

## 5. 控制结构

- `if` 分支以 `return` 结尾时，应移除多余的 `else` 块
- 直接使用 `if condition`，而非 `if condition == true`
- 嵌套**建议不超过 3 层**，优先使用卫语句降低复杂度
- 保持正常代码路径最小缩进，优先处理错误并提前返回

---

## 6. Panic 与 Recover

- `panic` **仅限用于**程序启动阶段的致命错误或"绝不应该发生"的逻辑 bug
- **严禁**在正常业务逻辑中使用 `panic`
- `recover()` **必须**在 `defer` 函数中直接调用
- 每个 goroutine 入口点都应有 `recover` 机制

---

## 7. 错误处理

- **必须**显式检查并处理每一个返回的 `error`，**严禁**使用 `_` 忽略错误
- 使用 `errors.New` 创建简单静态错误，需要上下文时使用 `fmt.Errorf` 配合 `%w` 包装错误
