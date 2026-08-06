// RUN: %cleaner -f 
// RUN: cat %s | %filecheck %s
package main

import (
	"fmt"
	"reflect"
)

func Add(a, b int) int {
	return a + b
}

func Sub(a, b int) int {
	return a - b
}

func main() {
	pkgType := reflect.TypeOf(main)
	addFunc, _ := pkgType.MethodByName("Add")

	args := make([]reflect.Value, 2)
	args = append(args, reflect.ValueOf(1))
	args = append(args, reflect.ValueOf(2))
	fmt.Println(addFunc.Func.Call(args))
}

// CHECK: func Add(a, b int) int {
// CHECK-NEXT: 	return a + b
// CHECK-NEXT: }
// CHECK-EMPTY:
// CHECK-NEXT: func main() {
