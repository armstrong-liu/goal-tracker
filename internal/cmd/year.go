package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"goal-tracker/internal/model"
	"goal-tracker/internal/store"
)

// ---------- gt year ----------

var yearCmd = &cobra.Command{
	Use:   "year",
	Short: "🎯 年度目标管理",
}

// gt year add
var (
	yearAddTitle string
	yearAddYear  int
	yearAddDesc  string
)

var yearAddCmd = &cobra.Command{
	Use:   "add <标题>",
	Short: "添加年度目标",
	Example: `  gt year add "晋升P7"                       # 默认当前年
  gt year add "读完12本书" -d "每月一本"     # 带描述`,
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

		y := time.Now().Year()
		if cmd.Flags().Changed("year") {
			y = yearAddYear
		}

		yg, err := s.CreateYearGoal(store.CreateYearGoalInput{
			Title: title, Year: y, Description: yearAddDesc,
		})
		if err != nil {
			return err
		}
		PrintSuccess("已添加年度目标 #%d：%s", yg.ID, yg.Title)
		fmt.Printf("  年份：%d\n", yg.Year)
		return nil
	},
}

// gt year view
var yearViewYear int

var yearViewCmd = &cobra.Command{
	Use:   "view",
	Short: "查看年度目标（含季度目标进度）",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()

		y := time.Now().Year()
		if cmd.Flags().Changed("year") {
			y = yearViewYear
		}

		list, err := s.ListYearGoalsWithProgress(store.YearGoalFilter{Year: y})
		if err != nil {
			return err
		}

		PrintTitle(fmt.Sprintf("🎯 %d 年度目标（%d 项）", y, len(list)))
		if len(list) == 0 {
			PrintSubtitle("（本年还没有目标，用 'gt year add' 添加）")
			return nil
		}
		fmt.Println()
		for _, yg := range list {
			printYearGoalWithQuarters(yg)
		}
		return nil
	},
}

func printYearGoalWithQuarters(yg store.YearGoalWithProgress) {
	icon := GoalStatusIcon(yg.Status)
	progress := ProgressBar(yg.Progress())
	fmt.Printf("  %s #%d  %s    %s", icon, yg.ID, yg.Title, progress)
	if yg.HasProgress {
		fmt.Printf("  (%d/%d 季度)", yg.QuarterDone, yg.QuarterTotal)
	}
	fmt.Println()
	if yg.Description != "" {
		fmt.Printf("        %s\n", yg.Description)
	}
}

// gt year done
var yearDoneCmd = &cobra.Command{
	Use:   "done <year_goal_id>",
	Short: "标记年度目标完成",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := strconv.ParseInt(args[0], 10, 64)
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()
		yg, err := s.SetYearGoalStatus(id, model.YearGoalStatusCompleted)
		if err != nil {
			return err
		}
		PrintSuccess("已完成年度目标 #%d：%s", yg.ID, yg.Title)
		return nil
	},
}

// gt year edit
var (
	yearEditTitle string
	yearEditDesc  string
)

var yearEditCmd = &cobra.Command{
	Use:   "edit <year_goal_id>",
	Short: "编辑年度目标",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := strconv.ParseInt(args[0], 10, 64)
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()

		in := store.UpdateYearGoalInput{}
		changed := false
		if cmd.Flags().Changed("title") {
			in.Title = &yearEditTitle
			changed = true
		}
		if cmd.Flags().Changed("desc") {
			in.Description = &yearEditDesc
			changed = true
		}
		if !changed {
			PrintSubtitle("没有要更新的字段")
			return nil
		}
		yg, err := s.UpdateYearGoal(id, in)
		if err != nil {
			return err
		}
		PrintSuccess("已更新年度目标 #%d", yg.ID)
		return nil
	},
}

// gt year delete
var yearDeleteForce bool

var yearDeleteCmd = &cobra.Command{
	Use:   "delete <year_goal_id>",
	Short: "删除年度目标",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := strconv.ParseInt(args[0], 10, 64)
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()

		yg, _ := s.GetYearGoal(id)
		if yg == nil {
			return fmt.Errorf("年度目标 #%d 不存在", id)
		}
		if !yearDeleteForce {
			fmt.Printf("将删除年度目标 #%d：%s\n", id, yg.Title)
			if !confirm("确认删除？") {
				PrintSubtitle("已取消")
				return nil
			}
		}
		ok, err := s.DeleteYearGoal(id)
		if err != nil {
			return err
		}
		if ok {
			PrintSuccess("已删除年度目标 #%d", id)
		}
		return nil
	},
}

func init() {
	yearAddCmd.Flags().IntVarP(&yearAddYear, "year", "y", 0, "年（默认当前）")
	yearAddCmd.Flags().StringVarP(&yearAddDesc, "desc", "d", "", "描述")
	yearCmd.AddCommand(yearAddCmd)

	yearViewCmd.Flags().IntVarP(&yearViewYear, "year", "y", 0, "年（默认当前）")
	yearCmd.AddCommand(yearViewCmd)

	yearCmd.AddCommand(yearDoneCmd)

	yearEditCmd.Flags().StringVarP(&yearEditTitle, "title", "t", "", "新标题")
	yearEditCmd.Flags().StringVarP(&yearEditDesc, "desc", "d", "", "描述")
	yearCmd.AddCommand(yearEditCmd)

	yearDeleteCmd.Flags().BoolVarP(&yearDeleteForce, "force", "f", false, "跳过确认")
	yearCmd.AddCommand(yearDeleteCmd)

	rootCmd.AddCommand(yearCmd)
}
