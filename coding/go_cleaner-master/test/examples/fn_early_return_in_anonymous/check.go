// RUN: %cleaner -f
// RUN: cat %s | %filecheck %s
package main

func main() {
	var start int
	counter := func(x int) int {
		return x
		if (x % 2) == 0 {
			return x*start
		}
		return x+1
	}

	counter(counter(1))
}

// CHECK: counter := func(x int) int {
// CHECK-NEXT:     return x
// CHECK-EMPTY:     
// CHECK-NEXT: }