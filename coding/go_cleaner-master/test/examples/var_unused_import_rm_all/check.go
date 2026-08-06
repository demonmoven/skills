// RUN: %cleaner -f --passes=clean-unused-import
// RUN: cat %s | %filecheck %s
package main

import (
    "math"
)

var x int

func main() {
    x = 1
}

// CHECK: package main
// CHECK-NOT: 	import (
// CHECK: func main() {
