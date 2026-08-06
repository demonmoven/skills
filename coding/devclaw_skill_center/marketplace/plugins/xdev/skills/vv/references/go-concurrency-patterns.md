# Go 并发安全审查模式

---

## 1. Map 并发读写

```go
// 严重问题：非 sync.Map 的并发访问会导致 panic
var m = make(map[string]int)

// goroutine 1
go func() { m["key"] = 1 }()   // 写
// goroutine 2
go func() { _ = m["key"] }()   // 读，并发读写会 panic

// 解决方案 1：使用 sync.Map
var m sync.Map

// 解决方案 2：使用互斥锁
var mu sync.RWMutex
mu.Lock()
m["key"] = 1
mu.Unlock()
```

---

## 2. Goroutine 泄漏

```go
// 问题：goroutine 永远阻塞
func process() {
    ch := make(chan int)
    go func() {
        result := doWork()
        ch <- result  // 如果没有接收者，永远阻塞
    }()
    // 忘记从 ch 接收
}

// 解决方案：使用带缓冲的 channel 或 context
func process(ctx context.Context) {
    ch := make(chan int, 1)  // 带缓冲
    go func() {
        select {
        case ch <- doWork():
        case <-ctx.Done():
            return
        }
    }()
}
```

---

## 3. Channel 问题

```go
// 问题 1：向已关闭的 channel 发送数据会 panic
close(ch)
ch <- value  // panic

// 问题 2：多次关闭同一 channel 会 panic
close(ch)
close(ch)  // panic

// 问题 3：从 nil channel 接收会永远阻塞
var ch chan int
<-ch  // 永远阻塞
```

---

## 4. 共享变量竞态

```go
// 问题：共享变量无锁访问
var counter int
for i := 0; i < 100; i++ {
    go func() { counter++ }()  // 竞态条件
}

// 解决方案 1：原子操作
var counter int64
atomic.AddInt64(&counter, 1)

// 解决方案 2：互斥锁
var mu sync.Mutex
mu.Lock()
counter++
mu.Unlock()
```

---

## 5. 锁使用问题

```go
// 问题 1：defer 解锁放在获取锁之前
defer mu.Unlock()  // 如果锁获取失败，会解锁未锁定的锁
mu.Lock()
// 正确做法：mu.Lock(); defer mu.Unlock()

// 问题 2：持有锁时调用阻塞操作
mu.Lock()
result := <-ch  // 可能永远阻塞，导致死锁
mu.Unlock()

// 问题 3：锁获取顺序不一致导致死锁
// goroutine 1: lockA -> lockB
// goroutine 2: lockB -> lockA  // 死锁风险
```
