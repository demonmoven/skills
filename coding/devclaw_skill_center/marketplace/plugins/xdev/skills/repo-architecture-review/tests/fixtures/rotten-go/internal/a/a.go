package a

import "example.com/rotten/internal/b"

func A() string {
	return b.B()
}
