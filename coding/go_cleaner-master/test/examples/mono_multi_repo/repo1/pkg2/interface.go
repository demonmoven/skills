package pkg2

type Interface interface {
	Do() int
}

func JustDoIt(i Interface) int {
	return i.Do()
}