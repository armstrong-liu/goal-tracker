package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"goal-tracker/internal/store"
	"goal-tracker/internal/util"
)

// todayCmd: gt today —— 今日概览。
// 展示：今日到期任务 + 过期任务 + 简单统计。
var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "🌅 今日概览",
	Long:  "查看今日到期的任务、已过期的未完成任务，以及简单的统计。",
	RunE:  runToday,
}

func runToday(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	now := time.Now()
	y, w := util.ISOWeek(now)
	q := util.CurrentQuarter(now)

	// 头部信息
	PrintTitle(fmt.Sprintf("🌅 %s  (%s · %s)",
		now.Format("2006-01-02 Monday"),
		util.WeekLabel(y, w),
		util.QuarterLabel(now.Year(), q),
	))
	fmt.Println()

	// 1. 过期未完成任务（最高优先级提醒）
	overdueTasks, err := s.ListTasks(store.TaskFilter{
		Status:    "pending",
		DueBefore: &now,
	})
	if err != nil {
		return err
	}
	if len(overdueTasks) > 0 {
		PrintTitle(fmt.Sprintf("⚠️  已过期（%d 项）", len(overdueTasks)))
		PrintTaskList(overdueTasks, now)
	}

	// 2. 今日到期
	todayTasks, err := s.ListTasks(store.TaskFilter{
		Status:  "pending",
		DueDate: &now,
	})
	if err != nil {
		return err
	}
	PrintTitle(fmt.Sprintf("📅 今日到期（%d 项）", len(todayTasks)))
	PrintTaskList(todayTasks, now)

	// 3. 统计
	pendingCount, _ := s.CountTasks(store.TaskFilter{Status: "pending"})
	doneCount, _ := s.CountTasks(store.TaskFilter{Status: "done"})
	fmt.Println()
	PrintSubtitle(fmt.Sprintf("统计：共 %d 个任务，待办 %d，已完成 %d",
		pendingCount+doneCount, pendingCount, doneCount))

	return nil
}

func init() {
	rootCmd.AddCommand(todayCmd)
}
