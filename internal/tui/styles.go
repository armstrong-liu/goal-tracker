// Package tui 提供 Goal Tracker 的终端交互界面（基于 Bubbletea）。
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ---------- 色彩主题 ----------

var (
	colorPrimary   = lipgloss.Color("63")  // 紫色 - 标题、激活
	colorAccent    = lipgloss.Color("213") // 粉色 - 强调
	colorSuccess   = lipgloss.Color("36")  // 青绿 - 成功/完成
	colorWarn      = lipgloss.Color("203") // 红 - 警告/过期
	colorInfo      = lipgloss.Color("117") // 蓝 - 信息
	colorMuted     = lipgloss.Color("245") // 灰 - 次要文字
	colorHighlight = lipgloss.Color("51")  // 亮青 - 选中项背景
	colorBorder    = lipgloss.Color("240") // 深灰 - 边框

	colorPriorityHigh   = lipgloss.Color("203")
	colorPriorityMedium = lipgloss.Color("221")
	colorPriorityLow    = lipgloss.Color("156")
)

// ---------- 样式 ----------

var (
	// 顶部标题栏：单行，深色背景
	styleTitleBar = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary).
			Padding(0, 1)

	// Tab 栏
	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary).
			Padding(0, 2)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(colorMuted).
				Background(lipgloss.Color("#2A2A2A")).
				Padding(0, 2)

	// 选中行：带左边框 + 高亮背景
	styleItemSelected = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#000000")).
				Background(colorHighlight)

	styleItemNormal = lipgloss.NewStyle()

	styleItemDone = lipgloss.NewStyle().
			Foreground(colorMuted).
			Strikethrough(true)

	// 内容区：带边框包围
	styleContentBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	// 状态栏：底部，深色背景
	styleStatusBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3A3A3A")).
			Padding(0, 1)

	// 输入框：强调边框
	styleInputBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 1)

	styleInputLabel = lipgloss.NewStyle().
			Foreground(colorInfo).
			Bold(true)

	// 提示文字
	styleHint = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	styleError = lipgloss.NewStyle().
			Foreground(colorWarn).
			Bold(true)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)
)

// ---------- 列对齐工具（核心修复） ----------

// fitWidth 把字符串渲染成固定显示宽度：
// - 超长 → 截断加 "…"
// - 不足 → 右侧补空格
// 使用 lipgloss.Width 计算真实宽度（正确处理中文/emoji/ANSI 转义）。
func fitWidth(s string, width int) string {
	actual := lipgloss.Width(s)
	if actual == width {
		return s
	}
	if actual > width {
		// 截断
		return truncateToWidth(s, width)
	}
	// 不足：补空格
	return s + strings.Repeat(" ", width-actual)
}

// truncateToWidth 按显示宽度截断字符串，末尾加 "…"。
// 若宽度 < 2，直接返回 width 个空格（避免 "…" 把宽度撑爆）。
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth < 2 {
		return strings.Repeat(" ", maxWidth)
	}
	target := maxWidth - 1 // 留 1 格给 "…"
	var b strings.Builder
	currentWidth := 0
	for _, r := range s {
		rw := runeDisplayWidth(r)
		if currentWidth+rw > target {
			break
		}
		b.WriteRune(r)
		currentWidth += rw
	}
	b.WriteRune('…')
	// 确保最终宽度精确（… 占 1 格）
	result := b.String()
	if pad := maxWidth - lipgloss.Width(result); pad > 0 {
		result += strings.Repeat(" ", pad)
	}
	return result
}

// runeDisplayWidth 返回单个 rune 的显示宽度。
// 简单规则：ASCII 可打印 = 1，其他 = 2（覆盖中日韩和 emoji 的大多数情况）。
// lipgloss.Width 内部用的也是类似规则，这里保持一致。
func runeDisplayWidth(r rune) int {
	if r < 0x80 {
		return 1 // ASCII
	}
	if r < 0x300 {
		return 1 // 拉丁扩展等
	}
	return 2 // CJK / emoji 等
}

// ---------- 渲染辅助函数 ----------

// priorityColor 返回优先级对应的颜色。
func priorityColor(priority string) lipgloss.Color {
	switch priority {
	case "high":
		return colorPriorityHigh
	case "medium":
		return colorPriorityMedium
	case "low":
		return colorPriorityLow
	default:
		return colorMuted
	}
}

// priorityBadge 渲染优先级（固定显示宽度，便于对齐）。
// 用纯文字 "高"/"中"/"低" + 颜色，避免 emoji 宽度不稳定。
func priorityBadge(priority string) string {
	label := priorityLabel(priority)
	return lipgloss.NewStyle().Foreground(priorityColor(priority)).Render(label)
}

func priorityLabel(p string) string {
	switch p {
	case "high":
		return "高"
	case "medium":
		return "中"
	case "low":
		return "低"
	default:
		return p
	}
}

// statusIcon 渲染任务状态图标。
// 使用稳定的方框字符，避免 Windows Terminal 宽度问题。
func statusIcon(status string) string {
	if status == "done" {
		return "[x]"
	}
	return "[ ]"
}

// progressBar 渲染进度条，width 为字符宽度。
func progressBar(progress, width int) string {
	if progress < 0 {
		return lipgloss.NewStyle().Foreground(colorMuted).Render("—")
	}
	if progress > 100 {
		progress = 100
	}
	if progress < 0 {
		progress = 0
	}

	filled := progress * width / 100
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	var colored string
	switch {
	case progress >= 100:
		colored = lipgloss.NewStyle().Foreground(colorSuccess).Render(bar)
	case progress >= 50:
		colored = lipgloss.NewStyle().Foreground(colorPriorityMedium).Render(bar)
	default:
		colored = lipgloss.NewStyle().Foreground(colorWarn).Render(bar)
	}

	pct := itoa(progress) + "%"
	pctStyled := lipgloss.NewStyle().Bold(true).Render(fitWidth(pct, 4))

	return "[" + colored + "] " + pctStyled
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
