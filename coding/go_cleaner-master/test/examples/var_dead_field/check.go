// RUN: %cleaner -f --passes=clean-unused-fields
// RUN: cat %s | %filecheck %s
package main

import "fmt"

type Foo struct {
	a int
	b int
}

func main() {
	f := &Foo{}
	f.a = 1
	fmt.Println(f.a)
	return
}

// CHECK: type Foo struct {
// CHECK-NEXT:         a int
// CHECK-NEXT: }
// CHECK-EMPTY: 
// CHECK-NEXT: func main() {
// CHECK-NEXT:         f := &Foo{}
// CHECK-NEXT:         f.a = 1
// CHECK-NEXT:         fmt.Println(f.a)
// CHECK-NEXT:         return
// CHECK-NEXT: }
