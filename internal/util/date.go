package util

import (
	"fmt"
	"strings"
	"time"
)

// ISOWeek 返回给定时间（默认当前时间）的 ISO 周年和周数。
// 例：2026 年第 32 周返回 (2026, 32)。
func ISOWeek(t time.Time) (year int, week int) {
	if t.IsZero() {
		t = time.Now()
	}
	return t.ISOWeek()
}

// CurrentQuarter 返回给定时间所在季度（1-4）。
func CurrentQuarter(t time.Time) int {
	if t.IsZero() {
		t = time.Now()
	}
	return int(t.Month()-1)/3 + 1
}

// QuarterForMonth 根据月份返回季度（1-4）。
func QuarterForMonth(month int) (int, error) {
	if month < 1 || month > 12 {
		return 0, fmt.Errorf("invalid month: %d", month)
	}
	return (month - 1) / 3 + 1, nil
}

// ParseDate 解析日期关键词或 YYYY-MM-DD 格式字符串。
// 支持的关键词：today / tomorrow / monday..sunday（下一个该工作日）。
// 返回的 time.Time 仅日期部分有效（时间归零到当地零点）。
func ParseDate(s string, now time.Time) (time.Time, error) {
	if now.IsZero() {
		now = time.Now()
	}
	s = strings.TrimSpace(strings.ToLower(s))

	if s == "" {
		return time.Time{}, nil // 空字符串表示无截止日期
	}

	switch s {
	case "today":
		return truncateToDate(now), nil
	case "tomorrow":
		return truncateToDate(now.AddDate(0, 0, 1)), nil
	case "monday", "tuesday", "wednesday", "thursday",
		"friday", "saturday", "sunday":
		return nextWeekday(now, parseWeekday(s)), nil
	}

	// 尝试 YYYY-MM-DD
	layouts := []string{"2006-01-02", "2006/01/02", "2006-1-2", "2006/1/2"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析日期: %q (支持 today/tomorrow/monday..sunday 或 YYYY-MM-DD)", s)
}

// FormatDate 将时间格式化为 YYYY-MM-DD 字符串。
// 零值返回空字符串。
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// IsOverdue 判断任务截止日期是否已过期（严格小于今天）。
// 零值（无截止日期）返回 false。
func IsOverdue(due time.Time, now time.Time) bool {
	if due.IsZero() {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	return truncateToDate(due).Before(truncateToDate(now))
}

// IsToday 判断给定日期是否为今天。
func IsToday(t time.Time, now time.Time) bool {
	if t.IsZero() {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	return truncateToDate(t).Equal(truncateToDate(now))
}

// truncateToDate 将时间截断到当日零点。
func truncateToDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// parseWeekday 将 monday..sunday 字符串转为 time.Weekday。
func parseWeekday(s string) time.Weekday {
	switch s {
	case "sunday":
		return time.Sunday
	case "monday":
		return time.Monday
	case "tuesday":
		return time.Tuesday
	case "wednesday":
		return time.Wednesday
	case "thursday":
		return time.Thursday
	case "friday":
		return time.Friday
	case "saturday":
		return time.Saturday
	default:
		return time.Sunday
	}
}

// nextWeekday 计算从 from 起，下一个目标 weekday 的日期（不含今天）。
// 例：from 是周三，目标 weekday 是周三，则返回下周三。
func nextWeekday(from time.Time, target time.Weekday) time.Time {
	offset := (int(target) - int(from.Weekday()) + 7) % 7
	if offset == 0 {
		offset = 7 // 同一天则跳到下周
	}
	return truncateToDate(from.AddDate(0, 0, offset))
}

// WeekLabel 返回 ISO 周的展示标签，如 "2026-W32"。
func WeekLabel(year, week int) string {
	return fmt.Sprintf("%04d-W%02d", year, week)
}

// QuarterLabel 返回季度展示标签，如 "2026 Q3"。
func QuarterLabel(year, quarter int) string {
	return fmt.Sprintf("%04d Q%d", year, quarter)
}
