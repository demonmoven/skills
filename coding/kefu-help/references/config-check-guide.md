# 配置检测指南

## 概述

本文档提供配置检测的完整指南，包括Git SSH密钥检测、代码权限检测和环境配置检查。

## Git SSH密钥检测

### 检测流程

1. **检查SSH密钥是否存在**
2. **验证密钥是否添加到SSH代理**
3. **测试SSH连接**
4. **检查密钥权限设置**
5. **如果密钥不存在，自动生成并提示用户绑定**

### 检测命令

```bash
# 检查SSH密钥是否存在
ls -la ~/.ssh/

# 检查密钥是否添加到SSH代理
ssh-add -l

# 测试SSH连接
ssh -T git@code.bytedance.net

# 检查密钥权限
ls -l ~/.ssh/id_rsa*  # 应该是 600 权限
```

### 自动检测和生成流程

当检测到没有权限时，按照以下流程处理：

```bash
#!/bin/bash

# 自动SSH密钥检测和生成脚本
echo "=== SSH密钥检测和配置 ==="

# 1. 检查SSH密钥是否存在
if [ ! -f ~/.ssh/id_rsa ]; then
    echo "✗ SSH密钥不存在，开始自动生成..."
    
    # 生成SSH密钥（使用公司邮箱）
    echo "请输入您的公司邮箱地址："
    read email
    
    ssh-keygen -t rsa -b 4096 -C "$email" -N "" -f ~/.ssh/id_rsa
    
    if [ $? -eq 0 ]; then
        echo "✓ SSH密钥生成成功"
    else
        echo "✗ SSH密钥生成失败"
        exit 1
    fi
else
    echo "✓ SSH密钥已存在"
fi

# 2. 修复密钥权限
chmod 600 ~/.ssh/id_rsa
chmod 644 ~/.ssh/id_rsa.pub
echo "✓ 密钥权限设置完成"

# 3. 添加密钥到SSH代理
ssh-add ~/.ssh/id_rsa > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ 密钥已添加到SSH代理"
else
    echo "⚠ 需要先启动SSH代理"
    eval `ssh-agent -s`
    ssh-add ~/.ssh/id_rsa
fi

# 4. 显示公钥内容并提示用户绑定
echo "\n=== SSH公钥内容 ==="
echo "请将以下公钥内容复制并绑定到代码平台："
echo "----------------------------"
cat ~/.ssh/id_rsa.pub
echo "----------------------------"

echo "\n=== SSH密钥绑定步骤 ==="
echo "1. 访问代码平台密钥配置页面："
echo "   https://code.byted.org/profile/keys"
echo ""
echo "2. 点击 'Add Key' 按钮"
echo ""
echo "3. 粘贴上述公钥内容"
echo ""
echo "4. 设置密钥名称（如：My-Laptop）"
echo ""
echo "5. 点击 'Add Key' 完成绑定"
echo ""

# 5. 测试连接
echo "=== 测试SSH连接 ==="
echo "请先完成上述密钥绑定步骤，然后按回车继续测试..."
read

ssh -T git@code.byted.org
if [ $? -eq 1 ]; then
    echo "✓ SSH连接测试成功"
else
    echo "✗ SSH连接测试失败，请检查密钥是否正确绑定"
fi

echo "\n=== 配置完成 ==="
```

### 密钥生成指南

如果没有SSH密钥，按照以下步骤生成：

```bash
# 生成SSH密钥（使用公司邮箱）
ssh-keygen -t rsa -b 4096 -C "your-email@bytedance.com"

# 添加密钥到SSH代理
ssh-add ~/.ssh/id_rsa

# 查看公钥内容（需要复制到代码平台）
cat ~/.ssh/id_rsa.pub
```

### SSH密钥配置步骤

1. **登录代码平台**：https://code.byted.org
2. **进入SSH密钥设置**：https://code.byted.org/profile/keys
3. **添加密钥**：粘贴公钥内容，设置密钥名称
4. **验证配置**：运行 `ssh -T git@code.byted.org` 测试连接

### 常见问题

| 问题 | 原因 | 解决方案 |
|------|------|----------|
| 密钥不存在 | 未生成SSH密钥 | 自动生成密钥并提示用户绑定 |
| 权限错误 | 密钥文件权限过高 | `chmod 600 ~/.ssh/id_rsa` |
| 连接失败 | 密钥未配置到代码平台 | 访问 https://code.byted.org/profile/keys 绑定密钥 |
| 代理问题 | 密钥未添加到代理 | `ssh-add ~/.ssh/id_rsa` |
| 链接错误 | 使用了错误的配置链接 | 使用 https://code.byted.org/profile/keys |

## 代码权限检测

### 权限检测流程

1. **验证仓库访问权限**
2. **检查分支权限**
3. **验证操作权限**

### 检测命令

```bash
# 验证仓库访问权限
git clone <repo-url>  # 测试克隆

# 测试推送权限
git push origin test-branch  # 测试推送

# 检查当前用户信息
git config user.name
git config user.email
```

### 权限申请

如果权限不足，需要：

1. **确定需要的权限级别**
2. **联系仓库管理员**
3. **提供个人信息**
4. **等待审批**

## 环境配置检查

### 系统环境检查

```bash
# 检查Git版本
git --version

# 检查操作系统
uname -a

# 检查网络连接
ping code.byted.org

# 检查DNS配置
nslookup code.byted.org
```

### Git配置检查

```bash
# 查看全局Git配置
git config --list

# 检查用户名配置
git config user.name

# 检查邮箱配置
git config user.email

# 检查SSH配置
git config core.sshCommand
```

### 网络环境检查

```bash
# 检查防火墙设置
# 检查代理配置
# 检查SSL证书
```

## 自动检测工具

### 配置检查脚本

```bash
#!/bin/bash

# 配置检测脚本
echo "=== 配置检测开始 ==="

echo "\n1. SSH密钥检测"
if [ -f ~/.ssh/id_rsa ]; then
    echo "✓ SSH私钥存在"
else
    echo "✗ SSH私钥不存在"
fi

if [ -f ~/.ssh/id_rsa.pub ]; then
    echo "✓ SSH公钥存在"
else
    echo "✗ SSH公钥不存在"
fi

echo "\n2. SSH代理检测"
ssh-add -l > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ SSH代理正常"
else
    echo "✗ SSH代理异常"
fi

echo "\n3. Git连接测试"
ssh -T git@code.byted.org > /dev/null 2>&1
if [ $? -eq 1 ]; then  # 正常情况下返回1
    echo "✓ Git连接正常"
else
    echo "✗ Git连接异常"
fi

echo "\n4. Git配置检查"
git --version > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ Git已安装"
else
    echo "✗ Git未安装"
fi

echo "\n5. 网络连接检查"
ping -c 3 code.byted.org > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ 网络连接正常"
else
    echo "✗ 网络连接异常"
fi

echo "\n=== 配置检测完成 ==="
```

## 修复指南

### SSH密钥问题修复

1. **生成新密钥**：`ssh-keygen -t rsa -b 4096 -C "your-email@bytedance.com"`
2. **添加到代理**：`ssh-add ~/.ssh/id_rsa`
3. **配置到代码平台**：复制公钥内容到 https://code.byted.org/profile/keys
4. **测试连接**：`ssh -T git@code.byted.org`

### 权限问题修复

1. **联系仓库管理员**
2. **提供个人信息**
3. **说明需要的权限级别**
4. **等待权限审批**
5. **验证权限生效**

### 环境配置修复

1. **安装Git**：根据操作系统安装Git
2. **配置Git**：设置用户名和邮箱
3. **检查网络**：确保网络连接正常
4. **配置代理**：如果需要，设置Git代理

## 最佳实践

1. **定期检查**：定期运行配置检测脚本
2. **备份密钥**：备份SSH密钥到安全位置
3. **使用SSH**：优先使用SSH协议而非HTTPS
4. **最小权限**：只申请必要的权限
5. **定期更新**：定期更新Git和相关工具

## 故障排除

### 连接超时

**原因**：
- 网络问题
- 防火墙阻止
- 代理配置错误

**解决方案**：
- 检查网络连接
- 检查防火墙设置
- 配置正确的代理

### 权限被拒绝

**原因**：
- SSH密钥未配置
- 密钥权限错误
- 未添加到代码平台

**解决方案**：
- 重新生成密钥
- 修复权限设置
- 添加公钥到代码平台

### 认证失败

**原因**：
- 用户名或邮箱错误
- 密钥不匹配
- 权限已被撤销

**解决方案**：
- 检查Git配置
- 重新生成密钥
- 联系管理员检查权限

## 联系支持

如果遇到无法解决的问题，请联系：

- **IT支持**：it-helpdesk@bytedance.com
- **代码平台支持**：code-platform@bytedance.com
- **Git支持**：git-support@bytedance.com
