import argparse
from datetime import datetime, timezone
import sys
import time

import requests
import json
import query_trace

headers = {
    "Accept": "application/json, text/plain, */*",
    "Accept-Language": "zh-CN,zh;q=0.9",
    "Connection": "keep-alive",
    "Content-Type": "application/json",
    "Origin": "https://f-tools.bytedance.net",
    "Referer": "https://f-tools.bytedance.net/repo/link_tool/usage?tool_id=42&source=9&navbar=false",
    "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36",
    "opt-user": "gary.0922",
    "sec-ch-ua": "\"Not(A:Brand\";v=\"8\", \"Chromium\";v=\"144\", \"Google Chrome\";v=\"144\"",
    "sec-ch-ua-mobile": "?0",
    "sec-ch-ua-platform": "\"macOS\"",
}


cookies = {
    "people-lang": "zh",
    "_sys_language_key": "zh-CN",
    "monitor_huoshan_web_id": "0905836613518819229",
    "user_locale": "zh",
}

def get_jwt_token():
    cache_path = query_trace._default_cache_path()
    try:
        cache = query_trace._load_cache(cache_path)
    except FileNotFoundError:
        cache = query_trace._run_login_flow(cache_path)
        if not cache:
            print(
                "Auto-login did not produce a token cache.",
                file=sys.stderr)
            return None
    jwt = cache.get("jwt_token") or ""
    custom_identity = cache.get("custom_identity") or ""
    user_agent = cache.get("user_agent") or "Mozilla/5.0"
    exp = query_trace._jwt_expiry(jwt)
    if exp and exp < int(time.time()):
        exp_dt = datetime.fromtimestamp(exp,tz=timezone.utc).isoformat()
        cache = query_trace._run_login_flow(
            cache_path)
        if not cache:
            print(
                f"Token expired at {exp_dt}. Auto-login failed.",
                file=sys.stderr)
            return 3
        jwt = cache.get("jwt_token") or ""
        custom_identity = cache.get("custom_identity") or ""
        user_agent = cache.get("user_agent") or "Mozilla/5.0"
    return jwt, custom_identity, user_agent

def run_link_task(settle_order_id="", trade_no="", charge_order_no=""):
    url = "https://f-tools.bytedance.net/api/link/task/run"

    payload = {
        "link_id": 42,
        "task_type": "node",
        "node_list": [
            430
        ],
        "params": f"{{\"SettleOrderId\":\"{settle_order_id}\",\"TradeNo\":\"{trade_no}\",\"ChargeOrderNo\":\"{charge_order_no}\"}}",
        "source": 1,
        "global_params": "{}"
    }
    jwt, custom_identity, user_agent = get_jwt_token()
    if not jwt:
        return None
    headers["x-jwt-token"] = jwt
    headers["x-custom-identity"] = custom_identity
    headers["User-Agent"] = user_agent

    cookies["x-jwt-token"] = jwt

    response = requests.post(url, headers=headers, json=payload, cookies=cookies)

    task_id = response.json()["data"]
    return task_id

def get_task_result(task_id):
    url = "https://f-tools.bytedance.net/api/link/task/report"
    payload = {
        "task_id": task_id,
    }
    jwt, custom_identity, user_agent = get_jwt_token()
    if not jwt:
        return None
    headers["x-jwt-token"] = jwt
    headers["x-custom-identity"] = custom_identity
    headers["User-Agent"] = user_agent

    cookies["x-jwt-token"] = jwt

    response = requests.post(url, headers=headers,  json=payload, cookies=cookies)
    print(
        f"Status Code: {response.status_code}")
    print(
        f"Response Body: {response.text}")

    # 3. 检查 task_status
    data = response.json()["data"]
    task_status = data.get(
        "task_status")



    if task_status == "SUCCESS":
        # 提取 task_output 字段
        task_output = data.get(
            "task_output")
        print(
            f"task_output 的值: '{task_output}'")

        # (可选) 如果 task_output 本身是 JSON 字符串，可以尝试进一步解析
        if task_output:
            try:
                task_output_json = json.loads(
                    task_output)
                print(
                    "task_output 解析后的数据:",
                    task_output_json)

                # 提取 "大模型语义化summary"
                summary = task_output_json.get(
                    "大模型语义化summary")
                print(
                    f"大模型语义化summary: '{summary}'")
                return True, summary

            except json.JSONDecodeError:
                print(
                    "task_output 不是 JSON 格式字符串")
        else:
            print(
                "注意: task_output 字段为空")
    else:
        print(
            "continue")
        return False, None

def main():
    parser = argparse.ArgumentParser(description="Run link task and get result.")
    parser.add_argument("--settle-order-id", type=str, required=True, help="SettleOrderId")
    parser.add_argument(
        "--trade-no",
        type=str,
        required=False,
        help="TradeNo")

    parser.add_argument(
        "--charge-order-no",
        type=str,
        required=False,
        help="ChargeOrderNo")



    args = parser.parse_args()

    task_id = run_link_task(settle_order_id=args.settle_order_id, trade_no=args.trade_no, charge_order_no=args.charge_order_no)

    ok, summary, cnt = False, None, 0
    while not ok and cnt < 20:
        time.sleep(1)
        ok, summary = get_task_result(task_id)
        cnt += 1


if __name__ == '__main__':
    main()