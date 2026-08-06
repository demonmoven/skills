// RUN: %cleaner -f --passes=clean-empty-files
package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello, World!")
}
