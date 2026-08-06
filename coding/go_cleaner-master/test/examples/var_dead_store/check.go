// RUN: %cleaner -f 
// RUN: cat %s | %filecheck %s
package main

var x = 1

func foo() {
	x = 2
	return
}

func main() {
	foo()
}

// CHECK: package main
// CHECK-EMPTY:
// CHECK-NEXT: func foo() {
// CHECK-NEXT:   return
// CHECK-NEXT: }
// CHECK-EMPTY:
// CHECK-NEXT: func main() {
// CHECK-NEXT:   foo()
// CHECK-NEXT: }
