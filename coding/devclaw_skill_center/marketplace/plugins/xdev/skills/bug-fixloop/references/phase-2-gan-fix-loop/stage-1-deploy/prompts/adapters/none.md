# Stage 1 / Deploy / Adapter: none

> PROFILE=none 的部署 adapter。**不执行实际部署**，仅校验用户提供的 BASE_URL 可达。
>
> 主 context 直接 Read 本文件，按以下流程执行。

## 适用场景

用户已经把服务自行部署在某个地方（本地、CI、自建 k8s 等），只需要让 fix-loop 测试它，不需要 fix-loop 帮忙做部署。

## 输入

- `BASE_URL`：服务的可访问地址，例如：
  - `http://localhost:8080`
  - `http://192.168.1.100:8080`
  - `https://api.staging.example.com`

## 流程

### Step 1: BASE_URL 可达性校验

```bash
# 用 curl 探测 base_url 是否能连接
http_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$BASE_URL" 2>/dev/null)

if [ -z "$http_code" ] || [ "$http_code" = "000" ]; then
  echo "ERROR: 无法连接到 $BASE_URL"
  echo "请检查："
  echo "  1. 服务是否已启动"
  echo "  2. BASE_URL 是否正确（含协议和端口）"
  echo "  3. 网络是否可达（防火墙 / VPN）"
  exit 1
fi

# 任意 HTTP 状态码都算"可达"（即使是 404，也说明服务在响应）
echo "BASE_URL 可达：$BASE_URL (HTTP $http_code)"
```

### Step 2: 跳过部署相关动作

PROFILE=none 不需要：
- bytedcli tce deploy-lane
- 实例状态轮询
- 部署失败修复子循环

直接返回成功，进入 stage-2 测试。

## 跳过条件

无（PROFILE=none 时本 adapter 必须执行）。

## 错误处理

- BASE_URL 不可达 → **阻断主流程**，提示用户先确保服务已启动并可访问
- 用户未提供 BASE_URL → 在 phase-0 wizard 中应已收集，到这一步还没有则报错

## 与 PROFILE=bytedance-tce 的差异

| 维度 | bytedance-tce | none |
|------|--------|------|
| 是否部署 | 是（TCE deploy-lane） | 否 |
| 是否轮询实例 | 是 | 否 |
| 失败修复子循环 | 有（最多 3 次） | 无 |
| 输入参数 | PSM, BRANCH, BUSINESS_REPO 等 | BASE_URL |
| 产出 | TCE_LANE 泳道名 | 无（BASE_URL 直接传递给 stage-2） |

## 后续

主 context 把 `BASE_URL` 传递给 stage-2-test 的 adapter（也是 none 版本），由 stage-2 负责把 BASE_URL 注入测试 config。
