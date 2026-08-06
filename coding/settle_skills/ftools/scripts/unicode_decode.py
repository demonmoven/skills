
import json
import argparse


def decode_unicode_json(unicode_str):
    try:
        # 第一步：解析JSON字符串为Python字典（解析过程会自动解码Unicode）
        decoded_dict = json.loads(unicode_str)
        return decoded_dict
    except json.JSONDecodeError as e:
        print(f"JSON解析失败：{e}")
        return None


def main():
    parser = argparse.ArgumentParser(
        description="Run link task and get result.")
    parser.add_argument(
        "--encode-str",
        type=str,
        required=True,
        help="encodeStr")

    args = parser.parse_args()

    # 待解码的Unicode字符串（注意原始字符串前加r避免转义冲突）
    original_str = args.encode_str

    # 执行解码
    result = decode_unicode_json(original_str)
    print(result)
    return result

if __name__ == '__main__':
    main()