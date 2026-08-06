// RUN: %cleaner -f --passes=clean-unused-import
// RUN: cat %s | %filecheck %s
package main

import (
	"fmt"
    "math"
)

func main() {
    fmt.Println("Hello, World!")
}

// CHECK: "fmt"
// CHECK-NOT: 	"math"
// CHECK: func main() {
