// RUN: %cleaner -f --unsafe_delete_method
// RUN: cat %s | %filecheck %s
package main

type Foo struct {
	a int
	b int
}

func (f *Foo) add() int {
	return f.a + f.b
}

func (f *Foo) sub() int {
	return f.a - f.b
}

func (f *Foo) mul() int {
	return f.a * f.b
}

func main() {
	f := &Foo{
		a: 1,
		b: 2,
	}
	f.add()
	x := f.mul
	x()
}

// CHECK: func (f *Foo) add() int {
// CHECK-NOT: func (f *Foo) sub() int {
// CHECK: func (f *Foo) mul() int {
// CHECK: func main() {


/*
# Name: command-line-arguments.main
# Package: command-line-arguments
# Location: /home/yuanting.k2/go_cleaner/test/examples/type_method_deletion/check.go:22:6
func main():
0:                                                                entry P:0 S:0
        t0 = new Foo (complit)                                             *Foo
        t1 = &t0.a [#0]                                                    *int
        t2 = &t0.b [#1]                                                    *int
        *t1 = 1:int
        *t2 = 2:int
        t3 = (*Foo).add(t0)                                                 int
        t4 = make closure (*Foo).mul$bound [t0]                      func() int
        t5 = t4()                                                           int
        return

# Name: (*command-line-arguments.Foo).mul$bound
# Synthetic: bound method wrapper for func (*command-line-arguments.Foo).mul() int
# Location: /home/yuanting.k2/go_cleaner/test/examples/type_method_deletion/check.go:18:15
# Free variables:
#   0:  recv *command-line-arguments.Foo
func mul$bound() int:
0:                                                                entry P:0 S:0
        t0 = (*command-line-arguments.Foo).mul(recv)                        int
        return t0
*/
