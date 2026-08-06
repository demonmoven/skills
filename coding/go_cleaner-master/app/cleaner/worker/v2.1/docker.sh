#!/bin/bash

set -ex

# 拉取idl_cleaner
IDL_CLEANER_TMP_DIR=/tmp/idl_cleaner_$(openssl rand -hex 6)
mkdir $IDL_CLEANER_TMP_DIR
git clone -b idl_cleaner https://oauth2:$CUSTOM_BAG_TOOL_OAUTH2@code.byted.org/webcast/bag_tools.git --depth=1 --single-branch $IDL_CLEANER_TMP_DIR
cd $IDL_CLEANER_TMP_DIR && tar -zxvf idl_cleaner.tar.gz && mv idl_cleaner ~/.tools/ && cd -
rm -rf $IDL_CLEANER_TMP_DIR || true

HUB_DOMAIN='hub.byted.org'
if [[ "$I18NWORKER" = "1" ]]; then
  HUB_DOMAIN='aliyun-sin-hub.byted.org'
fi

TMP_DOCKER_DIR=/tmp/docker_$(openssl rand -hex 6)
# 每个版本启动2个docker实例
for v in 1.18 1.19 1.20 1.21 1.22 1.23 1.24; do
      echo "start build docker for go${v}"
      rm -rf $TMP_DOCKER_DIR || true
      mkdir -p $TMP_DOCKER_DIR
      ls . | grep -v docker | grep -v output | xargs -I {} cp -r ./{} $TMP_DOCKER_DIR
      ls docker/${v} | xargs -I {} cp -r docker/${v}/{} $TMP_DOCKER_DIR
      cd $TMP_DOCKER_DIR
      docker build -t worker_v2.1_go${v}_v${1} --build-arg HUB_DOMAIN=$HUB_DOMAIN .
      for loop in $(seq 1 2) ; do
        name="${v}_${loop}"
        echo "start docker for go${v} name:${name}"
        mkdir -p /data00/go
        docker run --rm --network host -d --privileged \
        -e "RUNTIME_IDC_NAME=${2}" -e "BYTED_HOST_IPV6=$BYTED_HOST_IPV6" \
        -e "FOR_CLEANER_TEST=$FOR_CLEANER_TEST" \
        -e "BUILD_BRANCH=$BUILD_BRANCH" \
        -e "BUILD_COMMIT=$BUILD_COMMIT" \
        -e "TOKEN_CI_IESOPS=$CUSTOM_BAG_TOOL_OAUTH2" \
        -e "WORKER_ENV=$WORKER_ENV" \
        -e "I18NWORKER=$I18NWORKER" \
        -e "ARK_API_KEY=$ARK_API_KEY" \
        -e "GOBIN=/opt/tiger/compile_path/bin/${v}" \
        -v /home/tiger/gitconfig/tiktok/.gitconfig:/root/.gitconfig \
        -v /home/tiger/gitconfig/git_ssh:/root/.ssh \
        -v /data00/go:/opt/tiger/compile_path \
        -v ~/.tools:/root/tools \
        --name worker_v2.1_go${name}_v${1}_container_01 worker_v2.1_go${v}_v${1}
      done
      cd -
done

rm -rf $TMP_DOCKER_DIR || true
