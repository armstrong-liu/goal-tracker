package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBarWidth 进度条总宽度（字符数）。
const ProgressBarWidth = 10

// ProgressBar 渲染一个进度条，如 [█████░░░░░] 50%
// progress 为 -1（无关联子项）时返回 "—"。
func ProgressBar(progress int) string {
	if progress < 0 {
		return lipgloss.NewStyle().Foreground(colorMuted).Render("—")
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	filled := progress * ProgressBarWidth / 100
	empty := ProgressBarWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	var colored string
	switch {
	case progress >= 100:
		colored = lipgloss.NewStyle().Foreground(colorGreen).Render(bar)
	case progress >= 50:
		colored = lipgloss.NewStyle().Foreground(colorYellow).Render(bar)
	default:
		colored = lipgloss.NewStyle().Foreground(colorRed).Render(bar)
	}

	return fmt.Sprintf("[%s] %3d%%", colored, progress)
}

// GoalStatusIcon 渲染目标状态图标。
func GoalStatusIcon(status string) string {
	switch status {
	case "completed":
		return lipgloss.NewStyle().Foreground(colorGreen).Render("✅")
	case "archived":
		return lipgloss.NewStyle().Foreground(colorMuted).Render("📦")
	default: // active
		return "🎯"
	}
}

// PrintInfo 打印键值对信息。
func PrintInfo(key, value string) {
	keyStyle := lipgloss.NewStyle().Foreground(colorBlue).Width(10)
	fmt.Printf("  %s %s\n", keyStyle.Render(key), value)
}
