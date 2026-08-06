package ipkg1

func IPkg1Func() string {
	return "ipkg1"
}

func IPkg1Func2() string {
	return "ipkg1-2"
}

// CHECK: package ipkg1
// CHECK-EMPTY:
// CHECK-NEXT: func IPkg1Func2() string {
