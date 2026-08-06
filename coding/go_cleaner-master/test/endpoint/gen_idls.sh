#!/bin/bash

RED='\033[0;31m'
NC='\033[0m' # No Color

CURDIR=$(cd $(dirname $0); pwd)
print_red() {
  echo -e "${RED}$1${NC}"
}

echo "start regen for kitex-simple"
cd $CURDIR/cases/kitex-simple
rm -rf kitex_gen
kitex -disable-self-update -service endpoint.kitex.simple -module code.byted.org/webcast/endpoint_kitex_simple $CURDIR/idls/kitex-simple.thrift

echo "start regen for kitex-need_main"
cd $CURDIR/cases/kitex-need_main/rpc_service
rm -rf ../kitex_gen
kitex -disable-self-update -service endpoint.kitex.need_main -module code.byted.org/webcast/endpoint_kitex_need_main -gen-path ../kitex_gen $CURDIR/idls/kitex-need_main.thrift 

echo "start regen for kitex-handler_dir"
cd $CURDIR/cases/kitex-handler_dir
rm -rf kitex_gen
kitex -disable-self-update -service endpoint.kitex.handler_dir -module code.byted.org/webcast/endpoint_kitex_handler_dir $CURDIR/idls/kitex-handler_dir.thrift
mv handler.go handler/handler.new.go
echo -e "${RED}merge test/endpoint/cases/kitex-handler_dir/handler/handler.new.go into test/endpoint/cases/kitex-handler_dir/handler/service.go ${NC}"

echo "start regen for kitex-specify_handler"
cd $CURDIR/cases/kitex-specify_handler
rm -rf kitex_gen
mv handler.go impl/handler.new.go
kitex -disable-self-update -service endpoint.kitex.specify_handler -module code.byted.org/webcast/endpoint_kitex_specify_handler $CURDIR/idls/kitex-specify_handler.thrift
echo -e "${RED}merge test/endpoint/cases/kitex-specify_handler/impl/handler.new.go into test/endpoint/cases/kitex-specify_handler/impl/handler.go ${NC}"
