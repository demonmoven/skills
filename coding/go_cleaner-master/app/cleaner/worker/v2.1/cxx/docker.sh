#!/usr/bin/env bash

set -ex

echo "start build docker for llvm"
docker build -t worker_v2.1_llvm_v${1} -f ./cxx/docker/Dockerfile .

#每个版本启动1个docker实例
for loop in $(seq 1 4) ; do
  echo "start container for llvm"
  docker run --rm --network host -d \
  --cap-add=SYS_PTRACE \
  --security-opt seccomp=unconfined \
  --privileged \
  -e "RUNTIME_IDC_NAME=${2}" -e "BYTED_HOST_IPV6=$BYTED_HOST_IPV6" \
  -e "FOR_CLEANER_TEST=$FOR_CLEANER_TEST" \
  -e "BUILD_BRANCH=$BUILD_BRANCH" \
  -e "BUILD_COMMIT=$BUILD_COMMIT" \
  -e "WORKSPACE=/opt/tiger/workspace" \
  -v /home/tiger/gitconfig/tiktok/.gitconfig:/root/.gitconfig \
  -v /home/tiger/gitconfig/git_ssh:/root/.ssh \
  -v /opt/tiger/typhoon-blade:/opt/tiger/typhoon-blade:ro \
  -v /opt/tiger/ss_bin:/opt/tiger/ss_bin:ro \
  -v /opt/tmp/consul_agent:/opt/tmp/consul_agent:ro \
  --name worker_v2.1_llvm_${loop}_v${1}_container_01 worker_v2.1_llvm_v${1}
done