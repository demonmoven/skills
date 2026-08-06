package version

// Brand 集中管理 Gloop 品牌、吉祥物、目录、端口等所有对外字符串。
// 要换名只改这一个文件。

var Version = "0.6.0"

const (
	AppName  = "Gloop" // 正式品牌名
	RepoName = "gloop" // repo 名 / 二进制名 / CLI 主命令
	// Daemon 保留仅用于兼容旧的 `gloopd` 二进制调用：
	// 在 cli.Run 里检测到 argv[0] == "gloopd" 时，会打印 deprecation 警告并转发到 gloop start --no-detach。
	// npm bin 不再单独暴露 gloopd。
	Daemon = "gloopd" // Deprecated: 使用 gloop start 代替
	Slogan = "Agent 转生成为冒险者，然后陷入循环"
	Mascot = "Gloop" // 吉祥物（史莱姆），loading/empty state 用

	// 默认端口（A=2, G=4 不合适，用 AG 的 ASCII 累加值 37317 避撞）
	DefaultPort = 37317
	DefaultHost = "0.0.0.0"

	// CLI 鉴权 token 文件名（放 ~/.gloop/ 下）
	TokenFileName = "token.json"
)

// DirName 返回隐藏目录名（~/.gloop）
func DirName() string { return "." + RepoName }
