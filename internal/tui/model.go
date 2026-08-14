package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"goal-tracker/internal/model"
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
	inputNone   inputMode = iota
	inputAdd    // 添加（类型由 pendingKind 决定：任务/周/季/年目标）
	inputDelete // 删除确认（类型由 pendingKind 决定）
)

// itemKind 表示 TUI 中操作的条目类型（任务或各级目标）。
type itemKind int

const (
	kindTask itemKind = iota
	kindWeekGoal
	kindQuarterGoal
	kindYearGoal
)

// kindLabel 返回类型的中文标签（用于提示与确认面板）。
func kindLabel(k itemKind) string {
	switch k {
	case kindTask:
		return "任务"
	case kindWeekGoal:
		return "周目标"
	case kindQuarterGoal:
		return "季度目标"
	case kindYearGoal:
		return "年度目标"
	default:
		return "条目"
	}
}

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
	mode      inputMode
	textInput textinput.Model
	pendingID    int64     // 待操作的条目 ID（删除确认用）
	pendingKind  itemKind  // 待操作/待创建的条目类型
	pendingTitle string    // 待删除条目的标题（确认面板展示，防误删）

	// 详情面板：Enter 打开，Esc 关闭
	showDetail bool

	// 周/季度导航偏移：0 = 当前，-1 = 上一周/季度，+1 = 下一周/季度
	// 用 ←/→ 键切换，避免"当前期之外的目标隐身"
	weekOffset    int
	quarterOffset int

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
		// 如果详情面板打开，任意导航键关闭面板（q 不退出程序，只关面板）
		if m.showDetail {
			m.showDetail = false
			return m, nil
		}

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
		case "left", "h":
			// 周/季视图：切换到上一周/上一季度，光标归零
			if m.activeTab == tabWeek {
				m.weekOffset--
				m.week.cursor = 0
				m.message = ""
			} else if m.activeTab == tabQuarter {
				m.quarterOffset--
				m.quarter.cursor = 0
				m.message = ""
			}
		case "right", "l":
			// 周/季视图：切换到下一周/下一季度，光标归零
			if m.activeTab == tabWeek {
				m.weekOffset++
				m.week.cursor = 0
				m.message = ""
			} else if m.activeTab == tabQuarter {
				m.quarterOffset++
				m.quarter.cursor = 0
				m.message = ""
			}
		case " ":
			return m.handleToggle()
		case "enter":
			// Enter 打开当前选中项的详情面板（查看完整标题）
			m.message = ""
			m.showDetail = true
			return m, nil
		case "a":
			return m.startAdd()
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

	// 详情面板（如果打开）：替换主内容区，聚焦查看完整信息
	if m.showDetail {
		content = m.renderDetailPanel()
	} else if m.mode != inputNone {
		// 输入面板（如果激活）
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
	switch {
	case m.showDetail:
		hint = "[Esc/Enter/任意键] 关闭详情"
	case m.mode == inputNone:
		// 周/季视图额外显示切换提示
		nav := ""
		switch m.activeTab {
		case tabWeek:
			nav = "  [←/→]切换周"
		case tabQuarter:
			nav = "  [←/→]切换季度"
		}
		hint = "[j/k]移动  [Space]完成  [Enter]详情  [a]添加  [x]删除" + nav + "  [Tab]切换  [?]帮助  [q]退出"
	default:
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

// kindForTab 返回当前视图对应的条目类型。
func kindForTab(t tabID) itemKind {
	switch t {
	case tabWeek:
		return kindWeekGoal
	case tabQuarter:
		return kindQuarterGoal
	case tabYear:
		return kindYearGoal
	default:
		return kindTask
	}
}

// stateForKind 返回类型对应的列表状态（用于刷新与光标调整）。
func (m *Model) stateForKind(k itemKind) *listState {
	switch k {
	case kindTask:
		return m.today
	case kindWeekGoal:
		return m.week
	case kindQuarterGoal:
		return m.quarter
	default:
		return m.year
	}
}

// startAdd 进入添加模式：按当前视图创建对应类型。
// 任务无周期；目标默认落在"当前显示期"（受 ←/→ 导航影响）。
func (m Model) startAdd() (tea.Model, tea.Cmd) {
	m.mode = inputAdd
	m.pendingKind = kindForTab(m.activeTab)
	m.textInput.Reset()
	m.textInput.Placeholder = fmt.Sprintf("输入%s标题，回车确认", kindLabel(m.pendingKind))
	m.textInput.Focus()
	return m, textinput.Blink
}

// startDelete 进入删除确认模式：删除当前视图选中的条目。
func (m Model) startDelete() (tea.Model, tea.Cmd) {
	kind, id, title, ok := m.selectedItem()
	if !ok {
		m.message = "没有选中的项"
		return m, nil
	}
	m.mode = inputDelete
	m.pendingKind = kind
	m.pendingID = id
	m.pendingTitle = title
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
	case inputAdd:
		title := strings.TrimSpace(m.textInput.Value())
		if title == "" {
			m.message = "标题不能为空"
			m.mode = inputNone
			m.textInput.Blur()
			return m, nil
		}
		if err := m.createItem(m.pendingKind, title); err != nil {
			m.message = "添加失败：" + err.Error()
			m.mode = inputNone
			m.textInput.Blur()
			return m, nil
		}
		m.stateForKind(m.pendingKind).refresh = true
		m.message = "✓ 已添加" + kindLabel(m.pendingKind) + "：" + title
		m.mode = inputNone
		m.textInput.Blur()
		return m, nil

	case inputDelete:
		ans := strings.ToLower(strings.TrimSpace(m.textInput.Value()))
		if ans != "y" && ans != "yes" {
			m.message = "已取消删除"
			m.mode = inputNone
			m.textInput.Blur()
			return m, nil
		}
		ok, err := m.deleteItem(m.pendingKind, m.pendingID)
		if err != nil {
			m.message = "删除失败：" + err.Error()
		} else if ok {
			m.message = "✓ 已删除" + kindLabel(m.pendingKind)
			state := m.stateForKind(m.pendingKind)
			state.refresh = true
			if state.cursor > 0 {
				state.cursor--
			}
		} else {
			m.message = "删除失败：" + kindLabel(m.pendingKind) + "不存在"
		}
		m.mode = inputNone
		m.textInput.Blur()
	}
	return m, nil
}

// createItem 按类型创建条目。目标落在当前显示期（受导航偏移影响）。
func (m Model) createItem(kind itemKind, title string) error {
	now := utilNow()
	switch kind {
	case kindTask:
		_, err := m.store.CreateTask(store.CreateTaskInput{Title: title})
		return err
	case kindWeekGoal:
		y, w := m.displayedWeek()
		_, err := m.store.CreateWeekGoal(store.CreateWeekGoalInput{Title: title, Year: y, Week: w})
		return err
	case kindQuarterGoal:
		y, q := m.displayedQuarter()
		_, err := m.store.CreateQuarterGoal(store.CreateQuarterGoalInput{Title: title, Year: y, Quarter: q})
		return err
	default: // kindYearGoal
		_, err := m.store.CreateYearGoal(store.CreateYearGoalInput{Title: title, Year: now.Year()})
		return err
	}
}

// deleteItem 按类型删除条目。
func (m Model) deleteItem(kind itemKind, id int64) (bool, error) {
	switch kind {
	case kindTask:
		return m.store.DeleteTask(id)
	case kindWeekGoal:
		return m.store.DeleteWeekGoal(id)
	case kindQuarterGoal:
		return m.store.DeleteQuarterGoal(id)
	default:
		return m.store.DeleteYearGoal(id)
	}
}

// handleToggle 处理 Space 键：切换当前视图选中项的完成状态。
// 今日视图切换任务，周/季/年视图切换对应目标。
func (m Model) handleToggle() (tea.Model, tea.Cmd) {
	switch m.activeTab {
	case tabToday:
		return m.toggleTask()
	case tabWeek:
		return m.toggleWeekGoal()
	case tabQuarter:
		return m.toggleQuarterGoal()
	case tabYear:
		return m.toggleYearGoal()
	}
	return m, nil
}

// toggleTask 切换任务完成状态（今日视图）。
func (m Model) toggleTask() (tea.Model, tea.Cmd) {
	task, ok := m.selectedTask()
	if !ok {
		return m, nil
	}
	newStatus := model.TaskStatusPending
	if task.Status == model.TaskStatusPending {
		newStatus = model.TaskStatusDone
	}
	_, err := m.store.SetTaskStatus(task.ID, newStatus)
	if err != nil {
		m.message = err.Error()
		return m, nil
	}
	m.today.refresh = true
	if newStatus == model.TaskStatusDone {
		m.message = "✓ 已完成"
	} else {
		m.message = "↩ 已恢复"
	}
	return m, nil
}

// toggleWeekGoal 切换周目标完成状态（active ↔ completed）。
func (m Model) toggleWeekGoal() (tea.Model, tea.Cmd) {
	wg, ok := m.selectedWeekGoal()
	if !ok {
		m.message = "没有选中的周目标"
		return m, nil
	}
	newStatus := model.WeekGoalStatusActive
	if wg.Status == model.WeekGoalStatusActive {
		newStatus = model.WeekGoalStatusCompleted
	}
	_, err := m.store.SetWeekGoalStatus(wg.ID, newStatus)
	if err != nil {
		m.message = err.Error()
		return m, nil
	}
	m.week.refresh = true
	if newStatus == model.WeekGoalStatusCompleted {
		m.message = "✓ 周目标已完成"
	} else {
		m.message = "↩ 周目标已恢复"
	}
	return m, nil
}

// toggleQuarterGoal 切换季度目标完成状态。
func (m Model) toggleQuarterGoal() (tea.Model, tea.Cmd) {
	qg, ok := m.selectedQuarterGoal()
	if !ok {
		m.message = "没有选中的季度目标"
		return m, nil
	}
	newStatus := model.QuarterGoalStatusActive
	if qg.Status == model.QuarterGoalStatusActive {
		newStatus = model.QuarterGoalStatusCompleted
	}
	_, err := m.store.SetQuarterGoalStatus(qg.ID, newStatus)
	if err != nil {
		m.message = err.Error()
		return m, nil
	}
	m.quarter.refresh = true
	if newStatus == model.QuarterGoalStatusCompleted {
		m.message = "✓ 季度目标已完成"
	} else {
		m.message = "↩ 季度目标已恢复"
	}
	return m, nil
}

// toggleYearGoal 切换年度目标完成状态。
func (m Model) toggleYearGoal() (tea.Model, tea.Cmd) {
	yg, ok := m.selectedYearGoal()
	if !ok {
		m.message = "没有选中的年度目标"
		return m, nil
	}
	newStatus := model.YearGoalStatusActive
	if yg.Status == model.YearGoalStatusActive {
		newStatus = model.YearGoalStatusCompleted
	}
	_, err := m.store.SetYearGoalStatus(yg.ID, newStatus)
	if err != nil {
		m.message = err.Error()
		return m, nil
	}
	m.year.refresh = true
	if newStatus == model.YearGoalStatusCompleted {
		m.message = "✓ 年度目标已完成"
	} else {
		m.message = "↩ 年度目标已恢复"
	}
	return m, nil
}

func (m Model) renderInputPanel() string {
	var label string
	switch m.mode {
	case inputAdd:
		label = fmt.Sprintf("➕ 新%s", kindLabel(m.pendingKind))
	case inputDelete:
		// 显示标题防止误删
		label = fmt.Sprintf("🗑️  确认删除%s #%d：%s (y/n)",
			kindLabel(m.pendingKind), m.pendingID, m.pendingTitle)
	}
	return styleInputBox.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			styleInputLabel.Render(label),
			m.textInput.View(),
		),
	)
}

// renderDetailPanel 渲染详情面板：完整显示选中项的所有字段。
// 核心价值：标题不截断，且显示上级目标的完整标题。
func (m Model) renderDetailPanel() string {
	var lines []string

	switch m.activeTab {
	case tabToday:
		if t, ok := m.selectedTask(); ok {
			due := "无"
			if t.DueDate != nil {
				due = util.FormatDate(*t.DueDate)
			}
			ref := "无"
			if t.WeekGoalID != nil {
				ref = fmt.Sprintf("周目标 #%d", *t.WeekGoalID)
				if wg, _ := m.store.GetWeekGoal(*t.WeekGoalID); wg != nil {
					ref += fmt.Sprintf("：%s", wg.Title)
				}
			}
			lines = []string{
				styleInputLabel.Render("📝 任务详情"),
				"",
				"标题：  " + t.Title,
				fmt.Sprintf("ID：    #%d", t.ID),
				"状态：  " + statusLabel(t.Status),
				"优先级：" + priorityLabel(t.Priority),
				"截止：  " + due,
				"关联：  " + ref,
				"创建：  " + t.CreatedAt.Format("2006-01-02 15:04"),
			}
		}

	case tabWeek:
		if wg, ok := m.selectedWeekGoal(); ok {
			pct := "—（未关联任务）"
			if p := wg.Progress(); p >= 0 {
				pct = fmt.Sprintf("%d%%（%d/%d）", p, wg.TaskDone, wg.TaskTotal)
			}
			upper := "无"
			if wg.QuarterGoalID != nil {
				upper = fmt.Sprintf("季度目标 #%d", *wg.QuarterGoalID)
				if qg, _ := m.store.GetQuarterGoal(*wg.QuarterGoalID); qg != nil {
					upper += fmt.Sprintf("：%s", qg.Title)
				}
			}
			lines = []string{
				styleInputLabel.Render("📅 周目标详情"),
				"",
				"标题：  " + wg.Title,
				fmt.Sprintf("ID：    #%d", wg.ID),
				"周期：  " + util.WeekLabel(wg.Year, wg.Week),
				"状态：  " + statusLabel(wg.Status),
				"进度：  " + pct,
				"上级：  " + upper,
				"创建：  " + wg.CreatedAt.Format("2006-01-02 15:04"),
			}
		}

	case tabQuarter:
		if qg, ok := m.selectedQuarterGoal(); ok {
			pct := "—（未关联周目标）"
			if p := qg.Progress(); p >= 0 {
				pct = fmt.Sprintf("%d%%（%d/%d 周）", p, qg.WeekDone, qg.WeekTotal)
			}
			upper := "无"
			if qg.YearGoalID != nil {
				upper = fmt.Sprintf("年度目标 #%d", *qg.YearGoalID)
				if yg, _ := m.store.GetYearGoal(*qg.YearGoalID); yg != nil {
					upper += fmt.Sprintf("：%s", yg.Title)
				}
			}
			lines = []string{
				styleInputLabel.Render("🏆 季度目标详情"),
				"",
				"标题：  " + qg.Title,
				fmt.Sprintf("ID：    #%d", qg.ID),
				"周期：  " + util.QuarterLabel(qg.Year, qg.Quarter),
				"状态：  " + statusLabel(qg.Status),
				"进度：  " + pct,
				"上级：  " + upper,
				"创建：  " + qg.CreatedAt.Format("2006-01-02 15:04"),
			}
			if qg.Description != "" {
				lines = append(lines, "描述：  "+qg.Description)
			}
		}

	case tabYear:
		if yg, ok := m.selectedYearGoal(); ok {
			pct := "—（未关联季度目标）"
			if p := yg.Progress(); p >= 0 {
				pct = fmt.Sprintf("%d%%（%d/%d 季度）", p, yg.QuarterDone, yg.QuarterTotal)
			}
			lines = []string{
				styleInputLabel.Render("🎯 年度目标详情"),
				"",
				"标题：  " + yg.Title,
				fmt.Sprintf("ID：    #%d", yg.ID),
				fmt.Sprintf("周期：  %d 年", yg.Year),
				"状态：  " + statusLabel(yg.Status),
				"进度：  " + pct,
				"创建：  " + yg.CreatedAt.Format("2006-01-02 15:04"),
			}
			if yg.Description != "" {
				lines = append(lines, "描述：  "+yg.Description)
			}
		}
	}

	if lines == nil {
		lines = []string{styleHint.Render("（没有选中项）")}
	}

	// 面板宽度跟随终端，超长标题自动换行（不会截断）
	panelWidth := m.width - 8
	if panelWidth < 40 {
		panelWidth = 40
	}
	return styleInputBox.Width(panelWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

// ---------- 工具函数 ----------

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
