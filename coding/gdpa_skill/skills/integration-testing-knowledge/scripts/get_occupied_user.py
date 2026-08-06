#!/usr/bin/env python3
"""
Get test account from duck Account service.

Usage:
    python get_occupied_user.py [--test_env ENV]

Environment variables (optional):
    USER_TOKEN: Token for authentication (default: public token)
    COLLECTION_ID: Collection ID for test data

Output: JSON with test account information
"""

import os
import sys
import json
import argparse
import tempfile

try:
    from duck import Account
except ImportError:
    print(json.dumps({"error": "duck library not installed. Run: pip install byted-duck"}))
    sys.exit(1)


def get_account(
    test_env: str = None,
) -> dict:
    """
    Get test accounts from duck Account service.

    Args:
        test_env: Test environment (e.g., sg, va, boe)

    Returns:
        dict with accounts list or error message
    """
    original_cwd = os.getcwd()

    try:
        tmp_dir_path=tempfile.mkdtemp(prefix="gdpa_skill_", dir="/tmp")
        os.chdir(tmp_dir_path) # in case too many weird logs added to original_dir

        account_client = Account(test_env=test_env,
                                 use_sdk_token=False,
                                 collect_id=os.environ.get("COLLECTION_ID"),
                                 user_token=os.environ.get("USER_TOKEN", "a3ba23e9478d1d4cc8dc49b4f7d94946"))

        accounts, err = account_client.get(occupy=False, num=1)

        if err and err != "success":
            return {"error": str(err), "accounts": []}

        # Extract useful fields
        result = []
        for acc in accounts:
            result.append({
                "user_id": acc.get("uid"),
                "device_id": acc.get("did"),
                "session_key": acc.get("sessionid"),
                "tags": acc.get("tags"),
                "mobile": acc.get("mobile"),
                "username": acc.get("username"),
                "store_country": acc.get("store_country"),
            })

        return {"accounts": result, "count": len(result)}

    except Exception as e:
        return {"error": str(e), "accounts": []}
    finally:
        os.chdir(original_cwd)


def main():
    parser = argparse.ArgumentParser(description="Get test account from duck Account service")
    parser.add_argument("--test_env", type=str, default="sg", help="Test environment (e.g., sg, va, boe)")

    args = parser.parse_args()

    result = get_account(test_env=args.test_env)
    print(json.dumps(result, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
