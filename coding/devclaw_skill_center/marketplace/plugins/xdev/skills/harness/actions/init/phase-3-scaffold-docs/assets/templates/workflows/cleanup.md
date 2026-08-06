# Cleanup Workflow

本文件定义低风险清理工作流，Agent 可以在不需要完整 ExecPlan 的情况下执行。

## Allowed (Low Risk)

以下操作可以在任何时候安全执行：

- Remove unused imports / variables / functions
- Fix typos in comments and documentation
- Add missing type annotations
- Normalize formatting (let the formatter handle it)
- Remove dead code that is clearly unreachable
- Update outdated comments to match current code behavior

## Not Allowed Without ExecPlan

以下操作需要先创建 ExecPlan，因为有破坏性风险：

- Rename public functions, types, or modules
- Change function signatures
- Modify database schemas or migration scripts
- Alter configuration file formats
- Restructure directory layout
- Remove apparently-unused code that might be used via reflection/dynamic dispatch

## Cleanup Loop

1. Identify the cleanup opportunity
2. Check if it's in the "Allowed" list above
3. If allowed: make the change, run tests, commit
4. If not allowed: log it in `docs/quality/debt-log.md` for later
5. Never mix cleanup with feature work in the same commit
