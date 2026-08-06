#!/usr/bin/env bash

CURDIR=$(cd $(dirname $0); pwd)

rm -rf output/
mkdir -p output/


cd $CURDIR/cmd/idl_cleaner
go build -ldflags="-w -s -X 'main.SCMVersion=$BUILD_VERSION' -X 'main.BuildUser=$BUILD_USER' -X 'main.Branch=$BUILD_BRANCH' -X 'main.DetailURL=https://cloud.bytedance.net/scm/detail/$BUILD_REPO_ID/versions?versionId=$BUILD_VERSION_ID' -X 'main.CommitHash=$BUILD_BASE_COMMIT_HASH'" -o $CURDIR/output/idl_cleaner 

if [ "$TASK_FROM" = "SCM" ]; then
    TMP_DIR="/tmp/bag_tools_$(openssl rand -hex 4)"
    mkdir -p $TMP_DIR
    cd $TMP_DIR
    git init
    git remote add origin https://oauth2:$CUSTOM_BAG_TOOL_OAUTH2@code.byted.org/webcast/bag_tools.git
    git checkout -b idl_cleaner
    tar -czf idl_cleaner.tar.gz -C $CURDIR/output idl_cleaner

    echo "SCMVersion=$BUILD_VERSION" > version.txt
    echo "BuildUser=$BUILD_USER" >> version.txt
    echo "Branch=$BUILD_BRANCH" >> version.txt
    echo "DetailURL=https://cloud.bytedance.net/scm/detail/$BUILD_REPO_ID/versions?versionId=$BUILD_VERSION_ID" >> version.txt
    echo "CommitHash=$BUILD_BASE_COMMIT_HASH" >> version.txt
    echo "BuildTime=$(date +%Y-%m-%dT%H:%M:%S)" >> version.txt
    
    git add -A
    git commit -m "https://cloud.bytedance.net/scm/detail/$BUILD_REPO_ID/versions?versionId=$BUILD_VERSION_ID"
    git push -f origin idl_cleaner
fi
