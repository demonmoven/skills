# Go 常见陷阱

---

## 1. 循环变量捕获

```go
// 严重问题：所有 goroutine 共享同一变量
for _, v := range items {
    go func() {
        fmt.Println(v)  // v 会是最后一个值！
    }()
}

// 解决方案 1：传参
for _, v := range items {
    go func(val string) { fmt.Println(val) }(v)
}

// 解决方案 2：创建局部变量 (Go 1.22+ 已自动修复)
for _, v := range items {
    v := v
    go func() { fmt.Println(v) }()
}
```

---

## 2. 切片共享底层数组

```go
// 问题：对切片的修改可能影响原数组
original := []int{1, 2, 3, 4, 5}
slice := original[1:3]  // [2, 3]
slice[0] = 100          // original 变成 [1, 100, 3, 4, 5]

// 如果需要独立副本
slice := make([]int, 2)
copy(slice, original[1:3])
```

---

## 3. 接口与 nil

```go
// 问题：接口值不为 nil，但底层值为 nil
var p *MyType = nil
var i interface{} = p
fmt.Println(i == nil)  // false！因为接口有类型信息

// 正确检查
if i == nil || reflect.ValueOf(i).IsNil() {
    // 真正为空
}
```

---

## 4. append 陷阱

```go
// 问题：append 可能返回新切片
a := []int{1, 2, 3}
b := a[:2]
b = append(b, 4)  // 可能修改 a[2]，也可能不会，取决于容量

// 安全做法：如果需要独立切片
b := make([]int, 2)
copy(b, a[:2])
b = append(b, 4)
```

---

## 5. defer 参数求值时机

```go
// 问题：defer 参数在 defer 语句执行时求值
func example() {
    i := 0
    defer fmt.Println(i)  // 打印 0，不是 1
    i++
}

// 如果需要延迟求值
defer func() { fmt.Println(i) }()
```

---

## 6. 字符串遍历

```go
// 注意：range 字符串遍历的是 rune，不是 byte
s := "你好"
for i, r := range s {
    // i 是字节索引，r 是 rune
    // i: 0, 3 （不是 0, 1）
}
```
