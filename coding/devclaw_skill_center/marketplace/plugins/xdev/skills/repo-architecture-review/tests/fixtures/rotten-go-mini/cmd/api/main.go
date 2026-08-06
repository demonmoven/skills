package main

import (
	"example.com/rotten/internal/domain"
	"fmt"
)

func main() {
	user := domain.GetUser(1)
	fmt.Println(user)
}
