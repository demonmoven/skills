# 安全漏洞审查模式

安全问题是**最高优先级**的审查项。

---

## 1. SQL 注入

```go
// 严重问题：SQL 注入
query := "SELECT * FROM users WHERE id = " + userInput
db.Query(query)

// 正确做法：参数化查询
db.Query("SELECT * FROM users WHERE id = ?", userInput)
```

---

## 2. 命令注入

```go
// 严重问题：命令注入
cmd := exec.Command("sh", "-c", "echo " + userInput)

// 正确做法：避免 shell，直接传参
cmd := exec.Command("echo", userInput)
```

---

## 3. 路径遍历

```go
// 严重问题：路径遍历
path := "/data/" + userInput  // 如果 userInput 是 "../etc/passwd"

// 正确做法：验证路径
path := filepath.Join("/data/", filepath.Clean(userInput))
if !strings.HasPrefix(path, "/data/") {
    return errors.New("invalid path")
}
```

---

## 4. 敏感信息泄露

检查点：
- 日志中打印密码、token
- 错误消息中暴露内部细节
- panic 信息中包含敏感数据

---

## 审查判定

- 安全漏洞（注入、泄露）永远是 **严重 (critical)** 级别
- 如果代码使用了 "Fail Fast" 或 "Graceful Degradation" 策略（如 fallback 到本地验证），**不应**视为 Bug，应标记为 **提示 (Notice)** 或 **Pass**
