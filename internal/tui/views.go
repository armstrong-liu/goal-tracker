package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"goal-tracker/internal/model"
	"goal-tracker/internal/store"
	"goal-tracker/internal/util"
)

// ---------- 今日任务视图 ----------

// taskColumns 定义今日任务表格的列。
// 改变列只需改这里，渲染逻辑完全不用动。
var taskColumns = []tableColumn{
	{Title: "状态", Width: 5},
	{Title: "ID", Width: 5},
	{Title: "任务", Width: 28},
	{Title: "截止", Width: 12},
	{Title: "优先级", Width: 6},
	{Title: "关联", Width: 8},
}

func (m Model) renderTodayView() string {
	now := utilNow()
	tasks := loadTasks(m.store)

	// 限制光标范围
	if m.today.cursor >= len(tasks) {
		m.today.cursor = len(tasks) - 1
	}
	if m.today.cursor < 0 {
		m.today.cursor = 0
	}

	var b strings.Builder

	// 视图标题
	title := lipgloss.NewStyle().Bold(true).Render("📋 待办任务")
	count := mutedStr(fmt.Sprintf("（%d 项）", len(tasks)))
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, title, " ", count) + "\n\n")

	if len(tasks) == 0 {
		b.WriteString(styleHint.Render("  🎉 没有待办任务，按 [a] 添加一个吧"))
		return wrapInContentBox(b.String(), m.width)
	}

	// 构造数据行（每个单元格的"原始值"，渲染交给 renderTable）
	rows := make([][]string, 0, len(tasks))
	for _, t := range tasks {
		ref := ""
		if t.WeekGoalID != nil {
			ref = fmt.Sprintf("周#%d", *t.WeekGoalID)
		}
		rows = append(rows, []string{
			statusIcon(t.Status),
			fmt.Sprintf("#%d", t.ID),
			t.Title,
			formatDateLabel(t.DueDate, now),
			priorityBadge(t.Priority),
			ref,
		})
	}

	// 一步渲染完整表格（表头 + 分隔线 + 数据行 + 选中高亮）
	b.WriteString(renderTable(taskColumns, rows, m.today.cursor, defaultTableStyle))

	return wrapInContentBox(b.String(), m.width)
}

// ---------- 周目标视图 ----------

func (m Model) renderWeekView() string {
	now := utilNow()
	y, w := util.ISOWeek(now)

	weekGoals, err := m.store.ListWeekGoalsWithProgress(store.WeekGoalFilter{Year: y, Week: w})
	if err != nil || len(weekGoals) == 0 {
		return wrapInContentBox(renderEmptyBody("📅 本周目标",
			"本周还没有目标",
			"用 'gt week add \"目标\"' 在命令行添加"), m.width)
	}

	// 光标范围
	if m.week.cursor >= len(weekGoals) {
		m.week.cursor = len(weekGoals) - 1
	}
	if m.week.cursor < 0 {
		m.week.cursor = 0
	}

	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("📅 %s 周目标", util.WeekLabel(y, w)))
	count := mutedStr(fmt.Sprintf("（%d 项）", len(weekGoals)))
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, title, " ", count) + "\n\n")

	for i, wg := range weekGoals {
		selected := i == m.week.cursor
		b.WriteString(renderWeekGoalRow(wg, selected))
		b.WriteString("\n")

		// 选中时展开任务列表
		if selected && wg.HasProgress {
			tasks, _ := m.store.ListTasks(store.TaskFilter{WeekGoalID: &wg.ID})
			for _, t := range tasks {
				mark := statusIcon(t.Status)
				tTitle := t.Title
				if t.Status == model.TaskStatusDone {
					tTitle = lipgloss.NewStyle().Strikethrough(true).Foreground(colorMuted).Render(tTitle)
				}
				b.WriteString(fmt.Sprintf("        %s %s\n", mark, tTitle))
			}
		}
	}

	return wrapInContentBox(b.String(), m.width)
}

func renderWeekGoalRow(wg store.WeekGoalWithProgress, selected bool) string {
	icon := "🎯"
	if wg.Status == model.WeekGoalStatusCompleted {
		icon = "✅"
	}
	progress := progressBar(wg.Progress(), 10)
	// 只有存在关联任务时才显示 (done/total)
	detail := ""
	if wg.HasProgress {
		detail = mutedStr(fmt.Sprintf("(%d/%d)", wg.TaskDone, wg.TaskTotal))
	}

	content := fmt.Sprintf("%s #%-3d %s    %s %s",
		icon, wg.ID, fitWidth(wg.Title, 28), progress, detail)

	if selected {
		return " " + styleItemSelected.Render("▶ " + content)
	}
	return "   " + content
}

// ---------- 季度目标视图 ----------

func (m Model) renderQuarterView() string {
	now := utilNow()
	y := now.Year()
	q := util.CurrentQuarter(now)

	quarterGoals, err := m.store.ListQuarterGoalsWithProgress(store.QuarterGoalFilter{Year: y, Quarter: q})
	if err != nil || len(quarterGoals) == 0 {
		return wrapInContentBox(renderEmptyBody("🏆 季度目标",
			"本季度还没有目标",
			"用 'gt quarter add \"目标\"' 在命令行添加"), m.width)
	}

	if m.quarter.cursor >= len(quarterGoals) {
		m.quarter.cursor = len(quarterGoals) - 1
	}
	if m.quarter.cursor < 0 {
		m.quarter.cursor = 0
	}

	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("🏆 %s 季度目标", util.QuarterLabel(y, q)))
	count := mutedStr(fmt.Sprintf("（%d 项）", len(quarterGoals)))
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, title, " ", count) + "\n\n")

	for i, qg := range quarterGoals {
		selected := i == m.quarter.cursor
		b.WriteString(renderQuarterGoalRow(qg, selected))
		b.WriteString("\n")
	}

	return wrapInContentBox(b.String(), m.width)
}

func renderQuarterGoalRow(qg store.QuarterGoalWithProgress, selected bool) string {
	icon := goalStatusIcon(qg.Status)
	progress := progressBar(qg.Progress(), 10)
	detail := ""
	if qg.HasProgress {
		detail = mutedStr(fmt.Sprintf("(%d/%d 周)", qg.WeekDone, qg.WeekTotal))
	}

	content := fmt.Sprintf("%s #%-3d %s    %s %s",
		icon, qg.ID, fitWidth(qg.Title, 28), progress, detail)

	if selected {
		return " " + styleItemSelected.Render("▶ " + content)
	}
	return "   " + content
}

// ---------- 年度目标视图 ----------

func (m Model) renderYearView() string {
	now := utilNow()
	y := now.Year()

	yearGoals, err := m.store.ListYearGoalsWithProgress(store.YearGoalFilter{Year: y})
	if err != nil || len(yearGoals) == 0 {
		return wrapInContentBox(renderEmptyBody("🎯 年度目标",
			"本年还没有目标",
			"用 'gt year add \"目标\"' 在命令行添加"), m.width)
	}

	if m.year.cursor >= len(yearGoals) {
		m.year.cursor = len(yearGoals) - 1
	}
	if m.year.cursor < 0 {
		m.year.cursor = 0
	}

	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("🎯 %d 年度目标", y))
	count := mutedStr(fmt.Sprintf("（%d 项）", len(yearGoals)))
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, title, " ", count) + "\n\n")

	for i, yg := range yearGoals {
		selected := i == m.year.cursor
		b.WriteString(renderYearGoalRow(yg, selected))
		b.WriteString("\n")
	}

	return wrapInContentBox(b.String(), m.width)
}

func renderYearGoalRow(yg store.YearGoalWithProgress, selected bool) string {
	icon := goalStatusIcon(yg.Status)
	progress := progressBar(yg.Progress(), 10)
	detail := ""
	if yg.HasProgress {
		detail = mutedStr(fmt.Sprintf("(%d/%d 季度)", yg.QuarterDone, yg.QuarterTotal))
	}

	content := fmt.Sprintf("%s #%-3d %s    %s %s",
		icon, yg.ID, fitWidth(yg.Title, 28), progress, detail)

	if selected {
		return " " + styleItemSelected.Render("▶ " + content)
	}
	return "   " + content
}

// ---------- 通用渲染辅助 ----------

// goalStatusIcon 返回目标的图标（根据状态）。
func goalStatusIcon(status string) string {
	switch status {
	case "completed":
		return "✅"
	case "archived":
		return "📦"
	default:
		return "🎯"
	}
}

// renderEmptyBody 渲染空状态的内容（不含外层边框）。
func renderEmptyBody(title, msg, hint string) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(title) + "\n\n")
	b.WriteString(styleHint.Render("  " + msg) + "\n\n")
	b.WriteString(mutedStr("  " + hint) + "\n")
	return b.String()
}

// wrapInContentBox 把内容用圆角边框包围，宽度对齐终端。
func wrapInContentBox(content string, termWidth int) string {
	// 内容区宽度 = 终端宽度 - 2（边框） - 2（padding）
	boxWidth := termWidth - 4
	if boxWidth < 40 {
		boxWidth = 40
	}
	return styleContentBox.Width(boxWidth).Render(content)
}
