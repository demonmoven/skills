# Go Mod 权限问题解决方案

## 问题场景

在执行 `go mod download` 或 `go mod tidy` 时，可能会遇到以下权限错误：

```
go: code.byted.org/xxx/xxx@v1.0.0: reading code.byted.org/xxx/xxx@v1.0.0/xxx.mod: 403 Forbidden
```

或

```
go get: module code.byted.org/xxx/xxx: git ls-remote -q origin in /path/to/repo exit status 128:
	fatal: could not read Username for 'https://code.byted.org': terminal prompts disabled
```

## 问题识别

当遇到以下情况时，说明是权限问题：

1. 错误信息包含 `403 Forbidden`
2. 错误信息包含 `permission denied`
3. 错误信息包含 `could not read Username`
4. 无法访问 `code.byted.org` 上的私有仓库

## 解决方案

### 一键申请权限

访问权限申请平台，一键申请所需仓库的访问权限：

**权限申请地址**: https://code.byted.org/permission_application

### 操作步骤

1. **访问权限申请平台**
   - 打开浏览器访问：https://code.byted.org/permission_application
   - 使用字节跳动内部账号登录

2. **填写申请信息**
   
   在权限申请页面需要填写以下信息：
   
   - **仓库名称**：填写需要访问的仓库名称（格式：`group/project`）
     
     **如何获取仓库名称**：
     
     在项目根目录下执行 `git remote -v` 命令，从输出中提取仓库名称：
     
     ```bash
     ➜  kefu_help git:(master) ✗ git remote -v
     origin  git@code.byted.org:ies-cs/kefu_help.git (fetch)
     origin  git@code.byted.org:ies-cs/kefu_help.git (push)
     ```
     
     从输出中可以看到仓库名称为：`ies-cs/kefu_help`
     
     - 格式说明：`git@code.byted.org:` 后面的部分，去掉 `.git` 后缀
     - 例如：`ies-cs/kefu_help`
   
   - **Branch（分支）**：可选字段
     - 可以不填写
     - 如需填写，通常填写 `master`
   
   - 点击"申请权限"按钮提交申请
   - 系统会自动处理权限申请

3. **等待审批**
   - 通常权限申请会在几分钟内自动审批完成
   - 如遇特殊情况，可能需要仓库管理员手动审批

4. **验证权限**
   - 权限生效后，重新执行 `go mod download`
   - 或执行 `go mod tidy` 验证是否可以正常下载依赖

## 常见问题

### Q: 权限申请后多久生效？
A: 通常几分钟内自动生效，如长时间未生效，请联系仓库管理员。

### Q: 如何查看已申请的权限？
A: 在权限申请平台可以查看所有已申请和已获得的权限列表。

### Q: 临时解决方案？
A: 可以尝试使用 `GOPRIVATE` 环境变量：
```bash
export GOPRIVATE=code.byted.org/*
```

## 相关链接

- 权限申请平台：https://code.byted.org/permission_application
- 字节跳动代码仓库：https://code.byted.org
