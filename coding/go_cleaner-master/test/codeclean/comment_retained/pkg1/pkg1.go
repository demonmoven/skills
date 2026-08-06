package pkg1

const (
	A, B string = "a", "b" // retained

	C = 1

	D = "string" // retained
)

var (
	Var1, Var2 string

	Var3 = func() {
		a := "a"
		_ = a
	}

	Var4 string
)

func RetainedFn1() {

}

func UnusedFn1() {

}

func UsedFn1() {

}
