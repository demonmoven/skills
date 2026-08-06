---
name: unittest
description: 指导AI为Go项目生成高质量的单元测试，确保测试覆盖率≥85%且所有测试通过。文档包含测试规范、覆盖率计算、自动检测命令和示例代码。
---

# Go单元测试生成指南

## 1. 概述
本文档指导AI如何为Go项目生成高质量的单元测试，确保测试覆盖率≥85%且所有测试通过。文档包含测试规范、覆盖率计算、自动检测命令和示例代码。

## 2. Go单元测试基础

### 2.1 测试文件命名
- 测试文件必须以`_test.go`结尾（如`service_test.go`）
- 测试文件应与被测试文件放在同一目录下

### 2.2 测试函数命名
- 测试函数名必须以`Test`开头，后面跟被测试函数名（如`TestMOQLDiff`）
- 测试函数签名：`func TestXxx(t *testing.T)`

### 2.3 表驱动测试
推荐使用表驱动测试提高覆盖率：
```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 1, 2, 3},
        {"negative numbers", -1, -2, -3},
        {"mixed numbers", 1, -2, -1},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
            }
        })
    }
}
```

## 3. 测试覆盖率要求

### 3.1 覆盖率计算指标
- 语句覆盖率：执行到的代码行数占总代码行数的百分比
- 分支覆盖率：执行到的分支占总分支数的百分比
- 函数覆盖率：执行到的函数占总函数数的百分比

### 3.2 覆盖率目标
- 整体覆盖率：≥85%
- 核心业务逻辑：≥90%
- 公共API：100%

## 4. 自动执行测试和检测覆盖率

### 4.1 运行所有测试
```bash
# 运行项目所有测试
make test

# 或直接使用go test
cd /Users/bytedance/go/src/code.byted.org/bits/search
go test ./...
```

### 4.2 计算测试覆盖率
```bash
# 运行测试并生成覆盖率报告
cd /Users/bytedance/go/src/code.byted.org/bits/search
go test -cover ./...

# 生成覆盖率profile文件
go test -coverprofile=coverage.out ./...

# 查看详细覆盖率报告
go tool cover -func=coverage.out

# 生成HTML覆盖率报告
go tool cover -html=coverage.out -o coverage.html
```

## 5. 针对不同类型代码的测试策略

### 5.1 函数测试
- 测试正常路径和异常路径
- 测试边界条件
- 使用mock替代外部依赖

### 5.2 结构体方法测试
- 测试结构体的所有方法
- 测试不同状态下的方法行为
- 使用接口模拟依赖

### 5.3 HTTP/GRPC接口测试
- 使用httptest/grpc/testing模拟服务
- 测试请求参数验证
- 测试响应格式和错误处理

### 5.4 数据库操作测试
- 使用内存数据库或mock
- 测试CRUD操作
- 测试事务处理

## 6. 测试工具推荐

### 6.1 断言库
- `github.com/stretchr/testify/assert`：提供丰富的断言函数
- `github.com/google/go-cmp/cmp`：高级比较库，支持自定义比较规则

### 6.2 Mock工具
- `github.com/golang/mock/gomock`：官方mock工具
- `github.com/stretchr/testify/mock`：与assert集成的mock工具

### 6.3 测试生成工具
- `gotests`：自动生成测试框架
- `go-testdeep`：深度比较工具

## 7. 示例：为MOQLDiff功能生成单元测试

### 7.1 被测试函数
```go
// service.go
func (s *Service) MOQLDiff(ctx context.Context, req *moqlsearch.MOQLDiffRequest) (*moqlsearch.MOQLDiffResponse, error) {
    // 实现逻辑
}
```

### 7.2 生成的测试函数
```go
// service_test.go
package impl

import (
    "context"
    "testing"

    "github.com/google/go-cmp/cmp"
    "github.com/stretchr/testify/assert"
    
    "code.byted.org/bits/search/engine/business/moql/kitex_gen/moqlsearch"
)

func TestService_MOQLDiff(t *testing.T) {
    tests := []struct {
        name           string
        req            *moqlsearch.MOQLDiffRequest
        expectedResp   *moqlsearch.MOQLDiffResponse
        expectedErr    error
        setupMocks     func()
    }{
        {
            name: "same content",
            req: &moqlsearch.MOQLDiffRequest{
                MOQLResponse:   &moqlsearch.MOQLSearchResponse{},
                ViewResponse:   &moqlsearch.MOQLSearchResponse{},
            },
            expectedResp: &moqlsearch.MOQLDiffResponse{
                IsSame:     boolPtr(true),
                DiffReason: nil,
            },
            expectedErr: nil,
            setupMocks:  func() {},
        },
        {
            name: "different content",
            req: &moqlsearch.MOQLDiffRequest{
                MOQLResponse:   &moqlsearch.MOQLSearchResponse{List: []*moqlsearch.SearchInfo{{GroupInfos: []*moqlsearch.MOQLGroupInfo{{GroupName: "test1"}}}}},
                ViewResponse:   &moqlsearch.MOQLSearchResponse{List: []*moqlsearch.SearchInfo{{GroupInfos: []*moqlsearch.MOQLGroupInfo{{GroupName: "test2"}}}}},
            },
            expectedResp: &moqlsearch.MOQLDiffResponse{
                IsSame:     boolPtr(false),
                DiffReason: stringPtr("MOQLResponse.List[0].GroupInfos[0].GroupName: expected 'test2', got 'test1'"),
            },
            expectedErr: nil,
            setupMocks:  func() {},
        },
        {
            name: "nil request",
            req:  nil,
            expectedResp: nil,
            expectedErr:  assert.AnError,
            setupMocks:   func() {},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 初始化测试环境
            s := NewService()
            
            // 设置mock
            tt.setupMocks()
            
            // 执行测试
            resp, err := s.MOQLDiff(context.Background(), tt.req)
            
            // 验证结果
            if tt.expectedErr != nil {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.expectedResp.IsSame, resp.IsSame)
            assert.Equal(t, tt.expectedResp.DiffReason, resp.DiffReason)
            
            // 使用go-cmp进行深度比较
            if diff := cmp.Diff(tt.expectedResp, resp); diff != "" {
                t.Errorf("MOQLDiff() mismatch (-want +got):\n%s", diff)
            }
        })
    }
}

// 辅助函数
func boolPtr(b bool) *bool {
    return &b
}

func stringPtr(s string) *string {
    return &s
}
```

## 8. 自动检测流程

AI应执行以下步骤确保测试质量：

1. **生成测试代码**：根据被测试代码生成完整的单元测试
2. **运行测试**：执行`go test ./...`确保所有测试通过
3. **计算覆盖率**：执行`go test -coverprofile=coverage.out ./...`
4. **检查覆盖率**：执行`go tool cover -func=coverage.out`并确保覆盖率≥85%，如果低于85%，则跳转至第1步继续补充测试，直至覆盖率≥85%
5. **生成报告**：如果需要，生成HTML覆盖率报告以便查看

## 9. 最佳实践

- **测试隔离**：每个测试用例应独立运行，不依赖其他测试
- **测试命名**：使用清晰的测试名称描述测试场景
- **避免测试实现细节**：测试应关注接口而非实现细节
- **定期运行测试**：确保代码变更不会破坏现有功能
- **持续优化**：根据覆盖率报告优化测试，提高覆盖率

## 10. 常见问题和解决方案

### 10.1 覆盖率不足
- 检查未覆盖的代码路径
- 为边界条件和异常路径添加测试
- 避免测试代码中包含过多条件语句

### 10.2 测试失败
- 检查被测试代码是否有bug
- 确保测试环境与生产环境一致
- 检查mock是否正确设置

### 10.3 测试运行缓慢
- 减少测试中的IO操作
- 使用并行测试（`t.Parallel()`）
- 避免重复初始化资源
```

        
