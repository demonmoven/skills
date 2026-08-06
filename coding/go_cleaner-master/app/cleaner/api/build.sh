#!/bin/bash

RUN_NAME="tiktok.ci.unused_code_api"

subDir=app/cleaner/api

mkdir -p output/bin output/conf output/static
cp -r ${subDir}/static output
cp ${subDir}/script/bootstrap.sh output 2>/dev/null
chmod +x output/bootstrap.sh
cp ${subDir}/script/bootstrap.sh output/bootstrap_staging.sh
chmod +x output/bootstrap_staging.sh
find ${subDir}/conf/ -type f ! -name "*_local.*" | xargs -I{} cp {} output/conf/

go build -o "output/bin/${RUN_NAME}" ./${subDir}
