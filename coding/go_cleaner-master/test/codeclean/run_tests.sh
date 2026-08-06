#!/bin/bash

set -e

CURDIR=$(cd $(dirname $0); pwd)
TESTFLAGS=-v

if [ "X$VERBOSE_OFF" != "X" ]; then
    TESTFLAGS=
fi

if [ "X$1" != "X" ]; then
    GoVersionRange=1.19 go test -count 1 -timeout 10m $TESTFLAGS -run $1
else
    GoVersionRange=1.19 go test -count 1 -timeout 10m $TESTFLAGS .
fi
