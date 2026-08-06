// RUN: %cleaner -f 
// RUN: cat %s | %filecheck %s
package main

var y = foo()

func foo() int {
	return 3
}

func bar() int {
	return 2
}

func main() {
	bar()
}

// CHECK: func foo() int {
