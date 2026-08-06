package main

import (
	"log"

	"code.byted.org/webcast/endpoint_kitex_specify_handler/impl"
	specify_handler "code.byted.org/webcast/endpoint_kitex_specify_handler/kitex_gen/endpoint/kitex/specify_handler/myservice"
)

func main() {
	svr := specify_handler.NewServer(new(impl.MyServiceImpl))

	err := svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
