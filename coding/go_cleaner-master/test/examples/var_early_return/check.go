// RUN: %cleaner -f --passes=clean-after-return
// RUN: cat %s | %filecheck %s
package main

var x, y int

func main() {
	x = 1
	y = 2
	return
	_ = x + y
}

// CHECK: func main() {
// CHECK-NEXT: 	x = 1
// CHECK-NEXT: 	y = 2
// CHECK-NEXT: 	return
// CHECK-EMPTY:
// CHECK-NEXT: }
