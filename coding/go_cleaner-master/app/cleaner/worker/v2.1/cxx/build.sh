#!/usr/bin/env bash

RUN_NAME="worker"

mkdir -p output/bin
mkdir -p output/script
mkdir -p output/codebase

cp cxx/bootstrap.sh output/
cp cxx/clean_cxx_script.sh output/script/
chmod +x output/bootstrap.sh
chmod +x output/script/*

go build -o output/bin/${RUN_NAME}