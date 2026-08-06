#!/bin/bash

set -e

if [ "X$2" = "X" ]; then
    echo "usage: $0 <user_name: for isolation> <branch>"
    exit 1
fi

RAND_BRANCH=tmp_$(openssl rand -hex 6)

CLEANER_DIR=~/$1/go_cleaner/
rm -rf $CLEANER_DIR || true
mkdir -p $CLEANER_DIR
git clone -b $2 https://oauth2:$CUSTOM_BAG_TOOL_OAUTH2@code.byted.org/analyzers/go_cleaner --depth=1 --single-branch $CLEANER_DIR
cd $CLEANER_DIR
CLEANER_COMMIT=$(git rev-parse HEAD)


echo ""
echo "cleaner has updated to commit: $CLEANER_COMMIT"

cd $CLEANER_DIR/app/cleaner/worker/v2.1
export BUILD_BRANCH=$(git rev-parse --abbrev-ref HEAD)
export BUILD_COMMIT=$(git rev-parse HEAD)

echo "kill current go workers..."
docker_list=$(docker ps -q --filter "name=worker_v2.1_go1")
if [ -n "${docker_list}" ];then
    docker stop ${docker_list}
    sleep 30
fi
bash docker.sh 0.0.1 lf

echo "start new go workers..."
if [ "$I18NWORKER" = "1" ]; then
    echo "skip c++-workers for i18n..."
else
    echo "kill current c++ workers..."
    docker_list=$(docker ps -q --filter "name=worker_v2.1_llvm")
    if [ -n "${docker_list}" ];then
        docker stop ${docker_list}
        sleep 30
    fi
    echo "start new c++ workers..."
    bash ./cxx/docker.sh 0.0.1 if
fi
