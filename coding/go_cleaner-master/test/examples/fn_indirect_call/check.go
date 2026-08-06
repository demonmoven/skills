// RUN: %cleaner -f 
// RUN: cat %s | %filecheck %s
package main

import "fmt"

func bar() {
	fmt.Println("I'm bar")
}

func foo() int {
	_ = bar
	return 0
}

func main() {
	call := foo
	call()
}

// CHECK: package main
// CHECK-EMPTY:
// CHECK-NEXT: func foo() int {
// CHECK-NEXT: return 0
// CHECK-NEXT: }
// CHECK-EMPTY:
// CHECK-NEXT: func main() {
// CHECK-NEXT: call := foo
// CHECK-NEXT: call()
// CHECK-NEXT: }
