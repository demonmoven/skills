#!/bin/bash

CURDIR=$(cd $(dirname $0); pwd)

differ --base master --target feat/autoTest2 --case-file $CURDIR/endpoint-case1.json --type endpoint --root $CURDIR/../.. --parallel 1
