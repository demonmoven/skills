package pkg1

func B() {
	println("b")
}

func A() {
	println("a")
}

// CHECK: package pkg1
// CHECK-NOT: func B() {
// CHECK: func A() {