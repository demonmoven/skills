package cli

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

// ==================== notify 命令 ====================
//
// 给用户发送飞书通知。agent 在 quest 内也可以调用。
// 优先走 server HTTP API，server 没跑就 fallback 到本地 engine。

func runNotifyCmd(ctx context.Context, log *slog.Logger, args []string) int {
	fs := flag.NewFlagSet("notify", flag.ContinueOnError)
	title := fs.String("title", "", "通知标题，简短清晰（15字以内最佳）")
	body := fs.String("body", "", "通知正文，Markdown 格式，简明扼要说明情况和下一步动作")
	priority := fs.String("priority", "normal", "优先级：normal（普通）/ high（高优）")
	dataDir := fs.String("data-dir", "", "数据目录（默认 ~/.gloop）")
	jsonOut := fs.Bool("json", false, "JSON 结构化输出")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *title == "" {
		fmt.Fprintln(os.Stderr, "title 必填")
		return 2
	}

	dd := resolveDataDirOrDefault(*dataDir)

	// 优先走 server API
	if proxy := tryServerProxy(dd); proxy != nil {
		return runNotifyViaServer(proxy, *title, *body, *priority, *jsonOut)
	}

	// fallback 到本地 engine
	return runNotifyLocal(dd, log, *title, *body, *priority, *jsonOut)
}

func runNotifyViaServer(proxy *serverProxy, title, body, priority string, jsonOut bool) int {
	result, err := proxy.post("/api/notify", map[string]string{
		"title":    title,
		"body":     body,
		"priority": priority,
	})
	if err != nil {
		if jsonOut {
			return writeJSONError(1, "notify_failed", err.Error())
		}
		fmt.Fprintf(os.Stderr, "通知发送失败: %v\n", err)
		return 1
	}

	ok, _ := result["ok"].(bool)
	msg, _ := result["message"].(string)

	if jsonOut {
		return writeJSONLine(map[string]any{
			"ok":      ok,
			"message": msg,
		})
	}

	if ok {
		fmt.Println("通知已发送")
	} else {
		fmt.Printf("通知发送失败: %s\n", msg)
		return 1
	}
	return 0
}

func runNotifyLocal(dataDir string, log *slog.Logger, title, body, priority string, jsonOut bool) int {
	root, err := fsstore.Open(dataDir)
	if err != nil {
		if jsonOut {
			return writeJSONError(1, "internal_error", "无法打开数据目录: "+err.Error())
		}
		fmt.Fprintf(os.Stderr, "无法打开数据目录: %v\n", err)
		return 1
	}
	_ = root
	_ = priority

	// 本地模式下，通知可能不可用（飞书通知通常需要 daemon 维持连接）
	// 先返回友好提示
	if jsonOut {
		return writeJSONError(1, "notify_unavailable", "本地模式不支持通知，请先启动 gloop server")
	}
	fmt.Fprintln(os.Stderr, "本地模式不支持通知，请先启动 gloop server（gloop start）")
	return 1
}
