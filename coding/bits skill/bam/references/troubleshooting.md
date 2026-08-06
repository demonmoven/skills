# 常见问题与处理

## 工具版本过期

**在排查任何错误之前**，先确保 bytedcli 和已安装的技能是最新版本：

```bash
bytedcli self update
```

升级完成后重试原始命令。很多问题是由 CLI 版本和已安装技能之间的版本偏差引起的。

## 1. Missing command

- 原因：未指定子命令
- 处理：`bytedcli <group> --help`

## 2. Missing argument

- 原因：缺少必需位置参数
- 处理：使用 `--help` 查看参数

## 3. Not authenticated

- 原因：未登录或 Token 过期
- 处理：`bytedcli auth login`
- 补充：部分命令会按 `BYTEDCLI_USER_CLOUD_JWT -> AIME_USER_CLOUD_JWT` 或 `BYTEDCLI_USER_CODE_JWT -> AIME_USER_CODE_JWT` 自动回退；只有这些环境变量也不可用时才需要重新登录/配置

## 4. 获取字节云 JWT 失败: 401

- 原因：目标站点未认证。认证隔离按 SSO 环境生效：`i18n-tt`、`eu-ttp`（TikTok SSO）与 `cn/i18n-bd`（ByteDance SSO）隔离。即使 cn 已登录，操作 i18n-tt 仍需单独认证
- 处理：为目标站点执行登录
  ```bash
  # 以 i18n-tt 为例
  BYTEDCLI_CLOUD_SITE=i18n-tt bytedcli auth login
  ```
- 验证：`BYTEDCLI_CLOUD_SITE=i18n-tt bytedcli auth status`

## 5. 网络/权限问题

- 确认内网访问权限
- 确认已登录且 Token 有效
