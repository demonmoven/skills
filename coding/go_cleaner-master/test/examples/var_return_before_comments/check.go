// RUN: %cleaner -f --passes=clean-after-return
// RUN: cat %s | %filecheck %s
package main

var x, y int

func main() {
	x = 1
	y = 2
	if (x+y)&x > 0 {
		return
	}
	return
	_ = x + y
	// Some comments here
	// Other comments here
}

// CHECK: func main() {
// CHECK-NEXT: 	x = 1
// CHECK-NEXT: 	y = 2
// CHECK-NEXT: 	if (x+y)&x > 0 {
// CHECK-NEXT: 		return
// CHECK-NEXT: 	}
// CHECK-NEXT: 	return
// CHECK-EMPTY:
// CHECK-NEXT: }
