// RUN: %cleaner -f 
// RUN: cat %s | %filecheck %s
package main

import "fmt"

const (
	SimpleTypeRead = "read"
	AlwaysStore    = "write"
)

func main() {
	x := AlwaysStore + AlwaysStore
	x = x + x
	fmt.Print(x)
}

// CHECK: const (
// CHECK-NOT: SimpleTypeRead = "read"
// CHECK: AlwaysStore = "write"
