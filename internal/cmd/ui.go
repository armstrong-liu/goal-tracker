// Package cmd 内的 ui.go：CLI 输出美化工具。
// 集中定义颜色、图标、表格渲染等，供 task/today 等命令复用。
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"goal-tracker/internal/model"
	"goal-tracker/internal/util"
)

// ---------- 样式定义 ----------

var (
	// 主题色
	colorTitle   = lipgloss.Color("63")  // 紫色
	colorSuccess = lipgloss.Color("36")  // 青绿
	colorMuted   = lipgloss.Color("245") // 灰色
	colorRed     = lipgloss.Color("203") // 浅红
	colorYellow  = lipgloss.Color("221") // 浅黄
	colorGreen   = lipgloss.Color("156") // 浅绿
	colorBlue    = lipgloss.Color("117") // 浅蓝

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorTitle)

	styleSubtitle = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	styleError = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	styleDone = lipgloss.NewStyle().
			Foreground(colorMuted).
			Strikethrough(true)
)

// ---------- 通用渲染函数 ----------

// PrintTitle 打印一个带样式的标题。
func PrintTitle(s string) {
	fmt.Println(styleTitle.Render(s))
}

// PrintSubtitle 打印副标题/说明文字。
func PrintSubtitle(s string) {
	fmt.Println(styleSubtitle.Render(s))
}

// PrintSuccess 打印成功信息（带 ✓）。
func PrintSuccess(format string, a ...any) {
	fmt.Println(styleSuccess.Render("✓ " + fmt.Sprintf(format, a...)))
}

// PrintError 打印错误信息（带 ✗）。
func PrintError(format string, a ...any) {
	fmt.Fprintln(os.Stderr, styleError.Render("✗ "+fmt.Sprintf(format, a...)))
}

// ---------- 任务相关渲染 ----------

// PriorityBadge 渲染优先级徽章。
func PriorityBadge(p string) string {
	switch p {
	case model.PriorityHigh:
		return lipgloss.NewStyle().Foreground(colorRed).Render("🔴 高")
	case model.PriorityMedium:
		return lipgloss.NewStyle().Foreground(colorYellow).Render("🟡 中")
	case model.PriorityLow:
		return lipgloss.NewStyle().Foreground(colorGreen).Render("🟢 低")
	default:
		return p
	}
}

// StatusIcon 渲染任务状态图标。
func StatusIcon(status string) string {
	if status == model.TaskStatusDone {
		return "☑ "
	}
	return "□ "
}

// DueDateLabel 渲染截止日期，过期则高亮为红色。
func DueDateLabel(due *time.Time, now time.Time) string {
	if due == nil {
		return lipgloss.NewStyle().Foreground(colorMuted).Render("—")
	}
	label := util.FormatDate(*due)
	if util.IsOverdue(*due, now) {
		return lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render("⚠️ " + label)
	}
	if util.IsToday(*due, now) {
		return lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Render("📅 今天")
	}
	return lipgloss.NewStyle().Foreground(colorBlue).Render("📅 " + label)
}

// WeekGoalRef 渲染关联的周目标引用。
func WeekGoalRef(id *int64) string {
	if id == nil {
		return ""
	}
	return lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("← week#%d", *id))
}

// PrintTaskList 渲染任务列表。
func PrintTaskList(tasks []model.Task, now time.Time) {
	if len(tasks) == 0 {
		PrintSubtitle("（无任务）")
		return
	}

	// 表头
	fmt.Printf("\n  %-3s %-30s %-16s %-10s %s\n",
		"ID", "任务", "截止", "优先级", "关联")
	fmt.Println("  " + strings.Repeat("─", 70))

	for _, t := range tasks {
		// 标题：已完成的加删除线
		title := t.Title
		if t.Status == model.TaskStatusDone {
			title = styleDone.Render(t.Title)
		}
		due := DueDateLabel(t.DueDate, now)
		pri := PriorityBadge(t.Priority)
		ref := WeekGoalRef(t.WeekGoalID)

		fmt.Printf("  %s%-2d %-30s %-16s %-10s %s\n",
			StatusIcon(t.Status), t.ID,
			truncate(title, 30), due, pri, ref)
	}
	fmt.Println()
}

// truncate 截断字符串到指定显示宽度（不区分样式）。
func truncate(s string, max int) string {
	if lipgloss.Width(s) <= max {
		return s
	}
	// 简单按 rune 截断
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
