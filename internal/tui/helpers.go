package tui

import (
	"time"

	"github.com/charmbracelet/lipgloss"

	"goal-tracker/internal/model"
	"goal-tracker/internal/store"
	"goal-tracker/internal/util"
)

// utilNow 返回当前时间（便于测试时替换）。
func utilNow() time.Time {
	return time.Now()
}

// loadTasks 加载今日视图的任务列表（包含 pending 和 done，按状态排序：pending 在前）。
// TUI 中需要显示已完成的任务（带删除线），所以查询所有状态。
func loadTasks(s *store.Store) []model.Task {
	tasks, err := s.ListTasks(store.TaskFilter{Status: "all"})
	if err != nil {
		return nil
	}
	return tasks
}

// selectedTask 返回今日视图中当前选中的任务。
func (m Model) selectedTask() (model.Task, bool) {
	if m.activeTab != tabToday {
		return model.Task{}, false
	}
	tasks := loadTasks(m.store)
	if m.today.cursor < 0 || m.today.cursor >= len(tasks) {
		return model.Task{}, false
	}
	return tasks[m.today.cursor], true
}

// selectedTaskID 返回选中任务的 ID（today/week 都可以）。
func (m Model) selectedTaskID() (int64, bool) {
	t, ok := m.selectedTask()
	if !ok {
		return 0, false
	}
	return t.ID, true
}

// formatDateLabel 渲染截止日期标签（固定宽度，便于对齐）。
func formatDateLabel(due *time.Time, now time.Time) string {
	if due == nil {
		return mutedStr("无")
	}
	if util.IsOverdue(*due, now) {
		return boldStr(colorWarn, "⚠ "+util.FormatDate(*due))
	}
	if util.IsToday(*due, now) {
		return boldStr(colorPriorityMedium, "今天")
	}
	return coloredStr(colorInfo, util.FormatDate(*due))
}

// 以下小工具用 lipgloss 渲染带颜色的字符串。
func mutedStr(s string) string {
	return lipgloss.NewStyle().Foreground(colorMuted).Render(s)
}
func boldStr(c lipglossColor, s string) string {
	return lipgloss.NewStyle().Foreground(c).Bold(true).Render(s)
}
func coloredStr(c lipglossColor, s string) string {
	return lipgloss.NewStyle().Foreground(c).Render(s)
}

// lipglossColor 是 lipgloss.Color 的别名，便于在 helpers.go 中引用。
type lipglossColor = lipgloss.Color
