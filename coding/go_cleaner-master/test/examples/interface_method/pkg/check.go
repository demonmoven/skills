// RUN: %cleaner -f
// RUN: cat %s | %filecheck %s
package pkg

type T interface {
	Method()
}

type S1 struct {
	x int
}

func (s *S1) Method() {
	s.x = 1
}

func (s *S1) UsedFunc() {
	s.x = 2
}

type S2 struct {
	y int
}

func (s *S2) UnusedFunc() {
	s.y = 1
}

func (s *S2) UsedFunc() {
	s.y = 2
}

// CHECK: func (s *S1) Method() {
// CHECK-NOT: func (s *S2) UnusedFunc() {
// CHECK: func (s *S2) UsedFunc() {