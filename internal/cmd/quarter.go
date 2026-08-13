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

// ---------- gt quarter ----------

var quarterCmd = &cobra.Command{
	Use:   "quarter",
	Short: "🏆 季度目标管理",
}

// gt quarter add
var (
	quarterAddYear        int
	quarterAddQuarter     int
	quarterAddDesc        string
	quarterAddYearGoal    int
	quarterAddHasYearGoal bool
)

var quarterAddCmd = &cobra.Command{
	Use:   "add <标题>",
	Short: "添加季度目标",
	Example: `  gt quarter add "项目A上线"                # 默认当前季度
  gt quarter add "Q4 重点" -q 4             # 指定季度
  gt quarter add "年度关键" -g 1 -d "描述"   # 关联年度目标`,
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
		y := now.Year()
		q := util.CurrentQuarter(now)
		if cmd.Flags().Changed("year") {
			y = quarterAddYear
		}
		if cmd.Flags().Changed("quarter") {
			q = quarterAddQuarter
		}

		in := store.CreateQuarterGoalInput{
			Title: title, Year: y, Quarter: q, Description: quarterAddDesc,
		}
		if cmd.Flags().Changed("year-goal") {
			yID := int64(quarterAddYearGoal)
			in.YearGoalID = &yID
		}

		qg, err := s.CreateQuarterGoal(in)
		if err != nil {
			return err
		}
		PrintSuccess("已添加季度目标 #%d：%s", qg.ID, qg.Title)
		fmt.Printf("  季度：%s\n", util.QuarterLabel(qg.Year, qg.Quarter))
		return nil
	},
}

// gt quarter view
var (
	quarterViewYear    int
	quarterViewQuarter int
)

var quarterViewCmd = &cobra.Command{
	Use:   "view",
	Short: "查看季度目标（含周目标进度）",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()

		now := time.Now()
		y := now.Year()
		q := util.CurrentQuarter(now)
		if cmd.Flags().Changed("year") {
			y = quarterViewYear
		}
		if cmd.Flags().Changed("quarter") {
			q = quarterViewQuarter
		}

		list, err := s.ListQuarterGoalsWithProgress(store.QuarterGoalFilter{Year: y, Quarter: q})
		if err != nil {
			return err
		}

		PrintTitle(fmt.Sprintf("🏆 %s 季度目标（%d 项）", util.QuarterLabel(y, q), len(list)))
		if len(list) == 0 {
			PrintSubtitle("（本季度还没有目标，用 'gt quarter add' 添加）")
			return nil
		}
		fmt.Println()
		for _, qg := range list {
			printQuarterGoalWithWeeks(qg)
		}
		return nil
	},
}

func printQuarterGoalWithWeeks(qg store.QuarterGoalWithProgress) {
	icon := GoalStatusIcon(qg.Status)
	progress := ProgressBar(qg.Progress())
	fmt.Printf("  %s #%d  %s    %s", icon, qg.ID, qg.Title, progress)
	if qg.HasProgress {
		fmt.Printf("  (%d/%d 周)", qg.WeekDone, qg.WeekTotal)
	}
	fmt.Println()
	if qg.Description != "" {
		fmt.Printf("        %s\n", qg.Description)
	}
}

// gt quarter done
var quarterDoneCmd = &cobra.Command{
	Use:   "done <quarter_goal_id>",
	Short: "标记季度目标完成",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := strconv.ParseInt(args[0], 10, 64)
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()
		qg, err := s.SetQuarterGoalStatus(id, model.QuarterGoalStatusCompleted)
		if err != nil {
			return err
		}
		PrintSuccess("已完成季度目标 #%d：%s", qg.ID, qg.Title)
		return nil
	},
}

// gt quarter edit
var (
	quarterEditTitle   string
	quarterEditDesc    string
	quarterEditYG      int
	quarterEditHasYG   bool
	quarterEditClearYG bool
)

var quarterEditCmd = &cobra.Command{
	Use:   "edit <quarter_goal_id>",
	Short: "编辑季度目标",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := strconv.ParseInt(args[0], 10, 64)
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()

		in := store.UpdateQuarterGoalInput{}
		changed := false
		if cmd.Flags().Changed("title") {
			in.Title = &quarterEditTitle
			changed = true
		}
		if cmd.Flags().Changed("desc") {
			in.Description = &quarterEditDesc
			changed = true
		}
		if quarterEditClearYG {
			in.ClearLink = true
			changed = true
		} else if cmd.Flags().Changed("year-goal") {
			yID := int64(quarterEditYG)
			in.YearGoalID = &yID
			changed = true
		}
		if !changed {
			PrintSubtitle("没有要更新的字段")
			return nil
		}
		qg, err := s.UpdateQuarterGoal(id, in)
		if err != nil {
			return err
		}
		PrintSuccess("已更新季度目标 #%d", qg.ID)
		return nil
	},
}

// gt quarter delete
var quarterDeleteForce bool

var quarterDeleteCmd = &cobra.Command{
	Use:   "delete <quarter_goal_id>",
	Short: "删除季度目标",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := strconv.ParseInt(args[0], 10, 64)
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()

		qg, _ := s.GetQuarterGoal(id)
		if qg == nil {
			return fmt.Errorf("季度目标 #%d 不存在", id)
		}
		if !quarterDeleteForce {
			fmt.Printf("将删除季度目标 #%d：%s\n", id, qg.Title)
			if !confirm("确认删除？") {
				PrintSubtitle("已取消")
				return nil
			}
		}
		ok, err := s.DeleteQuarterGoal(id)
		if err != nil {
			return err
		}
		if ok {
			PrintSuccess("已删除季度目标 #%d", id)
		}
		return nil
	},
}

func init() {
	quarterAddCmd.Flags().IntVarP(&quarterAddYear, "year", "y", 0, "年（默认当前）")
	quarterAddCmd.Flags().IntVarP(&quarterAddQuarter, "quarter", "q", 0, "季度 1-4（默认当前）")
	quarterAddCmd.Flags().StringVarP(&quarterAddDesc, "desc", "d", "", "描述")
	quarterAddCmd.Flags().IntVarP(&quarterAddYearGoal, "year-goal", "g", 0, "关联年度目标 ID")
	quarterAddCmd.Flags().BoolVar(&quarterAddHasYearGoal, "has-year-goal", false, "内部使用")
	quarterCmd.AddCommand(quarterAddCmd)

	quarterViewCmd.Flags().IntVarP(&quarterViewYear, "year", "y", 0, "年（默认当前）")
	quarterViewCmd.Flags().IntVarP(&quarterViewQuarter, "quarter", "q", 0, "季度（默认当前）")
	quarterCmd.AddCommand(quarterViewCmd)

	quarterCmd.AddCommand(quarterDoneCmd)

	quarterEditCmd.Flags().StringVarP(&quarterEditTitle, "title", "t", "", "新标题")
	quarterEditCmd.Flags().StringVarP(&quarterEditDesc, "desc", "d", "", "描述")
	quarterEditCmd.Flags().IntVarP(&quarterEditYG, "year-goal", "g", 0, "关联年度目标 ID")
	quarterEditCmd.Flags().BoolVar(&quarterEditHasYG, "has-year-goal", false, "内部使用")
	quarterEditCmd.Flags().BoolVar(&quarterEditClearYG, "clear-year-goal", false, "清除年度目标关联")
	quarterCmd.AddCommand(quarterEditCmd)

	quarterDeleteCmd.Flags().BoolVarP(&quarterDeleteForce, "force", "f", false, "跳过确认")
	quarterCmd.AddCommand(quarterDeleteCmd)

	rootCmd.AddCommand(quarterCmd)
}
