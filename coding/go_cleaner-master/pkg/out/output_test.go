package out

import (
	"fmt"
	"testing"
)

func TestOutput(t *testing.T) {
	out, err := []byte{}, []byte{}
	o := NewBytesOutput(&out, &err)
	o.SetWarn("it is a warning")
	o.SetErrorf("it is a err")
	fmt.Println(string(out))
	fmt.Println(string(err))
}
