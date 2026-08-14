package tui

import (
	"testing"
	"time"

	"goal-tracker/internal/store"
	"goal-tracker/internal/util"
)

// ---------- x 删除：所有视图 ----------

func TestModel_DeleteQuarterGoal(t *testing.T) {
	s := newFeatureStore(t) // 建了 1 个季度目标
	m := withSize(NewModel(s), 100, 30)

	// 切到季度视图 → 按 x → 输入 y → 回车
	updated, _ := m.Update(keyMsg("3"))
	m = updated.(Model)
	updated, _ = m.Update(keyMsg("x"))
	m = updated.(Model)
	if m.mode != inputDelete {
		t.Fatal("季度视图按 x 应进入删除确认模式")
	}
	// 确认面板应显示目标标题（防误删）
	if m.pendingTitle == "" {
		t.Error("pendingTitle 应记录目标标题")
	}

	m.textInput.SetValue("y")
	updated, _ = m.Update(keyMsg("enter"))
	m = updated.(Model)

	// 验证已删除
	goals, _ := s.ListQuarterGoals(store.QuarterGoalFilter{})
	if len(goals) != 0 {
		t.Errorf("季度目标应已删除，剩余 %d 个", len(goals))
	}
}

func TestModel_DeleteWeekGoal_Cancel(t *testing.T) {
	s := newFeatureStore(t)
	m := withSize(NewModel(s), 100, 30)

	updated, _ := m.Update(keyMsg("2"))
	m = updated.(Model)
	updated, _ = m.Update(keyMsg("x"))
	m = updated.(Model)

	// 输入 n 取消
	m.textInput.SetValue("n")
	updated, _ = m.Update(keyMsg("enter"))
	m = updated.(Model)

	goals, _ := s.ListWeekGoals(store.WeekGoalFilter{})
	if len(goals) != 1 {
		t.Errorf("取消删除后周目标应保留，剩余 %d 个", len(goals))
	}
}

func TestModel_DeleteYearGoal(t *testing.T) {
	s := newFeatureStore(t)
	m := withSize(NewModel(s), 100, 30)

	updated, _ := m.Update(keyMsg("4"))
	m = updated.(Model)
	updated, _ = m.Update(keyMsg("x"))
	m = updated.(Model)
	m.textInput.SetValue("y")
	updated, _ = m.Update(keyMsg("enter"))
	m = updated.(Model)

	goals, _ := s.ListYearGoals(store.YearGoalFilter{})
	if len(goals) != 0 {
		t.Errorf("年度目标应已删除，剩余 %d 个", len(goals))
	}
}

// ---------- a 添加：所有视图 ----------

func TestModel_AddQuarterGoal(t *testing.T) {
	s := newFeatureStore(t)
	m := withSize(NewModel(s), 100, 30)

	// 切到季度视图 → 按 a → 输入标题 → 回车
	updated, _ := m.Update(keyMsg("3"))
	m = updated.(Model)
	updated, _ = m.Update(keyMsg("a"))
	m = updated.(Model)
	if m.mode != inputAdd || m.pendingKind != kindQuarterGoal {
		t.Fatalf("季度视图按 a 应进入添加季度目标模式，mode=%v kind=%v", m.mode, m.pendingKind)
	}

	m.textInput.SetValue("新建的季度目标")
	updated, _ = m.Update(keyMsg("enter"))
	m = updated.(Model)

	goals, _ := s.ListQuarterGoals(store.QuarterGoalFilter{})
	if len(goals) != 2 {
		t.Fatalf("应有 2 个季度目标，得到 %d", len(goals))
	}
	found := false
	for _, qg := range goals {
		if qg.Title == "新建的季度目标" {
			found = true
			// 应落在当前季度
			if qg.Quarter != util.CurrentQuarter(time.Now()) {
				t.Errorf("新目标应落在当前季度 Q%d，得到 Q%d",
					util.CurrentQuarter(time.Now()), qg.Quarter)
			}
		}
	}
	if !found {
		t.Error("未找到新建的季度目标")
	}
}

// 关键测试：导航到下一周后按 a，目标应创建在"显示中的周"而不是当前周
func TestModel_AddWeekGoalAfterNavigation(t *testing.T) {
	s := newFeatureStore(t)
	m := withSize(NewModel(s), 100, 30)

	// 周视图 → 切到下一周 → 添加目标
	updated, _ := m.Update(keyMsg("2"))
	m = updated.(Model)
	updated, _ = m.Update(keyMsg("right"))
	m = updated.(Model)
	updated, _ = m.Update(keyMsg("a"))
	m = updated.(Model)
	m.textInput.SetValue("下周的新目标")
	m.Update(keyMsg("enter"))

	// 验证目标创建在下一周
	now := time.Now()
	goals, _ := s.ListWeekGoals(store.WeekGoalFilter{
		Year: nextWeekYear(now), Week: nextWeekNum(now),
	})
	if len(goals) != 1 || goals[0].Title != "下周的新目标" {
		t.Errorf("目标应创建在下一周，该周目标列表 = %v", goals)
	}
	// 当前周不应被创建
	cy, cw := util.ISOWeek(now)
	curGoals, _ := s.ListWeekGoals(store.WeekGoalFilter{Year: cy, Week: cw})
	for _, g := range curGoals {
		if g.Title == "下周的新目标" {
			t.Error("目标不应创建在当前周")
		}
	}
}

func TestModel_AddYearGoal(t *testing.T) {
	s := newFeatureStore(t)
	m := withSize(NewModel(s), 100, 30)

	updated, _ := m.Update(keyMsg("4"))
	m = updated.(Model)
	updated, _ = m.Update(keyMsg("a"))
	m = updated.(Model)
	m.textInput.SetValue("新的年度目标")
	updated, _ = m.Update(keyMsg("enter"))
	m = updated.(Model)

	goals, _ := s.ListYearGoals(store.YearGoalFilter{Year: time.Now().Year()})
	if len(goals) != 2 {
		t.Errorf("应有 2 个年度目标，得到 %d", len(goals))
	}
}
