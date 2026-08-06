#!/usr/bin/env bash

RUN_NAME="worker"

mkdir -p output/bin
mkdir -p output/script
mkdir -p output/script/custom_script
mkdir -p output/codebase

cp script/bootstrap.sh output/
chmod +x output/bootstrap.sh
ls script | grep -v bootstrap.sh | xargs -I {} cp script/{} output/script
chmod +x output/script/*

cp script/custom_script/* output/script/custom_script
chmod +x output/script/custom_script/*

set -ex

go build -o output/bin/${RUN_NAME}