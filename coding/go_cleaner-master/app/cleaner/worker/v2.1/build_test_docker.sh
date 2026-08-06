#!/usr/bin/env bash

set -x
TEST_ID=${1}
GO_VERSION=${2}
CLEANER_VERSION=${3}
OUTPUT_DIR=${4}
IDC=${5}
GOROOT_DIR=${6}
echo "TestID:${TEST_ID} GoVersion:${GO_VERSION} CleanerVersion:${CLEANER_VERSION} OuputDir:${OUTPUT_DIR} IDC:${IDC} GOTOOL_DIR:${GOROOT_DIR}"

rm -rf ~/_tmp_docker_test || true
mkdir -p ~/_tmp_docker_test
ls . | grep -v docker | grep -v output | xargs -I {} cp -r ./{} ~/_tmp_docker_test/
ls docker/$GO_VERSION | xargs -I {} cp -r docker/$GO_VERSION/{} ~/_tmp_docker_test/
cd ~/_tmp_docker_test
docker build -t test_${TEST_ID}_go${GO_VERSION} .

docker run --rm --network host -d \
        -e "RUNTIME_IDC_NAME=$IDC" -e "BYTED_HOST_IPV6=$BYTED_HOST_IPV6" \
        -e "FOR_CLEANER_TEST=1" \
        -e "CLEANER_REVISION=$CLEANER_VERSION" \
        -e "GOBIN=/go/bin/$GO_VERSION" \
        -v $HOME/gitconfig/tiktok/.gitconfig:/root/.gitconfig \
        -v $HOME/gitconfig/git_ssh:/root/.ssh \
        -v $GOROOT_DIR:/go \
        -v ${OUTPUT_DIR}:/cleaner_tests \
        --name test_${TEST_ID}_go${GO_VERSION}_container test_${TEST_ID}_go${GO_VERSION}