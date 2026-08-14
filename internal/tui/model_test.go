package tui

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"goal-tracker/internal/model"
	"goal-tracker/internal/store"
)

func newTUIStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "tui_test.db"))
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// TestModel_Init 验证 Model 能正常创建（AC-19 基础）
func TestModel_Init(t *testing.T) {
	s := newTUIStore(t)
	m := NewModel(s)
	if m.activeTab != tabToday {
		t.Error("默认 activeTab 应为 tabToday")
	}
}

// TestModel_WindowSize 验证能接收窗口大小消息
func TestModel_WindowSize(t *testing.T) {
	s := newTUIStore(t)
	m := NewModel(s)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m2 := updated.(Model)
	if m2.width != 100 || m2.height != 30 {
		t.Errorf("窗口尺寸 = %dx%d，应为 100x30", m2.width, m2.height)
	}
}

// TestModel_TabSwitch 验证 Tab 切换（AC-20）
func TestModel_TabSwitch(t *testing.T) {
	s := newTUIStore(t)
	m := NewModel(s)

	// 用数字键切换
	cases := []struct {
		key      string
		expected tabID
	}{
		{"2", tabWeek},
		{"3", tabQuarter},
		{"4", tabYear},
		{"1", tabToday},
	}

	for _, c := range cases {
		updated, _ := m.Update(keyMsg(c.key))
		m = updated.(Model)
		if m.activeTab != c.expected {
			t.Errorf("按 %q 后 activeTab = %v，应为 %v", c.key, m.activeTab, c.expected)
		}
	}
}

// TestModel_TabKey 验证 Tab 键循环切换（AC-20）
func TestModel_TabKey(t *testing.T) {
	s := newTUIStore(t)
	m := NewModel(s)

	// Tab 键：today → week → quarter → year → today
	for _, expected := range []tabID{tabWeek, tabQuarter, tabYear, tabToday} {
		updated, _ := m.Update(keyMsg("tab"))
		m = updated.(Model)
		if m.activeTab != expected {
			t.Errorf("Tab 后 activeTab = %v，应为 %v", m.activeTab, expected)
		}
	}
}

// TestModel_Quit 验证 q 键退出（AC-24）
func TestModel_Quit(t *testing.T) {
	s := newTUIStore(t)
	m := NewModel(s)

	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Error("按 q 应返回 tea.Quit 命令")
	}
}

// TestModel_CursorMove 验证光标移动（AC-21）
func TestModel_CursorMove(t *testing.T) {
	s := newTUIStore(t)

	// 建几个任务
	s.CreateTask(store.CreateTaskInput{Title: "t1"})
	s.CreateTask(store.CreateTaskInput{Title: "t2"})
	s.CreateTask(store.CreateTaskInput{Title: "t3"})

	m := NewModel(s)
	if m.today.cursor != 0 {
		t.Error("初始 cursor 应为 0")
	}

	// 按 j / 下移
	updated, _ := m.Update(keyMsg("j"))
	m = updated.(Model)
	if m.today.cursor != 1 {
		t.Errorf("按 j 后 cursor = %d，应为 1", m.today.cursor)
	}

	updated, _ = m.Update(keyMsg("j"))
	m = updated.(Model)
	if m.today.cursor != 2 {
		t.Errorf("按 j 后 cursor = %d，应为 2", m.today.cursor)
	}

	// 按 k / 上移
	updated, _ = m.Update(keyMsg("k"))
	m = updated.(Model)
	if m.today.cursor != 1 {
		t.Errorf("按 k 后 cursor = %d，应为 1", m.today.cursor)
	}

	// 在 cursor=0 时按 k，不应变成负数
	m.today.cursor = 0
	updated, _ = m.Update(keyMsg("k"))
	m = updated.(Model)
	if m.today.cursor != 0 {
		t.Errorf("cursor=0 按 k 后应为 0，得到 %d", m.today.cursor)
	}
}

// TestModel_SpaceToggleTask 验证 Space 切换任务完成状态（AC-22）
func TestModel_SpaceToggleTask(t *testing.T) {
	s := newTUIStore(t)
	task, _ := s.CreateTask(store.CreateTaskInput{Title: "测试任务"})
	if task.Status != model.TaskStatusPending {
		t.Fatal("初始状态应为 pending")
	}

	m := NewModel(s)
	// 光标在 0（即第一个任务）
	updated, _ := m.Update(keyMsg(" "))
	m = updated.(Model)

	// 验证任务已变成 done
	updated2, _ := s.GetTask(task.ID)
	if updated2.Status != model.TaskStatusDone {
		t.Errorf("按 Space 后任务状态 = %q，应为 done", updated2.Status)
	}

	// 再按一次 Space，恢复 pending
	updated, _ = m.Update(keyMsg(" "))
	m = updated.(Model)
	updated2, _ = s.GetTask(task.ID)
	if updated2.Status != model.TaskStatusPending {
		t.Errorf("再按 Space 后任务状态 = %q，应为 pending", updated2.Status)
	}
}

// TestModel_AddTask 验证 a 键添加任务（AC-23）
func TestModel_AddTask(t *testing.T) {
	s := newTUIStore(t)
	m := NewModel(s)

	// 按 a 进入输入模式
	updated, _ := m.Update(keyMsg("a"))
	m = updated.(Model)
	if m.mode != inputAdd {
		t.Fatal("按 a 后应进入 inputAdd 模式")
	}

	// 模拟输入标题
	m.textInput.SetValue("新任务测试")

	// 按 Enter 确认
	updated, _ = m.Update(keyMsg("enter"))
	m = updated.(Model)

	// 验证任务已添加
	tasks, _ := s.ListTasks(store.TaskFilter{Status: "all"})
	if len(tasks) != 1 {
		t.Fatalf("任务数 = %d，应为 1", len(tasks))
	}
	if tasks[0].Title != "新任务测试" {
		t.Errorf("任务标题 = %q", tasks[0].Title)
	}
	if m.mode != inputNone {
		t.Error("确认后应退出输入模式")
	}
}

// TestModel_DeleteTask 验证 x 键删除任务
func TestModel_DeleteTask(t *testing.T) {
	s := newTUIStore(t)
	task, _ := s.CreateTask(store.CreateTaskInput{Title: "待删任务"})

	m := NewModel(s)
	// 光标在第一个任务上
	updated, _ := m.Update(keyMsg("x"))
	m = updated.(Model)
	if m.mode != inputDelete {
		t.Fatal("按 x 后应进入 inputDelete 模式")
	}

	// 输入 y 确认
	m.textInput.SetValue("y")
	updated, _ = m.Update(keyMsg("enter"))
	m = updated.(Model)

	// 验证已删除
	got, _ := s.GetTask(task.ID)
	if got != nil {
		t.Error("任务应已被删除")
	}
}

// TestModel_DeleteCancel 验证删除时输入 n 取消
func TestModel_DeleteCancel(t *testing.T) {
	s := newTUIStore(t)
	task, _ := s.CreateTask(store.CreateTaskInput{Title: "保留任务"})

	m := NewModel(s)
	m.Update(keyMsg("x"))
	m.textInput.SetValue("n")
	updated, _ := m.Update(keyMsg("enter"))
	m = updated.(Model)

	got, _ := s.GetTask(task.ID)
	if got == nil {
		t.Error("任务不应被删除")
	}
	if m.mode != inputNone {
		t.Error("取消后应退出输入模式")
	}
}

// TestModel_RenderTodayView 验证渲染不 panic
func TestModel_RenderTodayView(t *testing.T) {
	s := newTUIStore(t)
	s.CreateTask(store.CreateTaskInput{Title: "任务1"})
	s.CreateTask(store.CreateTaskInput{Title: "任务2"})

	m := NewModel(s)
	// 设置窗口大小，让 View 能正常渲染
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)

	out := m.View()
	if out == "" {
		t.Error("View() 不应返回空字符串")
	}
}

// TestModel_RenderAllTabs 验证所有 Tab 都能正常渲染
func TestModel_RenderAllTabs(t *testing.T) {
	s := newTUIStore(t)
	// 建一些数据让视图非空
	s.CreateYearGoal(store.CreateYearGoalInput{Title: "年度", Year: 2026})
	s.CreateQuarterGoal(store.CreateQuarterGoalInput{Title: "季度", Year: 2026, Quarter: 3})
	s.CreateWeekGoal(store.CreateWeekGoalInput{Title: "周", Year: 2026, Week: 33})
	s.CreateTask(store.CreateTaskInput{Title: "任务"})

	m := NewModel(s)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	for _, tab := range []tabID{tabToday, tabWeek, tabQuarter, tabYear} {
		m.activeTab = tab
		out := m.View()
		if out == "" {
			t.Errorf("Tab %d 的 View 为空", tab)
		}
	}
}

// ---------- 辅助 ----------

// keyMsg 构造一个 tea.KeyMsg（简化测试）
func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}
