#!/bin/bash

set -x

CURDIR=$(cd $(dirname $0); pwd)

go version

WORK_DIR=$CURDIR/cleaner-ep-test-$(tr -dc A-Za-z0-9 </dev/urandom | head -c 5)
echo "WORK_DIR: $WORK_DIR"
mkdir -p $WORK_DIR/bin

cd $CURDIR/../../app/cleaner/worker/v2.1
echo "build worker start..."
bash build.sh
mv output $WORK_DIR/worker
echo "build worker done"

cd $CURDIR/../../cmd/cleaner
echo "build cleaner start..."
go build -o $WORK_DIR/bin/cleaner
echo "build cleaner done"

cd $WORK_DIR
PATH=$WORK_DIR/bin:$WORK_DIR/worker/bin:$PATH GoVersionRange=1.18,1.19,1.20 $WORK_DIR/worker/script/clean_endpoint_script.sh "$@"
if [ "X$NOCLEAN" != "X1" ]; then
    rm -rf $WORK_DIR
fi