// RUN: pushd $(dirname $(pwd))
// RUN: %cleaner -f
// RUN: cat %s | %filecheck %s
package _ignored_pkg

import (
	"example/pkg"
)

func UsedByMain() int {
	pkg.UsedByIgnoredDir()
	return 0
}

// CHECK: "example/pkg"
