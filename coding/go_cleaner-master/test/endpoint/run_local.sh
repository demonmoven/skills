#!/bin/bash

CURDIR=$(cd $(dirname $0); pwd)

if [ "X$1" != "X" ]; then
    GoVersionRange=1.19 go test -count 1 -timeout 10m -run $1
else
    GoVersionRange=1.19 go test -count 1 -timeout 10m .
fi
