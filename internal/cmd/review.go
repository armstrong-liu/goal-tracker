package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"goal-tracker/internal/model"
	"goal-tracker/internal/store"
	"goal-tracker/internal/util"
)

// gt review：周回顾
var (
	reviewYear int
	reviewWeek int
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "📊 周回顾",
	Long:  "回顾本周（或指定周）的目标完成情况和任务统计。",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := openStore()
		if err != nil {
			return err
		}
		defer s.Close()

		now := time.Now()
		y, w := util.ISOWeek(now)
		if cmd.Flags().Changed("year") {
			y = reviewYear
		}
		if cmd.Flags().Changed("week") {
			w = reviewWeek
		}

		// 1. 周目标
		weekGoals, err := s.ListWeekGoalsWithProgress(store.WeekGoalFilter{Year: y, Week: w})
		if err != nil {
			return err
		}

		PrintTitle(fmt.Sprintf("📊 %s 周回顾", util.WeekLabel(y, w)))
		fmt.Println()

		// 2. 周目标完成统计
		var completedGoals, totalGoals int
		for _, wg := range weekGoals {
			totalGoals++
			if wg.Status == model.WeekGoalStatusCompleted {
				completedGoals++
			}
		}

		// 3. 任务统计（本周周目标下的所有任务）
		var totalTasks, doneTasks int
		for _, wg := range weekGoals {
			if wg.HasProgress {
				totalTasks += wg.TaskTotal
				doneTasks += wg.TaskDone
			}
		}

		// 输出统计
		fmt.Printf("  📅 周次：%s\n", util.WeekLabel(y, w))
		fmt.Printf("  🎯 周目标：%d 个（已完成 %d）\n", totalGoals, completedGoals)
		if totalTasks > 0 {
			pct := doneTasks * 100 / totalTasks
			fmt.Printf("  ✅ 任务：%d/%d（%d%%）\n", doneTasks, totalTasks, pct)
		} else {
			fmt.Printf("  ✅ 任务：0\n")
		}
		fmt.Println()

		// 4. 各周目标详情
		if len(weekGoals) > 0 {
			PrintTitle("目标详情")
			for _, wg := range weekGoals {
				printWeekGoalWithTasks(s, wg, now)
			}
		}

		// 5. 未完成的周任务（不在周目标下的、本周到期的）
		// 简单提示
		return nil
	},
}

func init() {
	reviewCmd.Flags().IntVarP(&reviewYear, "year", "y", 0, "年（默认当前）")
	reviewCmd.Flags().IntVarP(&reviewWeek, "week", "w", 0, "周（默认当前）")
	rootCmd.AddCommand(reviewCmd)
}
