#!/usr/bin/env bash

set -ex

TMP_DIR="/tmp/bag_tools_$(openssl rand -hex 4)"
mkdir -p $TMP_DIR
cd $TMP_DIR
git init
git remote add origin https://oauth2:$CUSTOM_BAG_TOOL_OAUTH2@code.byted.org/webcast/bag_tools.git
git fetch origin idl_cleaner:idl_cleaner
git checkout idl_cleaner
tar -zxvf idl_cleaner.tar.gz
mv idl_cleaner ~/.tools/