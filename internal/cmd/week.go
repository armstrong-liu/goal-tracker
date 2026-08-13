package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"goal-tracker/internal/model"
	"goal-tracker/internal/store"
	"goal-tracker/internal/util"
)

// ---------- gt week ----------

var weekCmd = &cobra.Command{
	Use:   "week",
	Short: "📅 周目标管理",
}

// gt week add
var (
	weekAddYear    int
	weekAddWeek    int
	weekAddQuarter int
	weekAddHasQ    bool
)

var weekAddCmd = &cobra.Command{
	Use:   "add <标题>",
	Short: "添加周目标",
	Example: `  gt week add "完成项目A方案"            # 默认当前周
  gt week add "下周重点" -w 34             # 指定周
  gt week add "Q3 关键" -q 2               # 关联季度目标`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.TrimSpace(args[0])
		if title == "" {
			return fmt.Errorf("标题不能为空")
		}
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()

		now := time.Now()
		y, w := util.ISOWeek(now)
		if cmd.Flags().Changed("year") {
			y = weekAddYear
		}
		if cmd.Flags().Changed("week") {
			w = weekAddWeek
		}

		in := store.CreateWeekGoalInput{Title: title, Year: y, Week: w}
		if cmd.Flags().Changed("quarter") {
			qID := int64(weekAddQuarter)
			in.QuarterGoalID = &qID
		}

		wg, err := s.CreateWeekGoal(in)
		if err != nil {
			return err
		}
		PrintSuccess("已添加周目标 #%d：%s", wg.ID, wg.Title)
		fmt.Printf("  周次：%s\n", util.WeekLabel(wg.Year, wg.Week))
		return nil
	},
}

// gt week view
var (
	weekViewYear int
	weekViewWeek int
)

var weekViewCmd = &cobra.Command{
	Use:   "view",
	Short: "查看周目标（含任务进度）",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()

		now := time.Now()
		y, w := util.ISOWeek(now)
		if cmd.Flags().Changed("year") {
			y = weekViewYear
		}
		if cmd.Flags().Changed("week") {
			w = weekViewWeek
		}

		list, err := s.ListWeekGoalsWithProgress(store.WeekGoalFilter{Year: y, Week: w})
		if err != nil {
			return err
		}

		PrintTitle(fmt.Sprintf("📅 %s 周目标（%d 项）", util.WeekLabel(y, w), len(list)))
		if len(list) == 0 {
			PrintSubtitle("（本周还没有目标，用 'gt week add' 添加）")
			return nil
		}
		fmt.Println()
		for _, wg := range list {
			printWeekGoalWithTasks(s, wg, now)
		}
		return nil
	},
}

// printWeekGoalWithTasks 打印一个周目标 + 其下的任务列表。
func printWeekGoalWithTasks(s *store.Store, wg store.WeekGoalWithProgress, now time.Time) {
	icon := GoalStatusIcon(wg.Status)
	progress := ProgressBar(wg.Progress())
	fmt.Printf("  %s #%d  %s    %s", icon, wg.ID, wg.Title, progress)
	if wg.HasProgress {
		fmt.Printf("  (%d/%d)", wg.TaskDone, wg.TaskTotal)
	}
	fmt.Println()

	// 展示关联任务
	if wg.HasProgress {
		tasks, err := s.ListTasks(store.TaskFilter{WeekGoalID: &wg.ID})
		if err == nil {
			for _, t := range tasks {
				mark := "□"
				if t.Status == model.TaskStatusDone {
					mark = "☑"
				}
				fmt.Printf("        %s %s\n", mark, t.Title)
			}
		}
	}
	fmt.Println()
}

// gt week done
var weekDoneCmd = &cobra.Command{
	Use:   "done <week_goal_id>",
	Short: "标记周目标完成",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("无效 ID: %s", args[0])
		}
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()

		wg, err := s.SetWeekGoalStatus(id, model.WeekGoalStatusCompleted)
		if err != nil {
			return err
		}
		PrintSuccess("已完成周目标 #%d：%s", wg.ID, wg.Title)
		return nil
	},
}

// gt week edit
var (
	weekEditTitle   string
	weekEditQuarter int
	weekEditHasQ    bool
	weekEditClearQ  bool
)

var weekEditCmd = &cobra.Command{
	Use:   "edit <week_goal_id>",
	Short: "编辑周目标",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("无效 ID: %s", args[0])
		}
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()

		in := store.UpdateWeekGoalInput{}
		changed := false
		if cmd.Flags().Changed("title") {
			in.Title = &weekEditTitle
			changed = true
		}
		if weekEditClearQ {
			in.ClearLink = true
			changed = true
		} else if cmd.Flags().Changed("quarter") {
			qID := int64(weekEditQuarter)
			in.QuarterGoalID = &qID
			changed = true
		}
		if !changed {
			PrintSubtitle("没有要更新的字段")
			return nil
		}
		wg, err := s.UpdateWeekGoal(id, in)
		if err != nil {
			return err
		}
		PrintSuccess("已更新周目标 #%d：%s", wg.ID, wg.Title)
		return nil
	},
}

// gt week delete
var weekDeleteForce bool

var weekDeleteCmd = &cobra.Command{
	Use:   "delete <week_goal_id>",
	Short: "删除周目标",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("无效 ID: %s", args[0])
		}
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()

		wg, _ := s.GetWeekGoal(id)
		if wg == nil {
			return fmt.Errorf("周目标 #%d 不存在", id)
		}
		if !weekDeleteForce {
			fmt.Printf("将删除周目标 #%d：%s\n", id, wg.Title)
			if !confirm("确认删除？") {
				PrintSubtitle("已取消")
				return nil
			}
		}
		ok, err := s.DeleteWeekGoal(id)
		if err != nil {
			return err
		}
		if ok {
			PrintSuccess("已删除周目标 #%d", id)
		}
		return nil
	},
}

func init() {
	// add
	weekAddCmd.Flags().IntVarP(&weekAddYear, "year", "y", 0, "年（默认当前）")
	weekAddCmd.Flags().IntVarP(&weekAddWeek, "week", "w", 0, "周（默认当前）")
	weekAddCmd.Flags().IntVarP(&weekAddQuarter, "quarter", "q", 0, "关联季度目标 ID")
	weekAddCmd.Flags().BoolVar(&weekAddHasQ, "has-quarter", false, "内部使用")
	weekCmd.AddCommand(weekAddCmd)

	// view
	weekViewCmd.Flags().IntVarP(&weekViewYear, "year", "y", 0, "年（默认当前）")
	weekViewCmd.Flags().IntVarP(&weekViewWeek, "week", "w", 0, "周（默认当前）")
	weekCmd.AddCommand(weekViewCmd)

	// done
	weekCmd.AddCommand(weekDoneCmd)

	// edit
	weekEditCmd.Flags().StringVarP(&weekEditTitle, "title", "t", "", "新标题")
	weekEditCmd.Flags().IntVarP(&weekEditQuarter, "quarter", "q", 0, "关联季度目标 ID")
	weekEditCmd.Flags().BoolVar(&weekEditHasQ, "has-quarter", false, "内部使用")
	weekEditCmd.Flags().BoolVar(&weekEditClearQ, "clear-quarter", false, "清除季度目标关联")
	weekCmd.AddCommand(weekEditCmd)

	// delete
	weekDeleteCmd.Flags().BoolVarP(&weekDeleteForce, "force", "f", false, "跳过确认")
	weekCmd.AddCommand(weekDeleteCmd)

	rootCmd.AddCommand(weekCmd)
}
