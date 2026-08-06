package orchestrator

import (
	"testing"
	"time"
)

func TestParseCronField_Star(t *testing.T) {
	f, err := parseCronField("*", 0, 59)
	if err != nil {
		t.Fatalf("parseCronField(*) err=%v", err)
	}
	if !f.isStar {
		t.Error("* should mark isStar=true")
	}
	for i := 0; i <= 59; i++ {
		if !f.matches(i) {
			t.Fatalf("* should match minute %d", i)
		}
	}
}

func TestParseCronField_SingleValue(t *testing.T) {
	f, err := parseCronField("3", 0, 59)
	if err != nil {
		t.Fatal(err)
	}
	if f.isStar {
		t.Error("3 should not be isStar")
	}
	if !f.matches(3) {
		t.Error("should match 3")
	}
	if f.matches(4) {
		t.Error("should not match 4")
	}
}

func TestParseCronField_Range(t *testing.T) {
	f, err := parseCronField("1-5", 0, 59)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 5; i++ {
		if !f.matches(i) {
			t.Fatalf("1-5 should match %d", i)
		}
	}
	if f.matches(0) || f.matches(6) {
		t.Error("1-5 should not match 0 or 6")
	}
}

func TestParseCronField_Enumeration(t *testing.T) {
	f, err := parseCronField("0,15,30,45", 0, 59)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []int{0, 15, 30, 45} {
		if !f.matches(v) {
			t.Errorf("enumeration should include %d", v)
		}
	}
	if f.matches(1) {
		t.Error("enumeration should not include 1")
	}
}

func TestParseCronField_StepFromStar(t *testing.T) {
	f, err := parseCronField("*/10", 0, 59)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []int{0, 10, 20, 30, 40, 50} {
		if !f.matches(v) {
			t.Errorf("*/10 should match %d", v)
		}
	}
	if f.matches(5) || f.matches(55) {
		t.Error("*/10 should not match 5 or 55")
	}
}

func TestParseCronField_StepInRange(t *testing.T) {
	f, err := parseCronField("1-10/3", 0, 59)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []int{1, 4, 7, 10} {
		if !f.matches(v) {
			t.Errorf("1-10/3 should match %d", v)
		}
	}
	if f.matches(13) {
		t.Error("1-10/3 should not match 13")
	}
}

func TestParseCronField_DayOfWeekSunday7(t *testing.T) {
	// 星期字段：7 应归一为 0（周日）
	f, err := parseCronField("7", 0, 6)
	if err != nil {
		t.Fatal(err)
	}
	if !f.matches(0) {
		t.Error("7 should match 0 (sunday) on dayOfWeek field")
	}
	if f.matches(7) {
		t.Error("7 should not be kept in dayOfWeek 0-6 range")
	}
	// 范围 1-7
	f2, err := parseCronField("1-7", 0, 6)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i <= 6; i++ {
		if !f2.matches(i) {
			t.Errorf("1-7 should match %d", i)
		}
	}
}

func TestParseCronField_InvalidRejected(t *testing.T) {
	cases := []struct {
		s     string
		min   int
		max   int
	}{
		{"-1", 0, 59},     // 不是数字
		{"60", 0, 59},     // 超出范围
		{"1-60", 0, 59},   // 范围超出
		{"5-3", 0, 59},    // 开始大于结束
		{"*/0", 0, 59},    // step=0
		{"*/abc", 0, 59},  // step 非数字
		{"a-b", 0, 59},    // 非数字范围
		{"abc", 0, 59},    // 非数字
		{"", 0, 59},       // 空
		{",", 0, 59},      // 仅分隔符
	}
	for i, c := range cases {
		if _, err := parseCronField(c.s, c.min, c.max); err == nil {
			t.Errorf("case %d: parseCronField(%q,%d,%d) expected error", i, c.s, c.min, c.max)
		}
	}
}

func TestNormalizeCronAlias(t *testing.T) {
	cases := map[string]string{
		"hourly":   "0 * * * *",
		"daily":    "0 9 * * *",
		"weekly":   "0 9 * * 1",
		"DAILY":    "0 9 * * *",
		"0 0 * * *": "0 0 * * *",
		"":         "",
	}
	for in, want := range cases {
		if got := normalizeCronAlias(in); got != want {
			t.Errorf("normalizeCronAlias(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestParseCron_WrongFields(t *testing.T) {
	if _, err := ParseCron("* * *"); err == nil {
		t.Error("3 fields should be rejected")
	}
	if _, err := ParseCron("* * * * * *"); err == nil {
		t.Error("6 fields should be rejected")
	}
	if _, err := ParseCron("* * * * x"); err == nil {
		t.Error("invalid day-of-week should be rejected")
	}
}

func TestCronSchedule_Matches(t *testing.T) {
	// "30 9 * * *" 每天 09:30
	c, err := ParseCron("30 9 * * *")
	if err != nil {
		t.Fatal(err)
	}
	matchTime := time.Date(2025, 1, 15, 9, 30, 0, 0, time.UTC)
	if !c.Matches(matchTime) {
		t.Errorf("30 9 * * * should match %v", matchTime)
	}
	if c.Matches(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)) {
		t.Error("should not match hour=10")
	}
	if c.Matches(time.Date(2025, 1, 15, 9, 31, 0, 0, time.UTC)) {
		t.Error("should not match minute=31")
	}

	// 每周一 9:00
	weekly, err := ParseCron("0 9 * * 1")
	if err != nil {
		t.Fatal(err)
	}
	monday := time.Date(2025, 6, 23, 9, 0, 0, 0, time.UTC) // 周一
	if monday.Weekday() != time.Monday {
		t.Fatalf("test assumption failed: %v weekday=%v", monday, monday.Weekday())
	}
	if !weekly.Matches(monday) {
		t.Errorf("0 9 * * 1 should match monday 9:00")
	}
	tuesday := time.Date(2025, 6, 24, 9, 0, 0, 0, time.UTC) // 周二
	if weekly.Matches(tuesday) {
		t.Error("0 9 * * 1 should NOT match tuesday")
	}

	// 日+周 OR 关系："0 9 15 * 1" → 每月 15 号或 周一
	both, err := ParseCron("0 9 15 * 1")
	if err != nil {
		t.Fatal(err)
	}
	day15 := time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC) // 周日
	if day15.Weekday() != time.Sunday {
		t.Fatalf("assumption failed")
	}
	if !both.Matches(day15) {
		t.Error("15th should match even if not monday (OR)")
	}
	if !both.Matches(monday) {
		t.Error("monday should match even if day !=15 (OR)")
	}
	notEither := time.Date(2025, 6, 10, 9, 0, 0, 0, time.UTC) // 周二 day=10
	if both.Matches(notEither) {
		t.Error("neither day=15 nor mon should not match")
	}
}

func TestParseCronAliasExpansion(t *testing.T) {
	c, err := ParseCron("hourly")
	if err != nil {
		t.Fatal(err)
	}
	if !c.minute.matches(0) {
		t.Error("hourly must fire at minute 0")
	}
	// 每分钟都行
	for h := 0; h <= 23; h++ {
		if !c.hour.matches(h) {
			t.Fatalf("hourly should match hour %d", h)
		}
	}
}
