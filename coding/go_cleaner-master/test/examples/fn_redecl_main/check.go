// RUN: %cleaner -f --passes=rename-redecl-mains
package main

import "fmt"

func foo() int {
	return 1
}

func bar() int {
	return 2
}

func baz() int {
	return 3
}

func main() {
	fmt.Println(foo())
}

// CHECK: func foo() int {
// CHECK: func bar() int {
// CHECK-NOT: func baz() int {
// CHECK: func main() {