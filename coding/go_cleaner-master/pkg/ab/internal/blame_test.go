package internal

import (
	"os"
	"testing"
)

func TestBlame(t *testing.T) {
	os.Chdir("/Users/bytedance/go/src/code.byted.org/tiktok/pack_server")
	NewBlame()
}
