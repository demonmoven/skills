#!/bin/bash

set -ex

if [ -z "$1" ]; then
    echo "Error: No branch specified." >&2
    exit 110
fi
export IDL_CLEANER_REVISION=$1
export CLEANER_REVISION=$1
RAND_BRANCH=tmp_$(openssl rand -hex 4)

docker exec default bash -c "cd /root/share/codebase/go_cleaner/ && git branch $RAND_BRANCH && git checkout -f $RAND_BRANCH && git branch -D $1 &>/dev/null && git fetch origin $1:$1 && git checkout $1 && git branch -D $RAND_BRANCH"
CLEANER_COMMIT=$(docker exec default bash -c "cd /root/share/codebase/go_cleaner/ && git rev-parse HEAD")

echo ""
echo "cleaner has updated to commit: $CLEANER_COMMIT"
echo "start kill current workers..."

cd ${HOME}/share/codebase/go_cleaner/app/cleaner/worker/v2.1
export BUILD_BRANCH=$(git rev-parse --abbrev-ref HEAD)
export BUILD_COMMIT=$(git rev-parse HEAD)

docker_list=$(docker ps -q --filter "name=worker_v2.1_llvm")
if [ -n "${docker_list}" ];then
    docker stop ${docker_list}
    sleep 30
fi

echo "start start new llvm workers..."

sh ./cxx/docker.sh 0.0.1 lf