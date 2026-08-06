#!/usr/bin/env bash

set -x
UNAME=${1}
GO_VERSION=${2}
IDC=${3}
GOROOT_DIR=${4}
echo "UNAME:${UNAME} GoVersion:${GO_VERSION} IDC:${IDC} GOTOOL_DIR:${GOROOT_DIR}"

rm -rf ~/_tmp_docker_test || true
mkdir -p ~/_tmp_docker_test
ls . | grep -v docker | grep -v output | xargs -I {} cp -r ./{} ~/_tmp_docker_test/
ls docker/$GO_VERSION | xargs -I {} cp -r docker/$GO_VERSION/{} ~/_tmp_docker_test/
cd ~/_tmp_docker_test
docker build -t ${UNAME}_go${GO_VERSION} .

docker run --rm --network host -d \
        -e "RUNTIME_IDC_NAME=$IDC" -e "BYTED_HOST_IPV6=$BYTED_HOST_IPV6" \
        -e "FOR_CLEANER_TEST=1" \
        -e "GOBIN=/go/bin/$GO_VERSION" \
        -e "GOMODCACHE=/mod_cache" \
        -v $HOME/gitconfig/tiktok/.gitconfig:/root/.gitconfig \
        -v $HOME/gitconfig/git_ssh:/root/.ssh \
        -v $GOROOT_DIR:/go \
        --name ${UNAME}_go${GO_VERSION}_container ${UNAME}_go${GO_VERSION}