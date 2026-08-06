// RUN: %cleaner -f
// RUN: cat %s | %filecheck %s
package main

import "testing"

func TestFoo(t *testing.T) {
	Foo()
}

func TestCallback(t *testing.T) {
	SetCallBack(UnusedFn)
	Callback(cb)
}

func TestBar(t *testing.T) {
	Bar()
}

// CHECK: func TestFoo(t *testing.T) {
// CHECK-NOT: func TestCallback(t *testing.T) {
// CHECK: func TestBar(t *testing.T) {