// RUN: %cleaner -f 
// RUN: cat %s | %filecheck %s
package main

import "fmt"

const (
	SimpleTypeRead  = "read"
	SimpleTypeWrite = "write"
	ExceedTradeThreshold   = "0"
	NoExceedTradeThreshold = "1"
)

func main() {
	x := SimpleTypeWrite + SimpleTypeWrite
	x = x + x
	fmt.Print(x)
}

// CHECK: 	SimpleTypeRead  = "read"
// CHECK-NEXT: SimpleTypeWrite = "write"
// CHECK-NEXT: )
