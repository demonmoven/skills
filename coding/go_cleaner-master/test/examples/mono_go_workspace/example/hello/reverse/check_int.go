// RUN: pushd $(dirname $(dirname $(pwd)))
// RUN: %cleaner -f
// RUN: cat %s | filecheck %s
package reverse

import "strconv"

// Int returns the decimal reversal of the integer i.
func UnusedInt(i int) int {
	i, _ = strconv.Atoi(String(strconv.Itoa(i)))
	return i + 1
}

// Int returns the decimal reversal of the integer i.
func Int(i int) int {
	i, _ = strconv.Atoi(String(strconv.Itoa(i)))
	return i
}

// CHECK: package reverse
// CHECK-NOT: func UnusedInt(i int) int {
// CHECK: func Int(i int) int {