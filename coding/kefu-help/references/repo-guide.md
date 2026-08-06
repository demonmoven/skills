# 代码仓库管理指南

## 概述

本文档提供代码仓库管理的完整指南，包括业务平台代码仓库表查询、Git仓库地址获取和仓库权限管理。

## 业务平台代码仓库表

### 仓库表地址

业务平台代码仓库表存储在飞书文档中：

- **文档地址**: [https://bytedance.larkoffice.com/wiki/RwpAwBKQqiu3zgkUKF1cexa6n1u](https://bytedance.larkoffice.com/wiki/RwpAwBKQqiu3zgkUKF1cexa6n1u)
- **文档名称**: 代码仓库

### 查询流程

1. **确定业务平台**：明确用户需要的业务平台名称
2. **访问仓库表**：打开上述飞书文档
3. **查找对应仓库**：在文档中找到相应业务平台的Git仓库地址
4. **返回仓库信息**：提供完整的Git仓库地址和相关信息

## Git仓库地址获取

### 格式规范

Git仓库地址通常采用以下格式：

- **SSH格式**: `git@code.bytedance.net:<project>/<repo>.git`
- **HTTPS格式**: `https://code.bytedance.net/<project>/<repo>.git`

### 推荐使用

优先使用SSH格式，因为：
1. 更安全（基于密钥认证）
2. 不需要每次输入密码
3. 速度更快

## 仓库权限管理

### 权限级别

| 权限级别 | 描述 | 对应操作 |
|---------|------|----------|
| 访客 | 只读权限 | 查看代码 |
| 开发者 | 读写权限 | 提交代码、创建分支 |
| 维护者 | 管理权限 | 合并MR、管理成员 |
| 所有者 | 完全权限 | 所有操作 |

### 权限申请流程

1. **确认需要的权限级别**
2. **访问权限申请页面**
3. **提供个人信息**（姓名、工号、邮箱）
4. **等待权限审批**
5. **验证权限生效**

### 权限申请链接

当遇到代码仓库权限不足时，可以通过以下方式申请权限：

#### 方式一：代码平台权限申请页面

访问代码平台的权限管理页面：

```
https://code.byted.org/<project>/<repo>/project_members
```

**示例**：
- 仓库地址：`git@code.byted.org:ies-cs/trace_go.git`
- 权限申请链接：`https://code.byted.org/ies-cs/trace_go/project_members`

#### 方式二：仓库页面直接申请

1. 访问代码平台：https://code.byted.org
2. 搜索或访问目标仓库
3. 点击 "Members" 或 "项目成员" 标签
4. 点击 "Request Access" 或 "申请权限" 按钮
5. 填写申请理由并提交

#### 方式三：联系仓库管理员

如果以上方式无法申请，可以联系仓库管理员：

1. 在仓库页面查看管理员信息
2. 通过飞书或邮件联系管理员
3. 提供个人信息和申请理由
4. 等待管理员添加权限

### 自动权限检测和申请提示

当检测到仓库权限不足时，系统会自动：

1. **解析仓库地址**：从 Git URL 中提取项目和仓库名称
2. **生成权限申请链接**：自动生成代码平台权限申请页面链接
3. **提供申请指南**：显示详细的申请步骤和注意事项

**示例脚本**：

```bash
#!/bin/bash

# 权限检测和申请链接生成脚本
function check_and_generate_permission_link() {
    local repo_url=$1
    
    echo "=== 仓库权限检测 ==="
    echo "检测仓库: $repo_url"
    echo ""
    
    # 尝试克隆测试
    TMP_DIR=$(mktemp -d)
    git clone --depth 1 $repo_url $TMP_DIR > /dev/null 2>&1
    
    if [ $? -eq 0 ]; then
        echo "✓ 权限正常，可以访问仓库"
        rm -rf $TMP_DIR
        return 0
    else
        echo "✗ 权限不足，无法访问仓库"
        echo ""
        
        # 解析仓库地址并生成权限申请链接
        if [[ $repo_url =~ git@code\.byted\.org:(.*)/(.*)\.git ]]; then
            local project=${BASH_REMATCH[1]}
            local repo=${BASH_REMATCH[2]}
            
            echo "=== 权限申请信息 ==="
            echo "项目名称: $project"
            echo "仓库名称: $repo"
            echo ""
            echo "=== 权限申请链接 ==="
            echo "代码平台权限申请页面："
            echo "  https://code.byted.org/$project/$repo/project_members"
            echo ""
            echo "=== 申请步骤 ==="
            echo "1. 点击上述链接访问权限申请页面"
            echo "2. 点击 'Request Access' 或 '申请权限' 按钮"
            echo "3. 填写申请理由（如：需要访问代码进行开发）"
            echo "4. 提交申请并等待审批"
            echo ""
            echo "=== 其他申请方式 ==="
            echo "- 访问代码平台: https://code.byted.org"
            echo "- 搜索仓库名称: $project/$repo"
            echo "- 在仓库页面点击 'Members' 标签申请权限"
            
        elif [[ $repo_url =~ https://code\.byted\.org/(.*)/(.*)\.git ]]; then
            local project=${BASH_REMATCH[1]}
            local repo=${BASH_REMATCH[2]}
            
            echo "=== 权限申请信息 ==="
            echo "项目名称: $project"
            echo "仓库名称: $repo"
            echo ""
            echo "=== 权限申请链接 ==="
            echo "代码平台权限申请页面："
            echo "  https://code.byted.org/$project/$repo/project_members"
            echo ""
            echo "=== 申请步骤 ==="
            echo "1. 点击上述链接访问权限申请页面"
            echo "2. 点击 'Request Access' 或 '申请权限' 按钮"
            echo "3. 填写申请理由（如：需要访问代码进行开发）"
            echo "4. 提交申请并等待审批"
            echo ""
            echo "=== 其他申请方式 ==="
            echo "- 访问代码平台: https://code.byted.org"
            echo "- 搜索仓库名称: $project/$repo"
            echo "- 在仓库页面点击 'Members' 标签申请权限"
        else
            echo "⚠ 无法解析仓库地址格式"
            echo ""
            echo "请手动访问代码平台申请权限："
            echo "1. 访问 https://code.byted.org"
            echo "2. 搜索目标仓库"
            echo "3. 在仓库页面申请权限"
        fi
        
        rm -rf $TMP_DIR
        return 1
    fi
}

# 使用示例
if [ $# -eq 0 ]; then
    echo "用法: $0 <repo_url>"
    echo "示例: $0 git@code.byted.org:ies-cs/trace_go.git"
    exit 1
fi

check_and_generate_permission_link "$1"
```

## 常见问题

### 无法访问仓库

**原因**：
- 没有仓库权限
- Git SSH密钥未配置
- 网络连接问题

**解决方案**：
- 申请仓库权限
- 配置Git SSH密钥
- 检查网络连接

### 权限不足

**原因**：
- 当前权限级别不够
- 分支保护设置

**解决方案**：
- 申请更高权限
- 联系仓库管理员处理

### 仓库地址变更

**原因**：
- 项目重命名
- 组织结构调整

**解决方案**：
- 从业务平台代码仓库表获取最新地址
- 更新本地仓库远程地址

## 最佳实践

1. **定期更新仓库表**：确保仓库地址信息最新
2. **统一使用SSH格式**：提高安全性和便捷性
3. **最小权限原则**：只申请必要的权限级别
4. **定期检查权限**：确保权限设置合理
5. **备份重要仓库**：防止意外情况

## 相关命令

### 查看远程仓库地址

```bash
git remote -v
```

### 更新远程仓库地址

```bash
git remote set-url origin <new-url>
```

### 测试仓库连接

```bash
git ls-remote <repo-url>
```

## 故障排除

### SSH连接问题

**症状**：`Permission denied (publickey)`

**解决方案**：
1. 检查SSH密钥是否存在：`ls -la ~/.ssh/`
2. 检查密钥是否添加到SSH代理：`ssh-add -l`
3. 检查密钥是否配置到代码平台

### 权限验证

**验证方法**：
1. 尝试克隆仓库：`git clone <repo-url>`
2. 尝试推送测试：`git push origin test-branch`
3. 检查仓库权限设置

## 联系支持

如果遇到无法解决的问题，请联系：

- **代码平台支持**：code-platform@bytedance.com
- **仓库管理员**：在仓库页面查看
- **IT支持**：it-helpdesk@bytedance.com
