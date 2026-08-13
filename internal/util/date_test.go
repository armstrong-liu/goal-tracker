package util

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	// 固定一个 "now" 方便断言
	now := time.Date(2026, 8, 12, 10, 0, 0, 0, time.Local) // 周三

	tests := []struct {
		name    string
		input   string
		want    string // YYYY-MM-DD
		wantErr bool
	}{
		{"today", "today", "2026-08-12", false},
		{"tomorrow", "tomorrow", "2026-08-13", false},
		{"iso format", "2026-08-15", "2026-08-15", false},
		{"slash format", "2026/08/15", "2026-08-15", false},
		{"empty", "", "", false},
		{"next monday", "monday", "2026-08-17", false}, // 8/12 周三 → 下周一 8/17
		{"invalid", "notadate", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDate(tt.input, now)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseDate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if FormatDate(got) != tt.want {
				t.Errorf("ParseDate(%q) = %s, want %s", tt.input, FormatDate(got), tt.want)
			}
		})
	}
}

func TestIsOverdue(t *testing.T) {
	now := time.Date(2026, 8, 12, 10, 0, 0, 0, time.Local)

	if IsOverdue(time.Time{}, now) {
		t.Error("零值日期不应算过期")
	}
	if IsOverdue(now, now) {
		t.Error("今天到期的不应算过期（严格小于今天才算过期）")
	}
	past := time.Date(2026, 8, 11, 10, 0, 0, 0, time.Local)
	if !IsOverdue(past, now) {
		t.Error("昨天到期的应算过期")
	}
}

func TestCurrentQuarter(t *testing.T) {
	tests := []struct {
		month int
		want  int
	}{
		{1, 1}, {3, 1}, {4, 2}, {6, 2},
		{7, 3}, {9, 3}, {10, 4}, {12, 4},
	}
	for _, tt := range tests {
		got := CurrentQuarter(time.Date(2026, time.Month(tt.month), 15, 0, 0, 0, 0, time.Local))
		if got != tt.want {
			t.Errorf("month %d → quarter %d, want %d", tt.month, got, tt.want)
		}
	}
}

func TestISOWeek(t *testing.T) {
	// 2026-08-12 是周三，ISO 周 33
	y, w := ISOWeek(time.Date(2026, 8, 12, 0, 0, 0, 0, time.Local))
	if y != 2026 {
		t.Errorf("year = %d, want 2026", y)
	}
	if w != 33 {
		t.Errorf("week = %d, want 33", w)
	}
}

func TestWeekLabel(t *testing.T) {
	if got := WeekLabel(2026, 33); got != "2026-W33" {
		t.Errorf("WeekLabel = %q, want %q", got, "2026-W33")
	}
}

func TestQuarterLabel(t *testing.T) {
	if got := QuarterLabel(2026, 3); got != "2026 Q3" {
		t.Errorf("QuarterLabel = %q, want %q", got, "2026 Q3")
	}
}
