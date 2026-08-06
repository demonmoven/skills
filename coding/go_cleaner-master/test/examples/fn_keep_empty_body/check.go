// RUN: %cleaner -f 
// RUN: cat %s | %filecheck %s
package main

import "fmt"

func foo() {
	return
}

func main() {
	fmt.Println("hello world")
	foo()
}

// CHECK: func foo() {
// CHECK-NEXT: 	return
// CHECK-NEXT: }
// CHECK-EMPTY:
// CHECK-NEXT: func main() {
// CHECK-NEXT: 	fmt.Println("hello world")
// CHECK-NEXT: 	foo()
// CHECK-NEXT: }
