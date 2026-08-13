package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ---------- 通用表格渲染 ----------
//
// 这套函数借鉴 bubbles/table 的渲染原理，但只做"渲染"不做"交互"。
// 核心思路：
//   1. 每个单元格用 lipgloss.NewStyle().Width(w).MaxWidth(w).Inline(true) 渲染
//      —— lipgloss 会自动按"显示宽度"补空格或截断，正确处理中文/emoji
//   2. 用 lipgloss.JoinHorizontal 拼接同一行的多个单元格
//   3. 表头、分隔线、数据行用同一套列宽定义 → 自动对齐
//
// 改变列内容、列宽、增删列，都不需要调整任何"对齐代码"。

// tableColumn 定义表格的一列。
type tableColumn struct {
	Title string
	Width int
}

// tableStyle 表格各部分的样式。
type tableStyle struct {
	Header    lipgloss.Style // 表头单元格
	Cell      lipgloss.Style // 普通数据单元格
	Selected  lipgloss.Style // 选中行单元格
	Separator lipgloss.Color // 列分隔符颜色
}

// defaultTableStyle 默认表格样式。
var defaultTableStyle = tableStyle{
	Header: lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#3A3A5A")),
	Cell:     lipgloss.NewStyle(),
	Selected: lipgloss.NewStyle().Bold(true).Background(colorHighlight).Foreground(lipgloss.Color("#000000")),
	Separator: colorBorder,
}

// renderCell 把单个值渲染成固定宽度的单元格。
// 用 lipgloss 的 Width 样式自动处理对齐：
//   - 不足宽度：右侧自动补空格
//   - 超过宽度：交给上层 truncate（这里不截断，保持简单）
// cellStyle 的颜色/背景会正确生效（不用 Inline，避免背景色被剥离）。
func renderCell(value string, width int, cellStyle lipgloss.Style) string {
	return cellStyle.Width(width).Render(value)
}

// renderRow 把一行数据渲染成字符串。
// 用 lipgloss.JoinHorizontal 拼接各单元格 + 分隔符，自动按宽度对齐。
func renderRow(cells []string, columns []tableColumn, cellStyle lipgloss.Style, separator string) string {
	parts := make([]string, 0, len(columns)*2)
	for i, col := range columns {
		if i >= len(cells) {
			break
		}
		if i > 0 {
			parts = append(parts, separator)
		}
		parts = append(parts, renderCell(cells[i], col.Width, cellStyle))
	}
	// JoinHorizontal 会正确处理各部分的显示宽度对齐
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderHeader 渲染表头行（列标题）。
func renderHeader(columns []tableColumn, ts tableStyle) string {
	titles := make([]string, 0, len(columns))
	for _, c := range columns {
		titles = append(titles, c.Title)
	}
	return renderRow(titles, columns, ts.Header, sepVertical(ts))
}

// sepVertical 渲染列间竖分隔符（带颜色）。
func sepVertical(ts tableStyle) string {
	return lipgloss.NewStyle().Foreground(ts.Separator).Render("│")
}

// renderDivider 渲染表头和数据之间的分隔线。
// 列宽完全跟随 columns 定义，不需要手动计算。
func renderDivider(columns []tableColumn, ts tableStyle) string {
	sep := lipgloss.NewStyle().Foreground(ts.Separator)
	parts := make([]string, 0, len(columns)*2)
	for i, col := range columns {
		if i > 0 {
			parts = append(parts, sep.Render("┼"))
		}
		// 每列用 col.Width 个 ─（和 renderCell 的 Width 一致）
		parts = append(parts, sep.Render(strings.Repeat("─", col.Width)))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderTable 一步到位渲染完整表格：表头 + 分隔线 + 数据行（带选中高亮）。
// selectedIndex < 0 表示没有选中项。
func renderTable(columns []tableColumn, rows [][]string, selectedIndex int, ts tableStyle) string {
	var b strings.Builder
	// 表头
	b.WriteString(renderHeader(columns, ts))
	b.WriteString("\n")
	// 分隔线
	b.WriteString(renderDivider(columns, ts))
	b.WriteString("\n")
	// 数据行
	for i, row := range rows {
		style := ts.Cell
		if i == selectedIndex {
			style = ts.Selected
		}
		b.WriteString(renderRow(row, columns, style, "│"))
		if i < len(rows)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
