// RUN: %cleaner -f 
// RUN: cat %s | %filecheck %s
package main

var x = 1

func foo() {
	x = 2
	return
}

func bar() {
	foo()
}

func main() {
	foo()
}

// CHECK: func foo() {
// CHECK-NEXT: 	x = 2
// CHECK-NEXT: 	return
// CHECK-NEXT: }
// CHECK-EMPTY:
// CHECK-NEXT: func main() {
// CHECK-NEXT: 	foo()
// CHECK-NEXT: }
