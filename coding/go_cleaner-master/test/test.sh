#!/bin/bash

set -e

CURDIR=$(dirname $0)
CURDIR=$(realpath $CURDIR)
TEST_DATA_DIRECTORY=$(python3 $CURDIR/util_lit/test-lit-setup.py)

pushd $TEST_DATA_DIRECTORY
$CURDIR/util_lit/test-lit-main.py $TEST_DATA_DIRECTORY -j1 -v $@
popd

pushd $CURDIR/codeclean
VERBOSE_OFF=1 ./run_tests.sh
popd

WORKER_DIR=$(git rev-parse --show-toplevel)/app/cleaner/worker/v2.1/pkg/scm
pushd $WORKER_DIR
GoVersionRange=1.19 go test -v -count 1 ./...
popd

WORKER_DIR=$(git rev-parse --show-toplevel)/pkg/endpoint/internal
pushd $WORKER_DIR
GoVersionRange=1.19 go test -v -count 1 ./...
popd