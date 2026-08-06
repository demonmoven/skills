package pkg2

func B() {
	println("b")
}

func A() {
	println("a")
}

// CHECK: package pkg2
// CHECK-NOT: func B() {
// CHECK: func A() {