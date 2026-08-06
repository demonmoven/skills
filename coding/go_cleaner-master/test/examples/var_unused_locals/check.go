// RUN: %cleaner -f --passes=clean-unused-decl
// RUN: cat %s | %filecheck %s
package main

var x int

func main() {
	x = 1
	y := 2
	return
}

// CHECK: func main() {
// CHECK-NEXT: 	x = 1
// CHECK-EMPTY:
// CHECK-NEXT: 	return
// CHECK-NEXT: }
