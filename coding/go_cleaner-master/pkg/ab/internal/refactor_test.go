package internal

import (
	"os"
	"testing"
)

func TestName(t *testing.T) {
	os.Chdir("/Users/bytedance/go/src/code.byted.org/xiaoxing.sn/abtest_clean_demo")
	newUnusedDeclCleaner().clean()
}
