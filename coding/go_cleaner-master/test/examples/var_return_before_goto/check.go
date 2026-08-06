// RUN: %cleaner -f --passes=clean-after-return,clean-unused-decl
// RUN: cat %s | %filecheck %s
package main

var x, y int

func main() {
	x = 1
	y = 2
	if (x+y)&x > 0 {
		goto Label
	}
	return
	_ = x + y
Label:
	y = x & y
}

// CHECK: func main() {
// CHECK-NEXT: 	x = 1
// CHECK-NEXT: 	y = 2
// CHECK-NEXT: 	if (x+y)&x > 0 {
// CHECK-NEXT: 		goto Label
// CHECK-NEXT: 	}
// CHECK-NEXT: 	return
// CHECK-EMPTY:
// CHECK-NEXT: Label:
// CHECK-NEXT: 	y = x & y
// CHECK-NEXT: }
