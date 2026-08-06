import requests
import argparse

def query_commercial_product_records(merchant_id, cas_session_id):
    url = 'https://cgi-zg.douyinpay-zg.net/douyinpay-fe_merchant/merchant-manager/?method=PageQueryCommercialProductRecords'

    cookies = {
        'cas_session_id': cas_session_id
    }

    headers = {
        'Accept': 'application/json, text/plain, */*',
        'Accept-Language': 'zh-CN,zh;q=0.9',
        'Connection': 'keep-alive',
        'Content-Type': 'application/json; charset=UTF-8',
        'Origin': 'https://operation-zg.douyinpay-zg.net',
        'Referer': 'https://operation-zg.douyinpay-zg.net/',
        'Sec-Fetch-Dest': 'empty',
        'Sec-Fetch-Mode': 'cors',
        'Sec-Fetch-Site': 'same-site',
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36',
        'sec-ch-ua': '"Not:A-Brand";v="99", "Google Chrome";v="145", "Chromium";v="145"',
        'sec-ch-ua-mobile': '?0',
        'sec-ch-ua-platform': '"macOS"',
        'x-cgi-rpc-method': 'PageQueryCommercialProductRecords',
        'x-rpc-type': 'json',
        'x-tt-env': 'prod',
    }

    json_data = {
        'BizScenes': [],
        'StatusList': [],
        'MerchantId': merchant_id,
        'Page': {
            'PageNum': 1,
            'PageSize': 10,
        },
        'PlatformBizIdentities': [
            'DEFAULT_PRODUCT',
        ],
    }

    response = requests.post(url, cookies=cookies, headers=headers, json=json_data)

    try:
        return response.json()
    except requests.exceptions.JSONDecodeError:
        return response.text

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Query Commercial Product Records')
    parser.add_argument('-m', '--merchant_id', required=True, help='Merchant ID (e.g., 6020260317901302)')
    parser.add_argument('-c', '--jwt', required=True, help='jwt cookie value')
    
    args = parser.parse_args()
    
    result = query_commercial_product_records(args.merchant_id, args.jwt)
    print(result)
