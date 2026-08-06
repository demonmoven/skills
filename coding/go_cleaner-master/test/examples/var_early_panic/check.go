// RUN: %cleaner -f --passes=clean-after-return
// RUN: cat %s | %filecheck %s
package main

var x, y int

func main() {
	x = 1
	y = 2
	panic("error")
	_ = x + y
}

// CHECK: func main() {
// CHECK-NEXT: 	x = 1
// CHECK-NEXT: 	y = 2
// CHECK-NEXT: 	panic("error")
// CHECK-EMPTY:
// CHECK-NEXT: }
