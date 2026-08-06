// RUN: %cleaner -f 
// RUN: cat %s | %filecheck %s
package main

import "fmt"

func fakeAdd(a, b int) int {
	return a
}

func main() {
	a := 1
	b := 2
	fmt.Println(fakeAdd(a, b))
	return
}

// CHECK: func fakeAdd(a int) int {
// CHECK-NEXT: 	return a
// CHECK-NEXT: }
// CHECK-EMPTY:
// CHECK-NEXT: func main() {
// CHECK-NEXT: 	a := 1
// CHECK-NEXT: 	fmt.Println(fakeAdd(a))
// CHECK-NEXT: 	return
// CHECK-NEXT: }
