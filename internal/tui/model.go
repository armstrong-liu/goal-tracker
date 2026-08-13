package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"goal-tracker/internal/store"
	"goal-tracker/internal/util"
)

// ---------- Tab 定义 ----------

type tabID int

const (
	tabToday tabID = iota
	tabWeek
	tabQuarter
	tabYear
)

var tabNames = []string{"今日任务", "本周目标", "季度目标", "年度目标"}

// ---------- 输入模式 ----------

type inputMode int

const (
	inputNone inputMode = iota
	inputAddTask    // 添加任务
	inputDeleteTask // 删除确认
)

// ---------- 主 Model ----------

// Model 是 TUI 的顶层 Model，实现 tea.Model 接口。
type Model struct {
	store *store.Store

	activeTab tabID
	width     int
	height    int

	// 各视图的本地状态
	today   *listState
	week    *listState
	quarter *listState
	year    *listState

	// 输入模式
	mode     inputMode
	textInput textinput.Model
	pendingID int64 // 待操作的任务/目标 ID

	// 消息（短暂提示，操作后显示）
	message string
}

// listState 是各视图通用的列表状态（光标 + 数据）。
type listState struct {
	cursor  int
	refresh bool // 标记需要重新加载数据
}

// NewModel 创建主 Model。
func NewModel(s *store.Store) Model {
	ti := textinput.New()
	ti.Placeholder = "输入内容..."
	ti.CharLimit = 200

	return Model{
		store:     s,
		activeTab: tabToday,
		today:     &listState{},
		week:      &listState{},
		quarter:   &listState{},
		year:      &listState{},
		textInput: ti,
	}
}

// ---------- tea.Model 接口实现 ----------

// Init 启动时执行的命令。
func (m Model) Init() tea.Cmd {
	// 启动时刷新所有视图的数据
	return nil
}

// Update 处理消息（按键、窗口大小等）。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// 如果处于输入模式，按键交给输入框处理
		if m.mode != inputNone {
			return m.handleInputMode(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "1":
			m.activeTab = tabToday
			m.today.refresh = true
		case "2":
			m.activeTab = tabWeek
			m.week.refresh = true
		case "3":
			m.activeTab = tabQuarter
			m.quarter.refresh = true
		case "4":
			m.activeTab = tabYear
			m.year.refresh = true

		case "tab":
			m.activeTab = (m.activeTab + 1) % 4
			m.markRefresh()
		case "shift+tab":
			m.activeTab = (m.activeTab + 3) % 4
			m.markRefresh()

		case "?":
			// TODO: 帮助浮层（v1.0 简化：直接在状态栏显示快捷键）
			m.message = "快捷键: 1234切换 | Tab下一个 | j/k或↑↓移动 | Space完成 | a添加 | x删除 | q退出"

		// 以下按键交给当前视图处理
		case "up", "k":
			m.curState().cursor = max(0, m.curState().cursor-1)
			m.message = ""
		case "down", "j":
			// 光标下移上限由视图数据条数决定（view 渲染时处理）
			m.curState().cursor++
			m.message = ""
		case " ", "enter":
			return m.handleAction()
		case "a":
			return m.startAddTask()
		case "x":
			return m.startDelete()
		}
	}

	return m, tea.Batch(cmds...)
}

// View 渲染界面。
func (m Model) View() string {
	if m.width == 0 {
		return "初始化中..."
	}

	// 顶部：标题栏 + Tab 栏（合成一个 header）
	header := lipgloss.JoinVertical(lipgloss.Left,
		m.renderTitleBar(),
		m.renderTabs(),
	)

	// 中部：当前视图（带边框）
	var content string
	switch m.activeTab {
	case tabToday:
		content = m.renderTodayView()
	case tabWeek:
		content = m.renderWeekView()
	case tabQuarter:
		content = m.renderQuarterView()
	case tabYear:
		content = m.renderYearView()
	}

	// 输入面板（如果激活）
	if m.mode != inputNone {
		content = lipgloss.JoinVertical(lipgloss.Left,
			content,
			"",
			m.renderInputPanel(),
		)
	}

	// 底部：状态栏
	status := m.renderStatusBar()

	// 所有区块对齐到终端宽度
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Width(m.width).Render(header),
		content,
		lipgloss.NewStyle().Width(m.width).Render(status),
	)
}

// ---------- 内部方法 ----------

func (m *Model) markRefresh() {
	switch m.activeTab {
	case tabToday:
		m.today.refresh = true
	case tabWeek:
		m.week.refresh = true
	case tabQuarter:
		m.quarter.refresh = true
	case tabYear:
		m.year.refresh = true
	}
}

func (m Model) curState() *listState {
	switch m.activeTab {
	case tabToday:
		return m.today
	case tabWeek:
		return m.week
	case tabQuarter:
		return m.quarter
	default:
		return m.year
	}
}

func (m Model) renderTitleBar() string {
	title := "🎯 Goal Tracker"
	now := utilNow()
	y, w := util.ISOWeek(now)
	q := util.CurrentQuarter(now)
	subtitle := fmt.Sprintf("%s · %s · %s",
		now.Format("2006-01-02"),
		util.WeekLabel(y, w),
		util.QuarterLabel(now.Year(), q),
	)
	// 标题（白字紫底）+ 副标题（灰字）拼接，整体用 Width 拉满形成完整背景色条
	left := styleTitleBar.Render(title)
	right := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC")).Render(subtitle)
	bar := lipgloss.JoinHorizontal(lipgloss.Center, left, "    ", right)
	return lipgloss.NewStyle().
		Background(colorPrimary).
		Width(m.width).
		Render(" " + bar)
}

func (m Model) renderTabs() string {
	var tabs []string
	for i, name := range tabNames {
		label := fmt.Sprintf("%d %s", i+1, name)
		if tabID(i) == m.activeTab {
			tabs = append(tabs, styleTabActive.Render(label))
		} else {
			tabs = append(tabs, styleTabInactive.Render(label))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	// 不拉满宽度，保持自然宽度，左侧缩进 0
	return row
}

func (m Model) renderStatusBar() string {
	var hint string
	if m.mode == inputNone {
		hint = "[j/k]移动  [Space]完成  [a]添加  [x]删除  [Tab]切换  [?]帮助  [q]退出"
	} else {
		hint = "[Enter]确认  [Esc]取消"
	}
	left := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render(hint)

	var msg string
	if m.message != "" {
		msg = "    " + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Bold(true).
			Render(m.message)
	}
	// 用 background 拉满到终端宽度，形成完整状态栏
	return styleStatusBar.Width(m.width).Render(left + msg)
}

// handleInputMode 处理输入模式下的按键。
func (m Model) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = inputNone
		m.textInput.Blur()
		m.message = "已取消"
		return m, nil
	case "enter":
		return m.confirmInput()
	}

	// 其他按键转给 textinput
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// startAddTask 进入添加任务模式。
func (m Model) startAddTask() (tea.Model, tea.Cmd) {
	// 只在"今日任务"视图支持添加（v1.0 简化）
	if m.activeTab != tabToday {
		m.message = "添加任务请在「今日任务」视图操作"
		return m, nil
	}
	m.mode = inputAddTask
	m.textInput.Reset()
	m.textInput.Placeholder = "输入任务标题，回车确认"
	m.textInput.Focus()
	return m, textinput.Blink
}

// startDelete 进入删除确认模式。
func (m Model) startDelete() (tea.Model, tea.Cmd) {
	id, ok := m.selectedTaskID()
	if !ok {
		m.message = "没有选中的项"
		return m, nil
	}
	m.mode = inputDeleteTask
	m.pendingID = id
	m.textInput.Reset()
	m.textInput.Placeholder = "输入 y 确认删除，n 取消"
	m.textInput.Focus()
	return m, textinput.Blink
}

// confirmInput 处理输入确认。
func (m Model) confirmInput() (tea.Model, tea.Cmd) {
	// 注意：值接收者，defer 修改的是副本，对返回值无效。
	// 所以在每个 return 前显式重置 mode。
	switch m.mode {
	case inputAddTask:
		title := strings.TrimSpace(m.textInput.Value())
		if title == "" {
			m.message = "标题不能为空"
			m.mode = inputNone
			m.textInput.Blur()
			return m, nil
		}
		_, err := m.store.CreateTask(store.CreateTaskInput{Title: title})
		if err != nil {
			m.message = "添加失败：" + err.Error()
			m.mode = inputNone
			m.textInput.Blur()
			return m, nil
		}
		m.today.refresh = true
		m.message = "✓ 已添加任务：" + title
		m.mode = inputNone
		m.textInput.Blur()
		return m, nil

	case inputDeleteTask:
		ans := strings.ToLower(strings.TrimSpace(m.textInput.Value()))
		if ans != "y" && ans != "yes" {
			m.message = "已取消删除"
			m.mode = inputNone
			m.textInput.Blur()
			return m, nil
		}
		ok, _ := m.store.DeleteTask(m.pendingID)
		if ok {
			m.message = "✓ 已删除任务"
			m.today.refresh = true
			if m.today.cursor > 0 {
				m.today.cursor--
			}
		} else {
			m.message = "删除失败：任务不存在"
		}
		m.mode = inputNone
		m.textInput.Blur()
	}
	return m, nil
}

// handleAction 处理 Space/Enter 键：在今日视图切换任务状态。
func (m Model) handleAction() (tea.Model, tea.Cmd) {
	if m.activeTab != tabToday {
		// 其他视图暂不处理 Space（v1.0 简化）
		return m, nil
	}
	task, ok := m.selectedTask()
	if !ok {
		return m, nil
	}
	newStatus := "pending"
	if task.Status == "pending" {
		newStatus = "done"
	}
	_, err := m.store.SetTaskStatus(task.ID, newStatus)
	if err != nil {
		m.message = err.Error()
		return m, nil
	}
	m.today.refresh = true
	if newStatus == "done" {
		m.message = "✓ 已完成"
	} else {
		m.message = "↩ 已恢复"
	}
	return m, nil
}

func (m Model) renderInputPanel() string {
	var label string
	switch m.mode {
	case inputAddTask:
		label = "➕ 新任务"
	case inputDeleteTask:
		label = fmt.Sprintf("🗑️  确认删除任务 #%d？(y/n)", m.pendingID)
	}
	return styleInputBox.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			styleInputLabel.Render(label),
			m.textInput.View(),
		),
	)
}

// ---------- 工具函数 ----------

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
