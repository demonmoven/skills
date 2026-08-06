// RUN: %cleaner -f --passes=clean-after-return
// RUN: cat %s | %filecheck %s
package main

var x, y int

func main() {
	return
	x = 1
	y = 2
	if (x+y)&x > 0 {
		return
	}
	_ = x + y
	// Some comments here
	// Other comments here
}

// CHECK: func main() {
// CHECK-NEXT: 	return
// CHECK-EMPTY:
// CHECK-NEXT: }
