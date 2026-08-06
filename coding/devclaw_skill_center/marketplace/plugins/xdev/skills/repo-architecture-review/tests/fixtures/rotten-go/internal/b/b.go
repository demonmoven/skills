package b

import "example.com/rotten/internal/a"

func B() string {
	return a.A()
}
