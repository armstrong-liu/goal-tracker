package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestRenderCell_Width 验证单元格渲染后宽度正确（含中文/emoji/ANSI 转义）。
func TestRenderCell_Width(t *testing.T) {
	cases := []struct {
		name  string
		value string
		width int
	}{
		{"ascii", "hello", 8},
		{"中文", "任务标题", 10},
		{"emoji", "🎯目标", 8},
		{"空字符串", "", 6},
		{"超长", "这是一个非常非常长的标题需要截断", 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := renderCell(c.value, c.width, lipgloss.NewStyle())
			// 渲染后剥离 ANSI 转义，检查纯文本显示宽度
			plain := lipgloss.NewStyle().Render(out) // 不可靠，用 Strip
			_ = plain
			// 用 lipgloss.Width 测量（它会自动忽略 ANSI 转义）
			actual := lipgloss.Width(out)
			if actual != c.width {
				t.Errorf("renderCell(%q, %d) 宽度 = %d，应为 %d\n输出: |%s|",
					c.value, c.width, actual, c.width, out)
			}
		})
	}
}

// TestRenderRow_ColumnAlignment 验证同一列在不同行的位置一致。
// 这是"对齐"的本质：每行的同一列起始位置相同。
func TestRenderRow_ColumnAlignment(t *testing.T) {
	cols := []tableColumn{
		{Title: "A", Width: 6},
		{Title: "B", Width: 8},
		{Title: "C", Width: 5},
	}
	rows := [][]string{
		{"短", "中文字", "x"},
		{"长内容", "这是一个比较长的内容", "yyyy"},
		{"", "空", ""},
	}

	// 渲染所有行，记录每行中第 2 列（B 列）的起始位置
	positions := []int{}
	for _, r := range rows {
		out := renderRow(r, cols, lipgloss.NewStyle(), "│")
		// 找第二个单元格的起始位置（第一个 │ 之后）
		// 用 lipgloss.Width 测量到第一个 │ 的宽度
		idx := strings.Index(out, "│")
		if idx < 0 {
			t.Fatalf("找不到分隔符: %s", out)
		}
		pos := lipgloss.Width(out[:idx])
		// pos 是第一列的宽度。第一个 │ 后是第二列开始。
		// 实际我们要的是"第二列内容"的起始显示位置
		// = 第一列宽度 + 分隔符宽度(1)
		col2Start := pos + 1
		positions = append(positions, col2Start)
	}

	t.Logf("各行第二列起始位置: %v", positions)
	for i := 1; i < len(positions); i++ {
		if positions[i] != positions[0] {
			t.Errorf("第 %d 行第二列起始位置 = %d，与第 0 行 %d 不一致（未对齐）",
				i, positions[i], positions[0])
		}
	}
}

// TestRenderTable_Complete 验证完整表格渲染（表头+分隔线+数据行）。
func TestRenderTable_Complete(t *testing.T) {
	cols := []tableColumn{
		{Title: "ID", Width: 5},
		{Title: "任务", Width: 10},
	}
	rows := [][]string{
		{"#1", "写文档"},
		{"#2", "开会"},
	}
	out := renderTable(cols, rows, 0, defaultTableStyle)

	lines := strings.Split(out, "\n")
	if len(lines) != 4 { // 表头 + 分隔线 + 2 数据行
		t.Errorf("行数 = %d，应为 4", len(lines))
	}

	// 检查表头包含 "ID" 和 "任务"
	if !strings.Contains(lines[0], "ID") {
		t.Error("表头应包含 ID")
	}

	// 检查分隔线包含 ┼
	if !strings.Contains(lines[1], "┼") {
		t.Error("分隔线应包含 ┼")
	}

	// 检查数据行包含内容
	if !strings.Contains(lines[2], "写文档") {
		t.Error("第一行数据应包含 '写文档'")
	}
}

// TestRenderTable_SelectedHighlight 验证选中行包含正确内容。
// 注意：非 TTY 环境（测试）下 lipgloss 会禁用颜色，所以只验证文本内容，
// 不验证 ANSI 转义（颜色在真实 TUI 中才会生效）。
func TestRenderTable_SelectedHighlight(t *testing.T) {
	cols := []tableColumn{{Title: "X", Width: 10}}
	rows := [][]string{
		{"第一行"},
		{"第二行"},
	}
	// 选中第二行（index=1）
	out := renderTable(cols, rows, 1, defaultTableStyle)
	lines := strings.Split(out, "\n")

	// 所有行都应包含对应的文本内容
	if !strings.Contains(lines[2], "第一行") {
		t.Error("第一行应包含 '第一行'")
	}
	if !strings.Contains(lines[3], "第二行") {
		t.Error("第二行应包含 '第二行'")
	}
}
