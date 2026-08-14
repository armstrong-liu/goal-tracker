package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"goal-tracker/internal/model"
	"goal-tracker/internal/store"
	"goal-tracker/internal/util"
)

// newFeatureStore 创建带目标层级的测试数据。
func newFeatureStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "feature_test.db"))
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	// 建层级：年 → 季 → 周
	yg, _ := s.CreateYearGoal(store.CreateYearGoalInput{Title: "年度目标", Year: time.Now().Year()})
	qg, _ := s.CreateQuarterGoal(store.CreateQuarterGoalInput{
		Title: "季度目标标题比较长的情况", Year: time.Now().Year(),
		Quarter: util.CurrentQuarter(time.Now()), YearGoalID: &yg.ID,
	})
	y, w := util.ISOWeek(time.Now())
	wg, _ := s.CreateWeekGoal(store.CreateWeekGoalInput{
		Title: "周目标标题", Year: y, Week: w, QuarterGoalID: &qg.ID,
	})
	s.CreateTask(store.CreateTaskInput{Title: "任务1", WeekGoalID: &wg.ID})
	return s
}

// withSize 设置窗口尺寸（Update 返回新 Model，必须接收返回值）。
func withSize(m Model, w, h int) Model {
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(Model)
}

// ---------- 方案3：Space 切换目标完成 ----------

func TestModel_SpaceToggleWeekGoal(t *testing.T) {
	s := newFeatureStore(t)
	m := withSize(NewModel(s), 100, 30)

	// 切到周视图（默认是今日任务）
	updated, _ := m.Update(keyMsg("2"))
	m = updated.(Model)

	// 按 Space → 周目标应变为 completed
	updated, _ = m.Update(keyMsg(" "))
	m = updated.(Model)

	goals, _ := s.ListWeekGoals(store.WeekGoalFilter{})
	if len(goals) == 0 || goals[0].Status != model.WeekGoalStatusCompleted {
		t.Errorf("按 Space 后周目标状态 = %v，应为 completed", goals)
	}

	// 再按 Space → 恢复 active
	m.Update(keyMsg(" "))
	goals, _ = s.ListWeekGoals(store.WeekGoalFilter{})
	if len(goals) == 0 || goals[0].Status != model.WeekGoalStatusActive {
		t.Errorf("再按 Space 后周目标状态 = %v，应为 active", goals)
	}
}

func TestModel_SpaceToggleQuarterGoal(t *testing.T) {
	s := newFeatureStore(t)
	m := withSize(NewModel(s), 100, 30)

	updated, _ := m.Update(keyMsg("3"))
	m = updated.(Model)
	m.Update(keyMsg(" "))

	goals, _ := s.ListQuarterGoals(store.QuarterGoalFilter{})
	if len(goals) == 0 || goals[0].Status != model.QuarterGoalStatusCompleted {
		t.Errorf("按 Space 后季度目标状态 = %v，应为 completed", goals)
	}
}

func TestModel_SpaceToggleYearGoal(t *testing.T) {
	s := newFeatureStore(t)
	m := withSize(NewModel(s), 100, 30)

	updated, _ := m.Update(keyMsg("4"))
	m = updated.(Model)
	m.Update(keyMsg(" "))

	goals, _ := s.ListYearGoals(store.YearGoalFilter{})
	if len(goals) == 0 || goals[0].Status != model.YearGoalStatusCompleted {
		t.Errorf("按 Space 后年度目标状态 = %v，应为 completed", goals)
	}
}

// ---------- 方案2：Enter 详情面板 ----------

func TestModel_EnterShowsDetail_FullTitle(t *testing.T) {
	s := newFeatureStore(t)
	// 添加一个超长标题任务，验证详情面板不截断
	longTitle := "这是一个非常非常长的任务标题用来验证详情面板可以完整显示所有内容不截断"
	s.CreateTask(store.CreateTaskInput{Title: longTitle})

	m := withSize(NewModel(s), 100, 30)

	// 任务列表第一个是"任务1"，按 j 下移到长标题任务
	updated, _ := m.Update(keyMsg("j"))
	m = updated.(Model)

	// 按 Enter 打开详情
	updated, _ = m.Update(keyMsg("enter"))
	m = updated.(Model)
	if !m.showDetail {
		t.Fatal("按 Enter 后 showDetail 应为 true")
	}

	// 渲染详情面板，完整标题应存在（不截断、不换行拆断）
	detail := m.renderDetailPanel()
	plain := stripStyleANSI(detail)
	if !strings.Contains(plain, longTitle) {
		t.Errorf("详情面板应包含完整标题（%d 字），实际输出：\n%s", len(longTitle), plain)
	}
}

func TestModel_EnterDetailOnWeekGoal(t *testing.T) {
	s := newFeatureStore(t)
	m := withSize(NewModel(s), 100, 30)

	// 切到周视图后按 Enter
	updated, _ := m.Update(keyMsg("2"))
	m = updated.(Model)
	updated, _ = m.Update(keyMsg("enter"))
	m = updated.(Model)
	if !m.showDetail {
		t.Fatal("周视图按 Enter 应打开详情")
	}
	detail := m.renderDetailPanel()
	if !strings.Contains(stripStyleANSI(detail), "周目标标题") {
		t.Error("周目标详情应包含标题")
	}
	// 应显示上级（季度目标）标题
	if !strings.Contains(stripStyleANSI(detail), "季度目标标题比较长的情况") {
		t.Error("周目标详情应包含上级季度目标的标题")
	}
}

func TestModel_DetailAnyKeyCloses(t *testing.T) {
	s := newFeatureStore(t)
	m := withSize(NewModel(s), 100, 30)

	updated, _ := m.Update(keyMsg("enter"))
	m = updated.(Model)
	if !m.showDetail {
		t.Fatal("前置条件失败")
	}

	// 任意键（如 j）应关闭面板
	updated, _ = m.Update(keyMsg("j"))
	m = updated.(Model)
	if m.showDetail {
		t.Error("按任意键后 showDetail 应为 false")
	}

	// q 在详情打开时不应退出程序，而是关闭面板
	updated, _ = m.Update(keyMsg("enter"))
	m = updated.(Model)
	updated, cmd := m.Update(keyMsg("q"))
	m = updated.(Model)
	if m.showDetail {
		t.Error("详情打开时按 q 应关闭面板而不是退出")
	}
	if cmd != nil {
		t.Error("详情打开时按 q 不应返回退出命令")
	}
}

// ---------- 方案4：列宽自适应 ----------

func TestAdaptiveTitleWidth(t *testing.T) {
	cases := []struct {
		width, reserve, min, want int
	}{
		{100, 49, 20, 51},  // 宽终端：100-49=51
		{60, 49, 20, 20},   // 窄终端：60-49=11 < 20，保底 20
		{200, 49, 20, 151}, // 超宽终端
	}
	for _, c := range cases {
		got := adaptiveTitleWidth(c.width, c.reserve, c.min)
		if got != c.want {
			t.Errorf("adaptiveTitleWidth(%d, %d, %d) = %d, want %d",
				c.width, c.reserve, c.min, got, c.want)
		}
	}
}

func TestTaskColumnsForWidth(t *testing.T) {
	cols := taskColumnsForWidth(120)
	if len(cols) != 6 {
		t.Fatalf("列数 = %d，应为 6", len(cols))
	}
	// 标题列宽 = 120 - 49 = 71
	if cols[2].Width != 71 {
		t.Errorf("标题列宽 = %d，应为 71", cols[2].Width)
	}

	// 窄终端保底
	cols = taskColumnsForWidth(50)
	if cols[2].Width < 20 {
		t.Errorf("窄终端标题列宽 %d 不应低于 20", cols[2].Width)
	}
}

// ---------- 辅助 ----------

// stripStyleANSI 去除 ANSI 转义序列（详情面板含样式）。
func stripStyleANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		if r == 0x1b || r == 0x9b {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
