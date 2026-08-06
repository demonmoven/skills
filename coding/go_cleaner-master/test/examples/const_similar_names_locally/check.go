// RUN: %cleaner -f
// RUN: cat %s | filecheck %s
package main

import "fmt"

func main() {
	var x string
	go func() {
		const (
			SimpleTypeRead  = "read"
			SimpleTypeWrite = "write"
		)
		x = SimpleTypeWrite + SimpleTypeWrite
		x = x + x
	}()
	fmt.Print(x)
}


// CHECK:			SimpleTypeRead  = "read"
// CHECK-NEXT:		SimpleTypeWrite = "write"
