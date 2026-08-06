# Troubleshooting

## 1. Missing command

- Cause: subcommand is not specified
- Fix:
  ```bash
  NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest <group> --help
  ```

## 2. Missing argument

- Cause: required positional argument is missing
- Fix: use `--help` to inspect available parameters

## 3. Not authenticated

- Cause: not logged in or token expired
- Fix:
  ```bash
  NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest auth login
  ```
- Note: some commands automatically fall back by mapping `BYTEDCLI_USER_CLOUD_JWT -> AIME_USER_CLOUD_JWT` or `BYTEDCLI_USER_CODE_JWT -> AIME_USER_CODE_JWT`. Only when these env vars are unavailable should you re-login / re-configure.

## 4. Failed to fetch ByteCloud JWT: 401

- Cause: the target site is not authenticated. Authentication isolation follows the SSO environment: `i18n-tt` (TikTok SSO) is isolated from `prod/i18n-bd/boei18n` (ByteDance SSO). Even if you are logged in to `prod`, you still need to log in separately for `i18n-tt`.
- Fix: login to the target site.
  ```bash
  # Example: i18n-tt
  BYTEDCLI_CLOUD_SITE=i18n-tt NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest auth login
  ```
- Verify:
  ```bash
  BYTEDCLI_CLOUD_SITE=i18n-tt NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest auth status
  ```

## 5. Network / permission issues

- Ensure you have intranet access
- Ensure you are logged in and the token is still valid
