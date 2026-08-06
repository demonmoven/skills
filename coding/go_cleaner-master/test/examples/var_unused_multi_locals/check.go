// RUN: %cleaner -f --passes=clean-unused-decl
// RUN: cat %s | %filecheck %s
package main

var x int

func main() {
	var (
		y int
		z int
	)
	x = 1
	return
}

// CHECK: func main() {
// CHECK-EMPTY:
// CHECK-NEXT: 	x = 1
// CHECK-NEXT: 	return
// CHECK-NEXT: }
