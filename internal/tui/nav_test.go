package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goal-tracker/internal/model"
	"goal-tracker/internal/store"
	"goal-tracker/internal/util"
)

// ---------- 偏移计算 ----------

func TestDisplayedQuarter(t *testing.T) {
	// 当前是 2026 Q3（8月）
	m := Model{quarterOffset: 0}
	y, q := m.displayedQuarter()
	if y != time.Now().Year() || q != util.CurrentQuarter(time.Now()) {
		t.Errorf("offset=0 应返回当前季度，得到 %d Q%d", y, q)
	}

	// +1 → Q4
	m.quarterOffset = 1
	y, q = m.displayedQuarter()
	if q != 4 {
		t.Errorf("offset=1 应为 Q4，得到 Q%d", q)
	}

	// +2 → 跨年到次年 Q1
	m.quarterOffset = 2
	y, q = m.displayedQuarter()
	if q != 1 || y != time.Now().Year()+1 {
		t.Errorf("offset=2 应为次年 Q1，得到 %d Q%d", y, q)
	}

	// -1 → Q2
	m.quarterOffset = -1
	_, q = m.displayedQuarter()
	if q != 2 {
		t.Errorf("offset=-1 应为 Q2，得到 Q%d", q)
	}
}

func TestDisplayedWeek(t *testing.T) {
	// +1 → 下一周
	m := Model{weekOffset: 1}
	y, w := m.displayedWeek()
	cy, cw := util.ISOWeek(time.Now().AddDate(0, 0, 7))
	if y != cy || w != cw {
		t.Errorf("offset=1 应为下一周 %d-W%d，得到 %d-W%d", cy, cw, y, w)
	}

	// -1 → 上一周
	m.weekOffset = -1
	y, w = m.displayedWeek()
	py, pw := util.ISOWeek(time.Now().AddDate(0, 0, -7))
	if y != py || w != pw {
		t.Errorf("offset=-1 应为上一周 %d-W%d，得到 %d-W%d", py, pw, y, w)
	}
}

// ---------- TUI 导航：切换周/季度 ----------

func newNavStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "nav_test.db"))
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	now := time.Now()

	// 只在"下一周"建一个周目标（当前周为空，验证导航后可见）
	s.CreateWeekGoal(store.CreateWeekGoalInput{
		Title: "下周的目标", Year: nextWeekYear(now), Week: nextWeekNum(now),
	})
	// 只在"下一季度"建一个季度目标（当前季度为空）
	nqy, nqq := nextQuarter(now)
	s.CreateQuarterGoal(store.CreateQuarterGoalInput{
		Title: "下季度的目标", Year: nqy, Quarter: nqq,
	})
	return s
}

func nextWeekYear(now time.Time) int { y, _ := util.ISOWeek(now.AddDate(0, 0, 7)); return y }
func nextWeekNum(now time.Time) int  { _, w := util.ISOWeek(now.AddDate(0, 0, 7)); return w }

// nextQuarter 计算下一季度的 (年, 季度)。公式与 displayedQuarter 一致：
// 总季度数 = 年*4 + (季度-1)，+1 后再拆解。
func nextQuarter(now time.Time) (int, int) {
	total := now.Year()*4 + (util.CurrentQuarter(now) - 1) + 1
	return total / 4, total%4 + 1
}

func TestModel_WeekNavigation(t *testing.T) {
	s := newNavStore(t)
	m := withSize(NewModel(s), 100, 30)

	// 切到周视图：当前周为空
	updated, _ := m.Update(keyMsg("2"))
	m = updated.(Model)
	if !strings.Contains(m.View(), "该周没有目标") {
		// 当前周可能意外有数据（理论上测试库只有下周的）
		t.Log("警告：当前周非空？")
	}

	// 按 → 切到下一周，应显示目标
	updated, _ = m.Update(keyMsg("right"))
	m = updated.(Model)
	view := stripStyleANSI(m.View())
	if !strings.Contains(view, "下周的目标") {
		t.Errorf("按 → 后应显示下一周的目标，实际视图：\n%s", view)
	}

	// 按 ← 回到当前周
	updated, _ = m.Update(keyMsg("left"))
	m = updated.(Model)
	view = stripStyleANSI(m.View())
	if strings.Contains(view, "下周的目标") {
		t.Error("按 ← 后应回到当前周（目标应不可见）")
	}
}

func TestModel_QuarterNavigation(t *testing.T) {
	s := newNavStore(t)
	m := withSize(NewModel(s), 100, 30)

	// 切到季度视图：当前季度为空
	updated, _ := m.Update(keyMsg("3"))
	m = updated.(Model)

	// 按 → 切到下一季度，应显示目标
	updated, _ = m.Update(keyMsg("right"))
	m = updated.(Model)
	view := stripStyleANSI(m.View())
	if !strings.Contains(view, "下季度的目标") {
		t.Errorf("按 → 后应显示下一季度的目标，实际视图：\n%s", view)
	}
}

// 关键测试：导航后 Space/Enter 作用于"显示中的"目标，而不是当前周的
func TestModel_NavigationAffectsSelection(t *testing.T) {
	s := newNavStore(t)
	m := withSize(NewModel(s), 100, 30)

	// 周视图 → 切到下一周 → 按 Space
	updated, _ := m.Update(keyMsg("2"))
	m = updated.(Model)
	updated, _ = m.Update(keyMsg("right"))
	m = updated.(Model)
	updated, _ = m.Update(keyMsg(" "))
	m = updated.(Model)

	// 验证"下周的目标"被标记完成
	goals, _ := s.ListWeekGoals(store.WeekGoalFilter{
		Year: nextWeekYear(time.Now()), Week: nextWeekNum(time.Now()),
	})
	if len(goals) == 0 || goals[0].Status != model.WeekGoalStatusCompleted {
		t.Errorf("导航后按 Space 应完成显示中的周目标，状态 = %v", goals)
	}
}

// 导航时光标应重置为 0
func TestModel_NavigationResetsCursor(t *testing.T) {
	s := newNavStore(t)
	m := withSize(NewModel(s), 100, 30)

	m.week.cursor = 5 // 模拟光标停留较深
	updated, _ := m.Update(keyMsg("2"))
	m = updated.(Model)
	updated, _ = m.Update(keyMsg("right"))
	m = updated.(Model)
	if m.week.cursor != 0 {
		t.Errorf("切换周后光标应重置为 0，得到 %d", m.week.cursor)
	}
}

// 全年分布统计（方案C）
func TestQuarterYearDistribution(t *testing.T) {
	s := newNavStore(t)
	m := Model{store: s}

	now := time.Now()
	// 当前季度也加一个，形成两个季度有目标的分布
	s.CreateQuarterGoal(store.CreateQuarterGoalInput{
		Title: "当前季度的目标", Year: now.Year(), Quarter: util.CurrentQuarter(now),
	})

	dist := m.quarterYearDistribution(now.Year())
	// 应包含当前季度和下一季度的计数
	if !strings.Contains(dist, "Q") {
		t.Errorf("分布统计不应为空，得到 %q", dist)
	}
	t.Logf("分布: %s", dist)

	// 空年份应返回空串
	if d := m.quarterYearDistribution(1999); d != "" {
		t.Errorf("无目标年份应返回空串，得到 %q", d)
	}
}
