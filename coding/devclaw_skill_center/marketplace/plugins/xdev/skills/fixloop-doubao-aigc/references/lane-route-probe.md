# 泳道路由探针

快速验证"BFF 是否识别本泳道"的一次性脚本。部署完一个新 lane 后，直接跑 probe 比等 pytest 跑完再排查问题快得多 —— probe 命中 200 说明环境平台侧 lane 注册到位；命中 400 就是 lane 没注册或 BFF 配置没同步。

## 适用场景

- `env create` 刚完成，想立刻确认 BFF 能否识别 lane（不用等服务部署）
- pytest 全部用例在第一跳 400 失败，怀疑 lane 路由，而非业务代码
- 切换新 lane 做对比实验

## 探针模板

在 `qa_model_effect` 仓库根目录放一个 `_probe_lane.py`（用完可删）：

```python
"""一次性探针：复现 test_*.py 第一步 alice.get_conversation_info_v2()，捕获原始 HTTP。"""
import json
import os
import sys

os.environ.setdefault("ENV_LABEL", "<lane-name>")  # 改成目标 lane
os.environ.setdefault("run_region", "cn")
os.environ.setdefault("RUNTIME_ENV", "online")
os.environ.setdefault("PYTHONPATH", ".")

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from evtest.env import Env
from common.im.api import AliceApi
from common.account import count_pool
from api.flow_im_gateway.flow_im_gateway import FlowImGateway

BOT_ID = "7234781073513644036"  # 豆包主 bot
POOL_USER = "liweiyi.11"
TAG_LIST = ["gen_video"]
REGION = "cn"

print(f"[probe] ENV_LABEL={os.environ['ENV_LABEL']} region={REGION}")
pool = count_pool.CountPool(POOL_USER, TAG_LIST, REGION, case_title="probe_lane")
account = pool.occupy_account(wait_time=3 * 60, use_time=60 * 10)
print(f"[probe] got account uid={account.get('uid')}")

try:
    alice = AliceApi(Env.Online, region=REGION, account=account)
    print(f"[probe] X-TT-ENV={alice.header.get('X-TT-ENV')} X-USE-PPE={alice.header.get('X-USE-PPE')}")

    resp = FlowImGateway.get_conversation_info(
        client=alice.client, query=alice.query,
        body={
            "cmd": 1110, "sequence_id": "probe-0001",
            "uplink_body": {"get_conv_info_uplink_body": {
                "bot_id": BOT_ID, "conversation_id": "",
                "conversation_type": 3, "ext": {"cold_start": "false"},
            }},
            "version": "1",
        },
        check_200=False, convert_json=False,
    )
    print("=" * 60)
    print(f"status_code: {resp.status_code}")
    print(f"X-Tt-Logid:  {resp.headers.get('X-Tt-Logid')}")
    print(f"url:         {resp.url}")
    if resp.status_code != 200:
        print(f"body: {resp.text[:600]}")
finally:
    pool.release_account(uid=account.get("uid"))
```

## 运行方式

```bash
cd $TEST_REPO_PATH   # qa_model_effect
ENV_LABEL=<lane-name> PYTHONPATH=. python3 _probe_lane.py 2>&1 | \
  grep -E "ENV_LABEL|status_code|X-Tt-Logid|url:|body:"
```

## 判断

| 结果 | 含义 |
|------|------|
| `status_code: 200` + `X-Tt-Logid` | BFF 识别 lane，路由打通（即使该 lane 还没部署服务，也会 fallback 到 baseline，返回 200） |
| `status_code: 400` | **BFF 查不到该 lane**。原因：lane 没在环境平台注册（检查是否走了 `env create`，是不是泳道名前缀/正则不符，是不是 `standard-env` 填错） |
| 账号占用超时 | CountPool 账号池耗尽，改一个 `POOL_USER` 试 |

## 注意

- probe 发的请求会真实计费/留痕，不要滥用（每次只跑一次验证用）
- 跑完请删 `_probe_lane.py`（或提交时别加进去）—— 它带 hardcoded BOT_ID / POOL_USER，留在仓库会误导别人
- 用完 `pool.release_account(uid=...)` 必须调，不然账号会被卡着 10 分钟
