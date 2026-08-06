package main

import (
	"fmt"
	"os"

	"code.byted.org/lihuanyu.0w0/gloop/internal/cli"
)

func main() {
	// Deprecated: `gloopd` 已合并到 `gloop start`。
	// 这里仍然保留独立入口二进制，避免升级后 $PATH 里直接写 gloopd 的脚本失效。
	// cli.Run 检测到 argv[0] == "gloopd" 时会打印 deprecation 警告，
	// 并把参数重写到 gloop start --no-detach --no-autostart。
	fmt.Fprintln(os.Stderr, "DEPRECATED: `gloopd` is deprecated. Use `gloop start` instead.")
	cli.MainEntry()
}
