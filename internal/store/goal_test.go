package store

import (
	"testing"

	"goal-tracker/internal/model"
)

// ===== 周目标测试 =====

func TestWeekGoal_CRUD(t *testing.T) {
	s := newTestStore(t)

	// Create
	w, err := s.CreateWeekGoal(CreateWeekGoalInput{
		Title: "完成方案A", Year: 2026, Week: 33,
	})
	if err != nil {
		t.Fatalf("CreateWeekGoal: %v", err)
	}
	if w.Title != "完成方案A" || w.Year != 2026 || w.Week != 33 {
		t.Errorf("字段不对: %+v", w)
	}
	if w.Status != model.WeekGoalStatusActive {
		t.Errorf("初始状态应为 active")
	}

	// Get
	got, err := s.GetWeekGoal(w.ID)
	if err != nil || got == nil {
		t.Fatalf("GetWeekGoal: %v %v", got, err)
	}

	// Update
	newTitle := "新标题"
	updated, err := s.UpdateWeekGoal(w.ID, UpdateWeekGoalInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateWeekGoal: %v", err)
	}
	if updated.Title != "新标题" {
		t.Errorf("更新后标题 = %q", updated.Title)
	}

	// SetStatus done
	updated, _ = s.SetWeekGoalStatus(w.ID, model.WeekGoalStatusCompleted)
	if updated.Status != model.WeekGoalStatusCompleted {
		t.Errorf("状态应为 completed")
	}

	// Delete
	deleted, _ := s.DeleteWeekGoal(w.ID)
	if !deleted {
		t.Error("应删除成功")
	}
}

func TestWeekGoal_EmptyTitle(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CreateWeekGoal(CreateWeekGoalInput{Title: "  ", Year: 2026, Week: 33})
	if err == nil {
		t.Error("空标题应报错")
	}
}

func TestWeekGoal_InvalidWeek(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CreateWeekGoal(CreateWeekGoalInput{Title: "x", Year: 2026, Week: 99})
	if err == nil {
		t.Error("无效周数应报错")
	}
}

// TestWeekGoal_Progress 进度计算（AC-18 核心）
// 进度 = 已完成任务数 / 总任务数
func TestWeekGoal_Progress(t *testing.T) {
	s := newTestStore(t)

	w, _ := s.CreateWeekGoal(CreateWeekGoalInput{Title: "周目标", Year: 2026, Week: 33})

	// 无任务时 HasProgress 应为 false
	list, err := s.ListWeekGoalsWithProgress(WeekGoalFilter{Year: 2026, Week: 33})
	if err != nil {
		t.Fatalf("ListWithProgress: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("应有 1 个周目标，得到 %d", len(list))
	}
	if list[0].HasProgress {
		t.Error("无任务时 HasProgress 应为 false")
	}
	if list[0].Progress() != -1 {
		t.Errorf("无任务时 Progress 应为 -1")
	}

	// 建 4 个任务，完成 3 个 → 进度 75%
	t1, _ := s.CreateTask(CreateTaskInput{Title: "t1", WeekGoalID: &w.ID})
	t2, _ := s.CreateTask(CreateTaskInput{Title: "t2", WeekGoalID: &w.ID})
	t3, _ := s.CreateTask(CreateTaskInput{Title: "t3", WeekGoalID: &w.ID})
	s.CreateTask(CreateTaskInput{Title: "t4", WeekGoalID: &w.ID})
	s.SetTaskStatus(t1.ID, model.TaskStatusDone)
	s.SetTaskStatus(t2.ID, model.TaskStatusDone)
	s.SetTaskStatus(t3.ID, model.TaskStatusDone)

	list, _ = s.ListWeekGoalsWithProgress(WeekGoalFilter{Year: 2026, Week: 33})
	if list[0].TaskTotal != 4 {
		t.Errorf("TaskTotal = %d，应为 4", list[0].TaskTotal)
	}
	if list[0].TaskDone != 3 {
		t.Errorf("TaskDone = %d，应为 3", list[0].TaskDone)
	}
	if list[0].Progress() != 75 {
		t.Errorf("Progress = %d%%，应为 75%%", list[0].Progress())
	}
}

func TestWeekGoal_FilterByQuarter(t *testing.T) {
	s := newTestStore(t)

	// 建季度目标
	res, _ := s.db.Exec(`INSERT INTO quarter_goals (title, year, quarter) VALUES ('Q3', 2026, 3)`)
	qID, _ := res.LastInsertId()

	// 两个周目标，一个关联季度目标
	s.CreateWeekGoal(CreateWeekGoalInput{Title: "w1", Year: 2026, Week: 33, QuarterGoalID: &qID})
	s.CreateWeekGoal(CreateWeekGoalInput{Title: "w2", Year: 2026, Week: 34})

	list, _ := s.ListWeekGoals(WeekGoalFilter{QuarterGoalID: &qID})
	if len(list) != 1 || list[0].Title != "w1" {
		t.Errorf("按季度过滤应只返回 w1，得到 %d 项", len(list))
	}
}

// ===== 季度目标测试 =====

func TestQuarterGoal_CRUD(t *testing.T) {
	s := newTestStore(t)

	q, err := s.CreateQuarterGoal(CreateQuarterGoalInput{
		Title: "Q3目标", Year: 2026, Quarter: 3, Description: "测试",
	})
	if err != nil {
		t.Fatalf("CreateQuarterGoal: %v", err)
	}
	if q.Quarter != 3 || q.Description != "测试" {
		t.Errorf("字段不对: %+v", q)
	}

	// Edit
	newTitle := "新Q3"
	updated, _ := s.UpdateQuarterGoal(q.ID, UpdateQuarterGoalInput{Title: &newTitle})
	if updated.Title != "新Q3" {
		t.Errorf("更新后 = %q", updated.Title)
	}

	// Delete
	ok, _ := s.DeleteQuarterGoal(q.ID)
	if !ok {
		t.Error("删除应成功")
	}
}

func TestQuarterGoal_InvalidQuarter(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CreateQuarterGoal(CreateQuarterGoalInput{Title: "x", Year: 2026, Quarter: 5})
	if err == nil {
		t.Error("无效季度应报错")
	}
}

// TestQuarterGoal_Progress 进度 = 已完成周目标 / 总周目标
func TestQuarterGoal_Progress(t *testing.T) {
	s := newTestStore(t)

	q, _ := s.CreateQuarterGoal(CreateQuarterGoalInput{Title: "Q3", Year: 2026, Quarter: 3})

	// 4 个周目标，完成 2 个 → 50%
	w1, _ := s.CreateWeekGoal(CreateWeekGoalInput{Title: "w1", Year: 2026, Week: 33, QuarterGoalID: &q.ID})
	w2, _ := s.CreateWeekGoal(CreateWeekGoalInput{Title: "w2", Year: 2026, Week: 34, QuarterGoalID: &q.ID})
	s.CreateWeekGoal(CreateWeekGoalInput{Title: "w3", Year: 2026, Week: 35, QuarterGoalID: &q.ID})
	s.CreateWeekGoal(CreateWeekGoalInput{Title: "w4", Year: 2026, Week: 36, QuarterGoalID: &q.ID})
	s.SetWeekGoalStatus(w1.ID, model.WeekGoalStatusCompleted)
	s.SetWeekGoalStatus(w2.ID, model.WeekGoalStatusCompleted)

	list, _ := s.ListQuarterGoalsWithProgress(QuarterGoalFilter{Year: 2026, Quarter: 3})
	if len(list) != 1 {
		t.Fatalf("应有 1 个季度目标，得到 %d", len(list))
	}
	if list[0].WeekTotal != 4 || list[0].WeekDone != 2 {
		t.Errorf("WeekTotal=%d WeekDone=%d，应为 4/2", list[0].WeekTotal, list[0].WeekDone)
	}
	if list[0].Progress() != 50 {
		t.Errorf("Progress = %d%%，应为 50%%", list[0].Progress())
	}
}

// ===== 年度目标测试 =====

func TestYearGoal_CRUD(t *testing.T) {
	s := newTestStore(t)

	y, err := s.CreateYearGoal(CreateYearGoalInput{
		Title: "2026年度目标", Year: 2026, Description: "重要",
	})
	if err != nil {
		t.Fatalf("CreateYearGoal: %v", err)
	}
	if y.Year != 2026 || y.Description != "重要" {
		t.Errorf("字段不对: %+v", y)
	}

	// Edit
	newTitle := "新年度"
	u, _ := s.UpdateYearGoal(y.ID, UpdateYearGoalInput{Title: &newTitle})
	if u.Title != "新年度" {
		t.Errorf("更新后 = %q", u.Title)
	}

	// Delete
	ok, _ := s.DeleteYearGoal(y.ID)
	if !ok {
		t.Error("删除应成功")
	}
}

func TestYearGoal_InvalidYear(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CreateYearGoal(CreateYearGoalInput{Title: "x", Year: 1990})
	if err == nil {
		t.Error("无效年份应报错")
	}
}

// TestYearGoal_Progress 进度 = 已完成季度目标 / 总季度目标
func TestYearGoal_Progress(t *testing.T) {
	s := newTestStore(t)

	y, _ := s.CreateYearGoal(CreateYearGoalInput{Title: "年度", Year: 2026})

	// 4 个季度目标，完成 1 个 → 25%
	q1, _ := s.CreateQuarterGoal(CreateQuarterGoalInput{Title: "Q1", Year: 2026, Quarter: 1, YearGoalID: &y.ID})
	s.CreateQuarterGoal(CreateQuarterGoalInput{Title: "Q2", Year: 2026, Quarter: 2, YearGoalID: &y.ID})
	s.CreateQuarterGoal(CreateQuarterGoalInput{Title: "Q3", Year: 2026, Quarter: 3, YearGoalID: &y.ID})
	s.CreateQuarterGoal(CreateQuarterGoalInput{Title: "Q4", Year: 2026, Quarter: 4, YearGoalID: &y.ID})
	s.SetQuarterGoalStatus(q1.ID, model.QuarterGoalStatusCompleted)

	list, _ := s.ListYearGoalsWithProgress(YearGoalFilter{Year: 2026})
	if len(list) != 1 {
		t.Fatalf("应有 1 个年度目标，得到 %d", len(list))
	}
	if list[0].QuarterTotal != 4 || list[0].QuarterDone != 1 {
		t.Errorf("QuarterTotal=%d Done=%d，应为 4/1", list[0].QuarterTotal, list[0].QuarterDone)
	}
	if list[0].Progress() != 25 {
		t.Errorf("Progress = %d%%，应为 25%%", list[0].Progress())
	}
}

// TestHierarchy_FullDrillDown 完整层级：年→季→周→任务
func TestHierarchy_FullDrillDown(t *testing.T) {
	s := newTestStore(t)

	// 层级：年度 → 季度 → 周 → 任务
	y, _ := s.CreateYearGoal(CreateYearGoalInput{Title: "年度", Year: 2026})
	q, _ := s.CreateQuarterGoal(CreateQuarterGoalInput{Title: "季度", Year: 2026, Quarter: 3, YearGoalID: &y.ID})
	w, _ := s.CreateWeekGoal(CreateWeekGoalInput{Title: "周", Year: 2026, Week: 33, QuarterGoalID: &q.ID})

	// 2 个任务，完成 1 个
	t1, _ := s.CreateTask(CreateTaskInput{Title: "任务1", WeekGoalID: &w.ID})
	s.CreateTask(CreateTaskInput{Title: "任务2", WeekGoalID: &w.ID})
	s.SetTaskStatus(t1.ID, model.TaskStatusDone)

	// 验证层级进度传导：周 50% → 季度 0%（因为周未 completed）→ 年 0%
	wList, _ := s.ListWeekGoalsWithProgress(WeekGoalFilter{Year: 2026, Week: 33})
	if wList[0].Progress() != 50 {
		t.Errorf("周目标进度 = %d%%，应为 50%%", wList[0].Progress())
	}
	qList, _ := s.ListQuarterGoalsWithProgress(QuarterGoalFilter{Year: 2026, Quarter: 3})
	if qList[0].Progress() != 0 {
		t.Errorf("季度目标进度 = %d%%，应为 0%%（周目标未标记完成）", qList[0].Progress())
	}

	// 把周目标标记完成 → 季度进度变为 100%
	s.SetWeekGoalStatus(w.ID, model.WeekGoalStatusCompleted)
	qList, _ = s.ListQuarterGoalsWithProgress(QuarterGoalFilter{Year: 2026, Quarter: 3})
	if qList[0].Progress() != 100 {
		t.Errorf("季度目标进度 = %d%%，应为 100%%", qList[0].Progress())
	}

	// 把季度目标标记完成 → 年度进度变为 100%
	s.SetQuarterGoalStatus(q.ID, model.QuarterGoalStatusCompleted)
	yList, _ := s.ListYearGoalsWithProgress(YearGoalFilter{Year: 2026})
	if yList[0].Progress() != 100 {
		t.Errorf("年度目标进度 = %d%%，应为 100%%", yList[0].Progress())
	}
}
