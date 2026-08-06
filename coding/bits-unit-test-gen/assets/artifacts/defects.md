# 缺陷输出规范

将 Step 4 产出的 `BUG_MAP` 转换为本文件定义的缺陷格式，以 **JSONL**（每行一个 JSON 对象，无数组包裹）写入 ${GEN_BUG_OUTPUT_FILE_PATH} 指定的文件中。若无缺陷，写入空文件（0 字节）。

1. ${GEN_BUG_OUTPUT_FILE_PATH} 如果在上下文中不存在， 建议使用 shell 指令从系统环境变量读取。
2. 每条缺陷必须包含以下字段：

| 字段 | 类型 | 说明 | 约束 |
|------|------|------|------|
| `test_file` | string | 测试文件相对路径 | |
| `test_func` | string | 触发缺陷的测试函数名 | |
| `scenario` | string | 触发条件描述 | 如「输入参数为 0 时」 |
| `evidence` | string | 缺陷定位与原因 | 格式：`「代码片段」的 X:Y 行触发错误：具体描述` |
| `bug_type` | string | 缺陷类型 | 枚举值定义见 `${SKILL_ROOT}/references/detect-bugs.md` 缺陷类型分类表 |
| `bug_range` | int[][] | 缺陷行号范围 | 如 `[[10,12],[15,16]]` |
| `file_path` | string | 业务代码文件相对路径 | |
| `target_func` | string | 业务函数名 | |
| `severity` | string | 严重程度 | 枚举：`low` / `medium` / `critical` |

示例：

```jsonl
{"test_file":"file_path_test.go","test_func":"TestAdd","scenario":"输入参数大于5时","evidence":"func Add(a int) int { return a + 1 } 的 10:12,15:16 行触发错误：返回值与预期不符","bug_type":"logic_errors","bug_range":[[10,12],[15,16]],"file_path":"a/b/c.go","target_func":"Add","severity":"critical"}