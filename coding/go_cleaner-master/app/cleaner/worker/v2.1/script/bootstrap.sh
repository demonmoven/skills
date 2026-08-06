#! /usr/bin/env bash

set -ex

mkdir -p /root/log || true

echo "[$(date +%Y-%m-%dT%H:%M:%S)]enter bootstrap" >> /root/log/bootstrap.log

CURDIR=$(cd $(dirname "$0"); pwd)

export GDP_ROOT=${CURDIR}
export GDP_LOG_ROOT=${CURDIR}/log

echo "[$(date +%Y-%m-%dT%H:%M:%S)]start to install git" >> /root/log/bootstrap.log

# 升级git至较新版本以支持“--ignore-revs-files”等选项
GITVERSION=2.45.0
apt update && apt install -y make libssl-dev libcurl4-openssl-dev zlib1g-dev gettext time openssh-server
if [ -d "/root/project/deps/git" ]; then
    echo "use git in /root/project/deps/git"
    cd /root/project/deps/git
else
    echo "download git"
    cd /tmp
    wget https://github.com/git/git/archive/refs/tags/v${GITVERSION}.tar.gz && tar -xzf v${GITVERSION}.tar.gz
    cd /tmp/git-${GITVERSION}
fi
make CFLAGS="-std=c99" prefix=/usr/local all -j8 && make CFLAGS="-std=c99" prefix=/usr/local install

echo "[$(date +%Y-%m-%dT%H:%M:%S)]finish install git" >> /root/log/bootstrap.log

cd ${CURDIR}

CLEANER_GO112_GO122=master
CLEANER_GO123_PLUS=release-branch.go1.23

# FIXME: 镜像调试不涉及go1.23，如果涉及则需要对DEFAULT_CLEANER_VERSION进行改动
DEFAULT_CLEANER_VERSION=$CLEANER_GO112_GO122
if [ "X${CLEANER_REVISION}" != "X" ]; then
    DEFAULT_CLEANER_VERSION=$CLEANER_REVISION
fi

if [ "X${IDL_CLEANER_REVISION}" = "X" ]; then
    IDL_CLEANER_REVISION=master
fi


mkdir -p ${GOBIN}

# 安装 goimports
GO_VERSION=$(go version | awk '{print $3}')
case $GO_VERSION in
    go1.18*)
        GOIMPORTS_VERSION="v0.18.0"
        ;;
    go1.19*)
        GOIMPORTS_VERSION="v0.24.0"
        ;;
    go1.20*)
        GOIMPORTS_VERSION="v0.24.0"
        ;;
    go1.21*)
        # 与SCM镜像中的GOTOOLCHAIN设置保持一致
        go env -w GOTOOLCHAIN=local
        GOIMPORTS_VERSION="v0.24.0"
        ;;
    go1.22*)
        go env -w GOTOOLCHAIN=local
        GOIMPORTS_VERSION="v0.26.0"
        ;;
    go1.23*)
        go env -w GOTOOLCHAIN=local
        GOIMPORTS_VERSION="v0.26.0"
        DEFAULT_CLEANER_VERSION=$CLEANER_GO123_PLUS
        ;;
    *)
        echo "Unsupported Go version: $GO_VERSION"
        exit 1
        ;;
esac

# tango镜像需要将GOBIN添加到PATH中
export PATH=$(go env GOBIN):$PATH
echo "[$(date +%Y-%m-%dT%H:%M:%S)]start install goimports@${GOIMPORTS_VERSION}" >> /root/log/bootstrap.log
go install "golang.org/x/tools/cmd/goimports@$GOIMPORTS_VERSION"
echo "Installed goimports version $GOIMPORTS_VERSION for Go version $GO_VERSION"
echo "[$(date +%Y-%m-%dT%H:%M:%S)]start install cleaner@${DEFAULT_CLEANER_VERSION}" >> /root/log/bootstrap.log
go install code.byted.org/analyzers/go_cleaner/cmd/cleaner@"${DEFAULT_CLEANER_VERSION}" || { echo "install cleaner failed"; exit 1; }
echo "Installed go cleaner branch $DEFAULT_CLEANER_VERSION for Go version $GO_VERSION"
echo "[$(date +%Y-%m-%dT%H:%M:%S)]finish install cleaner" >> /root/log/bootstrap.log
if [ "$FOR_CLEANER_TEST" != "1" ]; then
    echo "[$(date +%Y-%m-%dT%H:%M:%S)]start run worker FOR_CLEANER_TEST:${FOR_CLEANER_TEST}" >> /root/log/bootstrap.log
    ./bin/worker &
else
    echo "[$(date +%Y-%m-%dT%H:%M:%S)]skip run worker FOR_CLEANER_TEST:${FOR_CLEANER_TEST}" >> /root/log/bootstrap.log
fi

tail -f /dev/null