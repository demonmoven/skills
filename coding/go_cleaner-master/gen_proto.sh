#!/usr/bin/env bash

set -x -e

CURDIR=$(cd $(dirname $0); pwd)

PROTO_GEN_GO=$(which protoc-gen-go)

protoc --plugin=$PROTO_GEN_GO --proto_path=$CURDIR --go_out=$CURDIR pkg/idl_cleaner/proto/plugin/byteapi.proto