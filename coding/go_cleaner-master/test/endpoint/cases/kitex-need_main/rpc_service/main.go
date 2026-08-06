package main

import (
	need_main "code.byted.org/webcast/endpoint_kitex_need_main/kitex_gen/endpoint/kitex/need_main/myservice"
	"log"
)

func main() {
	svr := need_main.NewServer(new(MyServiceImpl))

	err := svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
