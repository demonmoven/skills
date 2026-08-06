#!/usr/bin/env python3
"""
Read test data from duck DataHelper service.

Usage:
    python read_test_data.py <namespace> <data_key> [--test_env ENV]

Arguments:
    namespace: Namespace name for test data
    data_key: The test data key/name to retrieve

Environment variables (optional):
    USER_TOKEN: Token for authentication

Output: JSON with test data for the specified key
"""

import os
import sys
import json
import argparse
import tempfile

try:
    import duck
except ImportError:
    print(json.dumps({"error": "duck library not installed. Run: pip install byted-duck"}))
    sys.exit(1)

def read_test_data(
    namespace: str,
    data_key: str,
    test_env: str,
) -> dict:
    """
    Read test data from duck DataHelper service.

    Args:
        data_key: The test data key/name to retrieve
        namespace: Namespace for test data (overrides env var)
        test_env: Default boei18n;test env of cases,like sg|va|gcp|my|ttp|...

    Returns:
        dict with test data or error message
    """

    original_cwd = os.getcwd()

    try:
        tmp_dir_path=tempfile.mkdtemp(prefix="gdpa_skill_", dir="/tmp")
        os.chdir(tmp_dir_path) # in case too many weird logs added to original_dir


        # Build DataHelper constructor kwargs
        data_helper = duck.DataHelper(namespace_name=namespace,
                                      user_token=os.environ.get("USER_TOKEN", "a3ba23e9478d1d4cc8dc49b4f7d94946"))

        # Get test data for the specified key
        test_data = data_helper.get_test_data(data_key=data_key, test_env=test_env)

        if test_data is None:
            return {"error": f"No data found for key: {data_key}", "data": None}

        return {
            "data_key": data_key,
            "data": test_data,
        }

    except Exception as e:
        return {"error": str(e), "data": None}
    finally:
        os.chdir(original_cwd)

def main():
    parser = argparse.ArgumentParser(description="Read test data from duck DataHelper service")
    parser.add_argument("namespace", help="Namespace name for test data")
    parser.add_argument("data_key", help="The test data key/name to retrieve")
    parser.add_argument("--test_env", type=str, default="sg", help="Test environment (e.g., sg, va, gcp, my, ttp)")

    args = parser.parse_args()

    result = read_test_data(
        namespace=args.namespace,
        data_key=args.data_key,
        test_env=args.test_env
    )
    print(json.dumps(result, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
