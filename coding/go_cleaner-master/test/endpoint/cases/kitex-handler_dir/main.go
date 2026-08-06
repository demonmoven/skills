package main

import (
	"log"

	"code.byted.org/webcast/endpoint_kitex_handler_dir/handler"
	handler_dir "code.byted.org/webcast/endpoint_kitex_handler_dir/kitex_gen/endpoint/kitex/handler_dir/myservice"
)

func main() {
	svr := handler_dir.NewServer(new(handler.MyServiceImpl))

	err := svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
