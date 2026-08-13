package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"goal-tracker/internal/model"
	"goal-tracker/internal/store"
	"goal-tracker/internal/util"
)

// exportCmd: gt export —— 导出 Markdown。
var (
	exportOutput string
	exportScope  string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "📄 导出 Markdown",
	Long: `把目标与任务导出为 Markdown 格式，便于归档、分享或生成周报。

支持 5 种导出范围：
  today    今日到期 + 过期任务
  week     本周目标及其任务（默认）
  quarter  本季度目标及其下的周目标
  year     本年度目标及其下的季度目标
  all      完整层级（年→季→周→任务）`,
	Example: `  gt export                           # 导出本周（输出到屏幕）
  gt export -o weekly.md              # 导出到文件
  gt export --scope all -o full.md    # 导出完整层级
  gt export --scope today             # 只导出今日任务`,
	RunE: runExport,
}

func runExport(cmd *cobra.Command, args []string) error {
	// 校验 scope
	validScopes := map[string]bool{"today": true, "week": true, "quarter": true, "year": true, "all": true}
	if !validScopes[exportScope] {
		return fmt.Errorf("无效的 scope: %s（支持 today/week/quarter/year/all）", exportScope)
	}

	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	content, err := buildExport(s, exportScope)
	if err != nil {
		return err
	}

	// 输出
	if exportOutput == "" {
		// 输出到 stdout
		fmt.Print(content)
		return nil
	}
	// 写文件
	if err := os.WriteFile(exportOutput, []byte(content), 0o644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	PrintSuccess("已导出到 %s（%d 字节）", exportOutput, len(content))
	return nil
}

// buildExport 根据范围构造 Markdown 内容。
func buildExport(s *store.Store, scope string) (string, error) {
	now := time.Now()
	y, w := util.ISOWeek(now)
	q := util.CurrentQuarter(now)
	year := now.Year()

	var b strings.Builder

	switch scope {
	case "today":
		writeExportHeader(&b, fmt.Sprintf("今日概览 - %s", now.Format("2006-01-02")))
		writeTodaySection(&b, s, now)

	case "week":
		writeExportHeader(&b, fmt.Sprintf("周回顾 - %s", util.WeekLabel(y, w)))
		writeWeekSection(&b, s, y, w)

	case "quarter":
		writeExportHeader(&b, fmt.Sprintf("季度目标 - %s", util.QuarterLabel(year, q)))
		writeQuarterSection(&b, s, year, q)

	case "year":
		writeExportHeader(&b, fmt.Sprintf("年度目标 - %d", year))
		writeYearSection(&b, s, year)

	case "all":
		writeExportHeader(&b, fmt.Sprintf("完整目标体系 - %d", year))
		writeAllSection(&b, s, year, now)
	}

	return b.String(), nil
}

// writeExportHeader 写入 Markdown 顶层标题 + 生成时间。
func writeExportHeader(b *strings.Builder, title string) {
	fmt.Fprintf(b, "# %s\n\n", title)
	fmt.Fprintf(b, "> 生成时间：%s\n\n", time.Now().Format("2006-01-02 15:04:05"))
}

// writeTodaySection 导出今日部分：过期 + 今日到期 + 统计。
func writeTodaySection(b *strings.Builder, s *store.Store, now time.Time) {
	// 过期任务
	overdue, _ := s.ListTasks(store.TaskFilter{Status: "pending", DueBefore: &now})
	if len(overdue) > 0 {
		fmt.Fprintf(b, "## ⚠️ 已过期（%d 项）\n\n", len(overdue))
		for _, t := range overdue {
			dueStr := ""
			if t.DueDate != nil {
				dueStr = util.FormatDate(*t.DueDate)
			}
			fmt.Fprintf(b, "- [ ] **%s** （截止 %s，%s）\n",
				t.Title, dueStr, priorityLabel(t.Priority))
		}
		fmt.Fprintln(b)
	}

	// 今日到期
	today, _ := s.ListTasks(store.TaskFilter{Status: "pending", DueDate: &now})
	fmt.Fprintf(b, "## 📅 今日到期（%d 项）\n\n", len(today))
	for _, t := range today {
		fmt.Fprintf(b, "- [ ] **%s** （%s）\n", t.Title, priorityLabel(t.Priority))
	}
	fmt.Fprintln(b)

	// 统计
	writeTaskStats(b, s)
}

// writeWeekSection 导出本周目标 + 任务。
func writeWeekSection(b *strings.Builder, s *store.Store, year, week int) {
	weekGoals, err := s.ListWeekGoalsWithProgress(store.WeekGoalFilter{Year: year, Week: week})
	if err != nil || len(weekGoals) == 0 {
		fmt.Fprintln(b, "（本周没有目标）")
		return
	}

	var completedGoals int
	var totalTasks, doneTasks int

	for i, wg := range weekGoals {
		num := i + 1
		statusMark := ""
		if wg.Status == model.WeekGoalStatusCompleted {
			statusMark = " ✓"
			completedGoals++
		}
		pct := wg.Progress()
		pctStr := "—"
		if pct >= 0 {
			pctStr = fmt.Sprintf("%d%%", pct)
		}
		fmt.Fprintf(b, "## %d. %s [%s]%s\n\n", num, wg.Title, pctStr, statusMark)

		// 任务列表
		tasks, _ := s.ListTasks(store.TaskFilter{WeekGoalID: &wg.ID})
		for _, t := range tasks {
			mark := "[ ]"
			if t.Status == model.TaskStatusDone {
				mark = "[x]"
				doneTasks++
			}
			fmt.Fprintf(b, "- %s %s\n", mark, t.Title)
			totalTasks++
		}
		fmt.Fprintln(b)
	}

	// 统计
	fmt.Fprintln(b, "## 📊 统计")
	fmt.Fprintf(b, "- 完成目标：%d/%d\n", completedGoals, len(weekGoals))
	if totalTasks > 0 {
		fmt.Fprintf(b, "- 完成任务：%d/%d（%d%%）\n", doneTasks, totalTasks, doneTasks*100/totalTasks)
	}
}

// writeQuarterSection 导出本季度目标。
func writeQuarterSection(b *strings.Builder, s *store.Store, year, quarter int) {
	quarterGoals, err := s.ListQuarterGoalsWithProgress(store.QuarterGoalFilter{Year: year, Quarter: quarter})
	if err != nil || len(quarterGoals) == 0 {
		fmt.Fprintln(b, "（本季度没有目标）")
		return
	}

	for i, qg := range quarterGoals {
		num := i + 1
		pct := qg.Progress()
		pctStr := "—"
		if pct >= 0 {
			pctStr = fmt.Sprintf("%d%%", pct)
		}
		fmt.Fprintf(b, "## %d. %s [%s]\n\n", num, qg.Title, pctStr)
		if qg.Description != "" {
			fmt.Fprintf(b, "> %s\n\n", qg.Description)
		}
		// 列出关联的周目标
		weekGoals, _ := s.ListWeekGoals(store.WeekGoalFilter{QuarterGoalID: &qg.ID})
		for _, wg := range weekGoals {
			mark := "[ ]"
			if wg.Status == model.WeekGoalStatusCompleted {
				mark = "[x]"
			}
			fmt.Fprintf(b, "- %s %s（%s）\n", mark, wg.Title, util.WeekLabel(wg.Year, wg.Week))
		}
		fmt.Fprintln(b)
	}
}

// writeYearSection 导出本年度目标。
func writeYearSection(b *strings.Builder, s *store.Store, year int) {
	yearGoals, err := s.ListYearGoalsWithProgress(store.YearGoalFilter{Year: year})
	if err != nil || len(yearGoals) == 0 {
		fmt.Fprintln(b, "（本年没有目标）")
		return
	}

	for i, yg := range yearGoals {
		num := i + 1
		pct := yg.Progress()
		pctStr := "—"
		if pct >= 0 {
			pctStr = fmt.Sprintf("%d%%", pct)
		}
		fmt.Fprintf(b, "## %d. %s [%s]\n\n", num, yg.Title, pctStr)
		if yg.Description != "" {
			fmt.Fprintf(b, "> %s\n\n", yg.Description)
		}
		// 列出关联的季度目标
		quarterGoals, _ := s.ListQuarterGoals(store.QuarterGoalFilter{YearGoalID: &yg.ID})
		for _, qg := range quarterGoals {
			mark := "[ ]"
			if qg.Status == model.QuarterGoalStatusCompleted {
				mark = "[x]"
			}
			fmt.Fprintf(b, "- %s %s（%s）\n", mark, qg.Title, util.QuarterLabel(qg.Year, qg.Quarter))
		}
		fmt.Fprintln(b)
	}
}

// writeAllSection 导出完整层级：年→季→周→任务。
func writeAllSection(b *strings.Builder, s *store.Store, year int, now time.Time) {
	yearGoals, err := s.ListYearGoals(store.YearGoalFilter{Year: year})
	if err != nil || len(yearGoals) == 0 {
		fmt.Fprintf(b, "（%d 年没有年度目标）\n", year)
		return
	}

	for _, yg := range yearGoals {
		fmt.Fprintf(b, "## 🎯 %s\n\n", yg.Title)
		if yg.Description != "" {
			fmt.Fprintf(b, "> %s\n\n", yg.Description)
		}

		// 季度目标
		quarterGoals, _ := s.ListQuarterGoals(store.QuarterGoalFilter{YearGoalID: &yg.ID})
		if len(quarterGoals) == 0 {
			fmt.Fprintln(b, "（无季度目标）")
			fmt.Fprintln(b)
			continue
		}
		for _, qg := range quarterGoals {
			fmt.Fprintf(b, "### 🏆 %s（%s）\n\n", qg.Title, util.QuarterLabel(qg.Year, qg.Quarter))

			// 周目标
			weekGoals, _ := s.ListWeekGoals(store.WeekGoalFilter{QuarterGoalID: &qg.ID})
			if len(weekGoals) == 0 {
				fmt.Fprintln(b, "（无周目标）")
				fmt.Fprintln(b)
				continue
			}
			for _, wg := range weekGoals {
				fmt.Fprintf(b, "#### 📅 %s（%s）\n\n", wg.Title, util.WeekLabel(wg.Year, wg.Week))
				// 任务
				tasks, _ := s.ListTasks(store.TaskFilter{WeekGoalID: &wg.ID})
				for _, t := range tasks {
					mark := "[ ]"
					if t.Status == model.TaskStatusDone {
						mark = "[x]"
					}
					due := ""
					if t.DueDate != nil {
						due = fmt.Sprintf("（截止 %s）", util.FormatDate(*t.DueDate))
					}
					fmt.Fprintf(b, "- %s %s %s\n", mark, t.Title, due)
				}
				fmt.Fprintln(b)
			}
		}
	}
}

// writeTaskStats 写入任务统计。
func writeTaskStats(b *strings.Builder, s *store.Store) {
	pending, _ := s.CountTasks(store.TaskFilter{Status: "pending"})
	done, _ := s.CountTasks(store.TaskFilter{Status: "done"})
	total := pending + done
	fmt.Fprintln(b, "## 📊 统计")
	fmt.Fprintf(b, "- 总任务：%d（待办 %d，已完成 %d）\n", total, pending, done)
	if total > 0 {
		fmt.Fprintf(b, "- 完成率：%d%%\n", done*100/total)
	}
}

// priorityLabel 返回优先级的中文标签。
func priorityLabel(p string) string {
	switch p {
	case model.PriorityHigh:
		return "高优先"
	case model.PriorityMedium:
		return "中优先"
	case model.PriorityLow:
		return "低优先"
	default:
		return p
	}
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "输出文件路径（默认 stdout）")
	exportCmd.Flags().StringVar(&exportScope, "scope", "week", "导出范围：today/week/quarter/year/all")
	rootCmd.AddCommand(exportCmd)
}
