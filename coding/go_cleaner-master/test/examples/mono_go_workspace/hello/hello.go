package main

import (
	"fmt"

	"code.byted.org/webcast/sopt_xexample/hello/reverse"
)

func main() {
	fmt.Println(reverse.String("Hello"), reverse.Int(24601))
}
