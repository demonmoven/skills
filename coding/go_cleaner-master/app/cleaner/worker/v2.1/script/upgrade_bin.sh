#! /usr/bin/env bash

if [ -z "$1" ]; then
    echo "Usage: $0 <version>"
    exit 1
fi
version=$1
go install code.byted.org/analyzers/go_cleaner/cmd/idl_cleaner@"${version}"

go install code.byted.org/analyzers/go_cleaner/cmd/cleaner@"${version}"