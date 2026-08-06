#!/usr/bin/env bash

subDir=ci/go_unused_tool

mkdir -p output
go build -o output/go_unused_code ./${subDir}

