package main

var x = 1
var cb func() int

func Bar() int {
	return x
}

func Foo() int {
	x += 1
	fn := Bar
	return fn()
}

func Callback(fn func() int) int {
	return Foo() + fn()
}

func SetCallBack(fn func() int) {
	cb = fn
}

func UnusedFn() int {
	Bar()
	return -1
}

func main() {
	Foo()
}