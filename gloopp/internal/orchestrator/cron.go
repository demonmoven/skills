package orchestrator

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ==================== 简化版 cron 表达式解析 ====================
//
// 支持 5 段标准 cron: 分 时 日 月 周
// 语法：
//   *         任意值
//   N         具体值
//   */N       每隔 N
//   A-B       范围
//   A,B,C     枚举
//   A-B/N     范围内每隔 N
//
// 只用于调度器分钟级匹配，不处理秒。

type cronField struct {
	values map[int]bool // 该字段允许的值集合
	min    int
	max    int
	isStar bool // 是否原生为 "*"（用于区分「用户写了 *」vs「用户枚举了全部值」）
}

func parseCronField(s string, min, max int) (*cronField, error) {
	f := &cronField{
		values: make(map[int]bool),
		min:    min,
		max:    max,
	}

	if s == "*" {
		f.isStar = true
		for i := min; i <= max; i++ {
			f.values[i] = true
		}
		return f, nil
	}

	// 按逗号拆成多个子项
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if err := f.addPart(part); err != nil {
			return nil, err
		}
	}

	if len(f.values) == 0 {
		return nil, fmt.Errorf("cron 字段为空: %q", s)
	}
	return f, nil
}

func (f *cronField) addPart(part string) error {
	// 处理 step: */N 或 A-B/N
	var step int
	stepIdx := strings.Index(part, "/")
	if stepIdx >= 0 {
		stepStr := part[stepIdx+1:]
		n, err := strconv.Atoi(stepStr)
		if err != nil || n <= 0 {
			return fmt.Errorf("无效的 step 值: %q", stepStr)
		}
		step = n
		part = part[:stepIdx]
	}

	var start, end int
	if part == "*" {
		start = f.min
		end = f.max
	} else if strings.Contains(part, "-") {
		rangeParts := strings.SplitN(part, "-", 2)
		s, err := strconv.Atoi(rangeParts[0])
		if err != nil {
			return fmt.Errorf("无效范围值: %q", rangeParts[0])
		}
		e, err := strconv.Atoi(rangeParts[1])
		if err != nil {
			return fmt.Errorf("无效范围值: %q", rangeParts[1])
		}
		// 周日：7 → 0（仅对 dayOfWeek 字段生效）
		if f.min == 0 && f.max == 6 {
			if s == 7 {
				s = 0
			}
			if e == 7 {
				e = 0
			}
		}
		// dayOfWeek 下 range 归一后出现 start>end（如 "1-7" -> 1-0）
		// 表示 wraparound 的 "全周"，展开为整个范围，避免越界校验失败。
		if f.min == 0 && f.max == 6 && s > e {
			s = f.min
			e = f.max
		}
		start = s
		end = e
	} else {
		// 单个值
		v, err := strconv.Atoi(part)
		if err != nil {
			return fmt.Errorf("无效 cron 值: %q", part)
		}
		// 周日：输入 7 归一到 0（当范围允许 0-6 时，即 dayOfWeek）
		if f.min == 0 && f.max == 6 && v == 7 {
			v = 0
		}
		start = v
		end = v
	}

	if start < f.min || end > f.max || start > end {
		return fmt.Errorf("值超出范围 [%d-%d]: %d-%d", f.min, f.max, start, end)
	}

	if step > 0 {
		for i := start; i <= end; i += step {
			f.values[i] = true
		}
	} else {
		for i := start; i <= end; i++ {
			f.values[i] = true
		}
	}
	return nil
}

func (f *cronField) matches(v int) bool {
	return f.values[v]
}

// cronSchedule 表示一个 cron 调度计划
type cronSchedule struct {
	minute     *cronField
	hour       *cronField
	dayOfMonth *cronField
	month      *cronField
	dayOfWeek  *cronField
}

// ParseCron 解析 5 段 cron 表达式
func ParseCron(expr string) (*cronSchedule, error) {
	expr = normalizeCronAlias(expr)
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron 表达式必须有 5 段，实际 %d 段: %q", len(fields), expr)
	}

	min, err := parseCronField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("分钟字段无效: %w", err)
	}
	hour, err := parseCronField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("小时字段无效: %w", err)
	}
	day, err := parseCronField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("日期字段无效: %w", err)
	}
	month, err := parseCronField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("月份字段无效: %w", err)
	}
	// 周日 = 0 或 7，统一为 0-6（对 7 归一到 0，对 "0-6" 范围、列表等已由 addPart 处理）
	week, err := parseCronField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("星期字段无效: %w", err)
	}

	return &cronSchedule{
		minute:     min,
		hour:       hour,
		dayOfMonth: day,
		month:      month,
		dayOfWeek:  week,
	}, nil
}

func normalizeCronAlias(expr string) string {
	switch strings.ToLower(strings.TrimSpace(expr)) {
	case "hourly":
		return "0 * * * *"
	case "daily":
		return "0 9 * * *"
	case "weekly":
		return "0 9 * * 1"
	default:
		return expr
	}
}

// Matches 判断给定时间是否匹配 cron 表达式（精确到分钟）
func (c *cronSchedule) Matches(t time.Time) bool {
	// 日和周是 OR 关系（标准 cron 语义）：
	// 如果日和周都不是 *，则满足一个就算匹配
	dayMatch := c.dayOfMonth.matches(t.Day())
	monthMatch := c.month.matches(int(t.Month()))
	weekMatch := c.dayOfWeek.matches(int(t.Weekday()))
	hourMatch := c.hour.matches(t.Hour())
	minuteMatch := c.minute.matches(t.Minute())

	if !monthMatch || !hourMatch || !minuteMatch {
		return false
	}

	// 日和周的 OR 逻辑：
	// 标准 cron：如果日和周都有限制（不是 *），则 OR
	// 如果只有一个有限制，则只看那个
	dayIsStar := c.dayOfMonth.isStar
	weekIsStar := c.dayOfWeek.isStar

	if dayIsStar && weekIsStar {
		return true
	}
	if dayIsStar {
		return weekMatch
	}
	if weekIsStar {
		return dayMatch
	}
	return dayMatch || weekMatch
}
