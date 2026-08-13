package store

import (
	"path/filepath"
	"testing"
	"time"

	"goal-tracker/internal/model"
)

// newTestStore 创建一个基于临时目录的 Store。
func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateTask_Basic(t *testing.T) {
	s := newTestStore(t)
	due := time.Date(2026, 8, 15, 0, 0, 0, 0, time.Local)

	task, err := s.CreateTask(CreateTaskInput{
		Title:    "写周报",
		DueDate:  &due,
		Priority: model.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("CreateTask 失败: %v", err)
	}
	if task.ID == 0 {
		t.Error("ID 不应为 0")
	}
	if task.Title != "写周报" {
		t.Errorf("Title = %q", task.Title)
	}
	if task.Priority != model.PriorityHigh {
		t.Errorf("Priority = %q", task.Priority)
	}
	if task.Status != model.TaskStatusPending {
		t.Errorf("Status = %q，应为 pending", task.Status)
	}
	if task.DueDate == nil || !task.DueDate.Equal(due) {
		t.Errorf("DueDate 不正确: %v", task.DueDate)
	}
}

func TestCreateTask_DefaultPriority(t *testing.T) {
	s := newTestStore(t)
	task, err := s.CreateTask(CreateTaskInput{Title: "测试"})
	if err != nil {
		t.Fatalf("CreateTask 失败: %v", err)
	}
	if task.Priority != model.PriorityMedium {
		t.Errorf("默认 Priority = %q，应为 medium", task.Priority)
	}
}

func TestCreateTask_EmptyTitle(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CreateTask(CreateTaskInput{Title: "   "})
	if err == nil {
		t.Error("空标题应返回错误")
	}
}

func TestCreateTask_InvalidPriority(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CreateTask(CreateTaskInput{Title: "测试", Priority: "urgent"})
	if err == nil {
		t.Error("无效优先级应返回错误")
	}
}

func TestGetTask_NotExist(t *testing.T) {
	s := newTestStore(t)
	task, err := s.GetTask(9999)
	if err != nil {
		t.Fatalf("GetTask 错误: %v", err)
	}
	if task != nil {
		t.Error("不存在的任务应返回 nil")
	}
}

func TestListTasks_StatusFilter(t *testing.T) {
	s := newTestStore(t)
	// 创建 2 个 pending，1 个 done
	s.CreateTask(CreateTaskInput{Title: "p1"})
	s.CreateTask(CreateTaskInput{Title: "p2"})
	done, _ := s.CreateTask(CreateTaskInput{Title: "d1"})
	s.SetTaskStatus(done.ID, model.TaskStatusDone)

	pending, err := s.ListTasks(TaskFilter{Status: "pending"})
	if err != nil {
		t.Fatalf("ListTasks pending 失败: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("pending 任务数 = %d，应为 2", len(pending))
	}

	doneTasks, err := s.ListTasks(TaskFilter{Status: "done"})
	if err != nil {
		t.Fatalf("ListTasks done 失败: %v", err)
	}
	if len(doneTasks) != 1 {
		t.Errorf("done 任务数 = %d，应为 1", len(doneTasks))
	}

	all, _ := s.ListTasks(TaskFilter{Status: "all"})
	if len(all) != 3 {
		t.Errorf("all 任务数 = %d，应为 3", len(all))
	}
}

func TestListTasks_DueBefore(t *testing.T) {
	s := newTestStore(t)
	now := time.Date(2026, 8, 12, 10, 0, 0, 0, time.Local)

	// 过期的任务（8/10）
	past := time.Date(2026, 8, 10, 0, 0, 0, 0, time.Local)
	s.CreateTask(CreateTaskInput{Title: "过期", DueDate: &past})

	// 未来的任务（8/15）
	future := time.Date(2026, 8, 15, 0, 0, 0, 0, time.Local)
	s.CreateTask(CreateTaskInput{Title: "未来", DueDate: &future})

	// 无截止日期
	s.CreateTask(CreateTaskInput{Title: "无截止"})

	overdue, err := s.ListTasks(TaskFilter{Status: "all", DueBefore: &now})
	if err != nil {
		t.Fatalf("ListTasks DueBefore 失败: %v", err)
	}
	if len(overdue) != 1 {
		t.Errorf("过期任务数 = %d，应为 1", len(overdue))
	}
	if len(overdue) > 0 && overdue[0].Title != "过期" {
		t.Errorf("过期任务 Title = %q，应为 %q", overdue[0].Title, "过期")
	}
}

func TestSetTaskStatus(t *testing.T) {
	s := newTestStore(t)
	task, _ := s.CreateTask(CreateTaskInput{Title: "测试"})

	// 标记完成
	updated, err := s.SetTaskStatus(task.ID, model.TaskStatusDone)
	if err != nil {
		t.Fatalf("SetTaskStatus done 失败: %v", err)
	}
	if updated.Status != model.TaskStatusDone {
		t.Errorf("Status = %q，应为 done", updated.Status)
	}

	// 标记回 pending
	updated, _ = s.SetTaskStatus(task.ID, model.TaskStatusPending)
	if updated.Status != model.TaskStatusPending {
		t.Errorf("Status = %q，应为 pending", updated.Status)
	}

	// 不存在的任务
	_, err = s.SetTaskStatus(9999, model.TaskStatusDone)
	if err == nil {
		t.Error("操作不存在的任务应报错")
	}
}

func TestUpdateTask(t *testing.T) {
	s := newTestStore(t)
	task, _ := s.CreateTask(CreateTaskInput{Title: "原标题", Priority: model.PriorityLow})

	newTitle := "新标题"
	updated, err := s.UpdateTask(task.ID, UpdateTaskInput{
		Title:    &newTitle,
		Priority: ptrString(model.PriorityHigh),
	})
	if err != nil {
		t.Fatalf("UpdateTask 失败: %v", err)
	}
	if updated.Title != "新标题" {
		t.Errorf("Title = %q", updated.Title)
	}
	if updated.Priority != model.PriorityHigh {
		t.Errorf("Priority = %q", updated.Priority)
	}
}

func TestUpdateTask_ClearDueDate(t *testing.T) {
	s := newTestStore(t)
	due := time.Date(2026, 8, 15, 0, 0, 0, 0, time.Local)
	task, _ := s.CreateTask(CreateTaskInput{Title: "测试", DueDate: &due})

	updated, err := s.UpdateTask(task.ID, UpdateTaskInput{ClearDue: true})
	if err != nil {
		t.Fatalf("UpdateTask ClearDue 失败: %v", err)
	}
	if updated.DueDate != nil {
		t.Errorf("清空后 DueDate 应为 nil，得到 %v", updated.DueDate)
	}
}

func TestUpdateTask_NotExist(t *testing.T) {
	s := newTestStore(t)
	newTitle := "x"
	_, err := s.UpdateTask(9999, UpdateTaskInput{Title: &newTitle})
	if err == nil {
		t.Error("更新不存在的任务应报错")
	}
}

func TestDeleteTask(t *testing.T) {
	s := newTestStore(t)
	task, _ := s.CreateTask(CreateTaskInput{Title: "待删"})

	deleted, err := s.DeleteTask(task.ID)
	if err != nil {
		t.Fatalf("DeleteTask 失败: %v", err)
	}
	if !deleted {
		t.Error("应返回 true")
	}

	// 确认查不到了
	got, _ := s.GetTask(task.ID)
	if got != nil {
		t.Error("删除后应查不到")
	}

	// 再删一次
	deleted, _ = s.DeleteTask(task.ID)
	if deleted {
		t.Error("重复删除应返回 false")
	}
}

func TestLinkTask(t *testing.T) {
	s := newTestStore(t)

	// 先建一个周目标
	res, _ := s.db.Exec(
		`INSERT INTO week_goals (title, year, week) VALUES (?, ?, ?)`,
		"周目标", 2026, 33,
	)
	weekID, _ := res.LastInsertId()

	task, _ := s.CreateTask(CreateTaskInput{Title: "测试"})
	linked, err := s.LinkTask(task.ID, weekID)
	if err != nil {
		t.Fatalf("LinkTask 失败: %v", err)
	}
	if linked.WeekGoalID == nil || *linked.WeekGoalID != weekID {
		t.Errorf("WeekGoalID = %v，应为 %d", linked.WeekGoalID, weekID)
	}

	// 关联不存在的周目标
	_, err = s.LinkTask(task.ID, 9999)
	if err == nil {
		t.Error("关联不存在的周目标应报错")
	}
}

func TestCountTasks(t *testing.T) {
	s := newTestStore(t)
	s.CreateTask(CreateTaskInput{Title: "p1"})
	s.CreateTask(CreateTaskInput{Title: "p2"})
	done, _ := s.CreateTask(CreateTaskInput{Title: "d1"})
	s.SetTaskStatus(done.ID, model.TaskStatusDone)

	all, _ := s.CountTasks(TaskFilter{Status: "all"})
	if all != 3 {
		t.Errorf("总数 = %d，应为 3", all)
	}
	pending, _ := s.CountTasks(TaskFilter{Status: "pending"})
	if pending != 2 {
		t.Errorf("pending = %d，应为 2", pending)
	}
}

func ptrString(s string) *string { return &s }
