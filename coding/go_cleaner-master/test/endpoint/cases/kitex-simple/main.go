package main

import (
	simple "code.byted.org/webcast/endpoint_kitex_simple/kitex_gen/endpoint/kitex/simple/myservice"
	"log"
)

func main() {
	svr := simple.NewServer(new(MyServiceImpl))

	err := svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
