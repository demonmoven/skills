#! /usr/bin/env bash

set -ex

CURDIR=$(cd $(dirname "$0"); pwd)

export GDP_ROOT=${CURDIR}
export GDP_LOG_ROOT=${CURDIR}/log

apt update && apt install -y time

if [ "X${CXX_CLEANER_REVISION}" = "X" ]; then
    CXX_CLEANER_REVISION=main
fi

cd "$(dirname $WORKSPACE)"
git clone git@code.byted.org:webcast/sopt_cxx_cleaner.git
cd sopt_cxx_cleaner/
git checkout $CXX_CLEANER_REVISION

cd "$WORKSPACE"
blade init

cd $(dirname $CURDIR)/output
if [ -z "$FOR_CLEANER_TEST" ]; then
  ./bin/worker &
fi
tail -f /dev/null