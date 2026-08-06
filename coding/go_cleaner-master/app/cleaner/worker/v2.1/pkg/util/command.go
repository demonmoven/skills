package util

import (
	"strconv"
	"strings"
)

// ParseCommandLineStringSimple 解析命令行字符串形式的参数, targetParam示例: --timeout, -timeout
func ParseCommandLineStringSimple(cmd string, targetParam string) string {
	args := strings.Fields(cmd)

	targetValue := ""

	for i, arg := range args {
		if arg == targetParam && i+1 < len(args) {
			// 如果参数和目标参数匹配，且后面还有参数值
			targetValue = args[i+1]
			break
		} else if strings.HasPrefix(arg, targetParam+"=") {
			// 参数的值可能直接跟在等号后面
			targetValue = strings.TrimPrefix(arg, targetParam+"=")
			// 去掉参数值两端的双引号（如果有）
			targetValue = strings.Trim(targetValue, "\"")
			break
		} else if strings.HasPrefix(arg, targetParam+" ") {
			// 参数的值可能直接跟在空格后面
			targetValue = strings.TrimPrefix(arg, targetParam+" ")
			targetValue = strings.Trim(targetValue, "\"")
			break
		}
	}

	return targetValue
}

// ParseCommandLineInt 解析命令行int的参数, targetParam示例: --timeout, -timeout
func ParseCommandLineInt(cmd string, targetParam string) *int {
	s := ParseCommandLineStringSimple(cmd, targetParam)
	if s == "" {
		return nil
	}
	timeout, _ := strconv.Atoi(s)
	return &timeout
}
