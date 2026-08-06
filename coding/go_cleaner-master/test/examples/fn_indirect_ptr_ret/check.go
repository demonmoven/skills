// RUN: %cleaner -f
// RUN: cat %s | filecheck %s
package main

import "fmt"

func bar() {
	fmt.Println("I'm bar")
}

func foo() func() {
	return bar
}

func main() {
	call := foo()
	call()
}

// CHECK: func bar() {
// CHECK: func foo() func() {
// CHECK: func main() {
