package out

import "strings"

const (
	escapedNewLineSuffix = "\\++"
)

var (
	keyUnescaper = strings.NewReplacer(
		"#@double-colon@#", "::",
		"#@newline@#", "\n",
	)
)

type Output struct {
	KVs  map[string]string
	Logs []Log
}

type Log struct {
	Level   string // warn, error, fatal, info
	Message string // 原始的日志内容
}

func Extract(cnt []byte) Output {
	out := Output{
		KVs:  make(map[string]string),
		Logs: make([]Log, 0),
	}
	lines := strings.Split(string(cnt), "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !strings.HasPrefix(line, "::") {
			continue
		} else if strings.HasPrefix(line, "::set-output::") {
			line = strings.TrimPrefix(line, "::set-output::")
			idx := strings.Index(line, "::")
			if idx == -1 {
				continue // 非法的内容直接跳过
			}
			key := keyUnescaper.Replace(line[:idx])
			var value string
			i, value = extractValue(line[idx+2:], i, lines)
			out.KVs[key] = value
		}
		for prefix, level := range map[string]string{
			"::set-warn::":  "warn",
			"::set-error::": "error",
			"::set-fatal::": "fatal",
			"::set-info::":  "info",
		} {
			if !strings.HasPrefix(line, prefix) {
				continue
			}
			line = strings.TrimPrefix(line, prefix)
			var msg string
			i, msg = extractValue(line, i, lines)
			out.Logs = append(out.Logs, Log{
				Level:   level,
				Message: msg,
			})
		}
	}
	return out
}

func extractValue(curLine string, i int, lines []string) (int, string) {
	valueLines := []string{}
	for strings.HasSuffix(curLine, escapedNewLineSuffix) {
		valueLines = append(valueLines, strings.TrimSuffix(curLine, escapedNewLineSuffix))
		i++ // 下一行一定是值的一部分，这里加一
		if i >= len(lines) {
			break // 这里其实不太可能，但是为了防止panic，还是加上
		}
		curLine = lines[i]
	}
	if !strings.HasSuffix(curLine, escapedNewLineSuffix) {
		valueLines = append(valueLines, curLine)
	}
	return i, strings.Join(valueLines, "\n")
}
