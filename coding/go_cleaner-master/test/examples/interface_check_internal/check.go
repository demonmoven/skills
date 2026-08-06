// RUN: %cleaner -f
// RUN: cat %s | %filecheck %s
package main

type T interface {
	Method()
}

type K struct {
	V int
}

func (k *K) Method() {}

var _ T = (*K)(nil)

func main() {}

// CHECK: type T interface {
// CHECK: type K struct {
