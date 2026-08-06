package out

import (
	"fmt"
	"testing"
)

func TestExtract(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		out := Extract([]byte(`adadsada

::set-output::key1::value1

dasdadsa

::set-output::key2::value2\++
value3

::set-warn::key3::value4

::set-error::key4::value5`))

		if len(out.KVs) != 2 {
			t.Errorf("Expected 2 KVs, got %d", len(out.KVs))
		}
		if out.KVs["key1"] != "value1" {
			t.Errorf("Expected key1 to be value1, got %s", out.KVs["key1"])
		}
		if out.KVs["key2"] != "value2\nvalue3" {
			t.Errorf("Expected key2 to be value2\\nvalue3, got %s", out.KVs["key2"])
		}
		if len(out.Logs) != 2 {
			t.Errorf("Expected 2 logs, got %d", len(out.Logs))
		}
		if out.Logs[0].Level != "warn" {
			t.Errorf("Expected log level to be warn, got %s", out.Logs[0].Level)
		}
		if out.Logs[0].Message != "key3::value4" {
			t.Errorf("Expected log message to be key3::value4, got %s", out.Logs[0].Message)
		}
		if out.Logs[1].Level != "error" {
			t.Errorf("Expected log level to be error, got %s", out.Logs[1].Level)
		}
		if out.Logs[1].Message != "key4::value5" {
			t.Errorf("Expected log message to be key4::value5, got %s", out.Logs[1].Message)
		}
	})
}
func TestOutput(t *testing.T) {
	out, err := []byte{}, []byte{}
	o := NewBytesOutput(&out, &err)
	o.SetWarn("it is a warning")
	o.SetErrorf("it is a err")
	fmt.Println(string(out))
	fmt.Println(string(err))
}
