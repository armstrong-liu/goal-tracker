package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"goal-tracker/internal/model"
	"goal-tracker/internal/store"
	"goal-tracker/internal/util"
)

// taskCmd 是 "gt task" 父命令，下挂 add/list/done/undone/edit/delete/link 子命令。
var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "📝 任务管理",
	Long:  "管理 TODO 任务，包括添加、查看、完成、编辑、删除、关联周目标。",
}

// ---------- gt task add ----------

var (
	taskAddDue       string
	taskAddPriority  string
	taskAddWeekGoal  int64
	taskAddHasWeek   bool
)

var taskAddCmd = &cobra.Command{
	Use:   "add <标题>",
	Short: "添加一个新任务",
	Example: `  gt task add "写周报" --due today --priority high
  gt task add "复习英语" -d tomorrow -p medium
  gt task add "完成方案A" -d 2026-08-15 -w 3`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskAdd,
}

func runTaskAdd(cmd *cobra.Command, args []string) error {
	title := strings.TrimSpace(args[0])
	if title == "" {
		return fmt.Errorf("任务标题不能为空")
	}

	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	in := store.CreateTaskInput{
		Title:    title,
		Priority: taskAddPriority,
	}

	// 解析截止日期
	if taskAddDue != "" {
		d, err := util.ParseDate(taskAddDue, time.Now())
		if err != nil {
			return err
		}
		in.DueDate = &d
	}

	// 关联周目标（用 Changed 判断用户是否显式传了 -w）
	if cmd.Flags().Changed("week") {
		wID := int64(taskAddWeekGoal)
		in.WeekGoalID = &wID
	}

	task, err := s.CreateTask(in)
	if err != nil {
		return err
	}

	PrintSuccess("已添加任务 #%d：%s", task.ID, task.Title)
	if task.DueDate != nil {
		fmt.Printf("  截止日期：%s\n", util.FormatDate(*task.DueDate))
	}
	fmt.Printf("  优先级：%s\n", task.Priority)
	return nil
}

// ---------- gt task list ----------

var (
	taskListStatus   string
	taskListDue      string
	taskListWeekGoal int64
	taskListHasWeek  bool
)

var taskListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "查看任务列表",
	Example: `  gt task list                  # 所有未完成任务
  gt task list -s done          # 已完成任务
  gt task list -d today         # 今日到期
  gt task list -d overdue       # 已过期`,
	RunE: runTaskList,
}

func runTaskList(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	filter := store.TaskFilter{
		Status: taskListStatus,
	}

	now := time.Now()
	switch strings.ToLower(taskListDue) {
	case "":
		// 不过滤
	case "today":
		filter.DueDate = &now
	case "week":
		// 本周：从周一到周日
		start := now.AddDate(0, 0, -(int(now.Weekday())-1+7)%7)
		if now.Weekday() == time.Sunday {
			start = now.AddDate(0, 0, -6)
		} else {
			start = now.AddDate(0, 0, -(int(now.Weekday()) - 1))
		}
		_ = start // 简化处理：本周过滤暂时不实现（阶段3完善）
	case "overdue":
		filter.DueBefore = &now
	default:
		// 尝试作为日期解析
		d, err := util.ParseDate(taskListDue, now)
		if err != nil {
			return fmt.Errorf("--due 参数无效: %w", err)
		}
		filter.DueDate = &d
	}

	if cmd.Flags().Changed("week") {
		wID := int64(taskListWeekGoal)
		filter.WeekGoalID = &wID
	}

	tasks, err := s.ListTasks(filter)
	if err != nil {
		return err
	}

	PrintTitle(fmt.Sprintf("📋 任务列表（%d 项）", len(tasks)))
	PrintTaskList(tasks, now)
	return nil
}

// ---------- gt task done ----------

var taskDoneForce bool

var taskDoneCmd = &cobra.Command{
	Use:     "done <task_id>",
	Short:   "标记任务为已完成",
	Aliases: []string{"complete"},
	Args:    cobra.ExactArgs(1),
	RunE:    runTaskDone,
}

func runTaskDone(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("无效的任务 ID: %s", args[0])
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	task, err := s.SetTaskStatus(id, model.TaskStatusDone)
	if err != nil {
		return err
	}
	PrintSuccess("已完成任务 #%d：%s", task.ID, task.Title)
	return nil
}

// ---------- gt task undone ----------

var taskUndoneCmd = &cobra.Command{
	Use:   "undone <task_id>",
	Short: "将已完成的任务恢复为待办",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskUndone,
}

func runTaskUndone(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("无效的任务 ID: %s", args[0])
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	task, err := s.SetTaskStatus(id, model.TaskStatusPending)
	if err != nil {
		return err
	}
	PrintSuccess("已恢复任务 #%d：%s", task.ID, task.Title)
	return nil
}

// ---------- gt task edit ----------

var (
	taskEditTitle    string
	taskEditDue      string
	taskEditPriority string
	taskEditClearDue bool
)

var taskEditCmd = &cobra.Command{
	Use:     "edit <task_id>",
	Short:   "编辑任务",
	Aliases: []string{"update"},
	Example: `  gt task edit 3 --title "新标题"
  gt task edit 3 -d 2026-08-20 -p high
  gt task edit 3 --clear-due`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskEdit,
}

func runTaskEdit(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("无效的任务 ID: %s", args[0])
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	in := store.UpdateTaskInput{}
	changed := false

	if cmd.Flags().Changed("title") {
		in.Title = &taskEditTitle
		changed = true
	}
	if taskEditClearDue {
		in.ClearDue = true
		changed = true
	} else if cmd.Flags().Changed("due") {
		d, err := util.ParseDate(taskEditDue, time.Now())
		if err != nil {
			return err
		}
		in.DueDate = &d
		changed = true
	}
	if cmd.Flags().Changed("priority") {
		in.Priority = &taskEditPriority
		changed = true
	}

	if !changed {
		PrintSubtitle("没有要更新的字段")
		return nil
	}

	task, err := s.UpdateTask(id, in)
	if err != nil {
		return err
	}
	PrintSuccess("已更新任务 #%d", task.ID)
	fmt.Printf("  标题：%s\n", task.Title)
	if task.DueDate != nil {
		fmt.Printf("  截止：%s\n", util.FormatDate(*task.DueDate))
	}
	fmt.Printf("  优先级：%s\n", task.Priority)
	return nil
}

// ---------- gt task delete ----------

var taskDeleteForce bool

var taskDeleteCmd = &cobra.Command{
	Use:     "delete <task_id>",
	Short:   "删除任务",
	Aliases: []string{"rm"},
	Example: `  gt task delete 3
  gt task delete 3 -f    # 跳过确认`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDelete,
}

func runTaskDelete(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("无效的任务 ID: %s", args[0])
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	// 先查看任务是否存在，并展示给用户确认
	task, err := s.GetTask(id)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("任务 #%d 不存在", id)
	}

	// 确认
	if !taskDeleteForce {
		fmt.Printf("将删除任务 #%d：%s\n", id, task.Title)
		if !confirm("确认删除？") {
			PrintSubtitle("已取消")
			return nil
		}
	}

	deleted, err := s.DeleteTask(id)
	if err != nil {
		return err
	}
	if deleted {
		PrintSuccess("已删除任务 #%d", id)
	} else {
		PrintError("任务 #%d 不存在", id)
	}
	return nil
}

// ---------- gt task link ----------

var taskLinkCmd = &cobra.Command{
	Use:     "link <task_id> <week_goal_id>",
	Short:   "关联任务到周目标",
	Example: `  gt task link 3 1    # 把任务 #3 关联到周目标 #1`,
	Args:    cobra.ExactArgs(2),
	RunE:    runTaskLink,
}

func runTaskLink(cmd *cobra.Command, args []string) error {
	taskID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("无效的任务 ID: %s", args[0])
	}
	weekID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("无效的周目标 ID: %s", args[1])
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	task, err := s.LinkTask(taskID, weekID)
	if err != nil {
		return err
	}
	PrintSuccess("已关联任务 #%d → week#%d", task.ID, weekID)
	return nil
}

// ---------- 公共辅助 ----------

// openStore 根据 --db flag（或默认路径）打开数据库。
func openStore() (*store.Store, error) {
	dbPath, err := ResolveDBPath()
	if err != nil {
		return nil, err
	}
	return store.Open(dbPath)
}

// confirm 在终端询问用户 y/n。
func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')
	resp = strings.TrimSpace(strings.ToLower(resp))
	return resp == "y" || resp == "yes"
}

func init() {
	// 注册 task 子命令到 root
	rootCmd.AddCommand(taskCmd)

	// add
	taskAddCmd.Flags().StringVarP(&taskAddDue, "due", "d", "", "截止日期（today/tomorrow/monday..sunday 或 YYYY-MM-DD）")
	taskAddCmd.Flags().StringVarP(&taskAddPriority, "priority", "p", "medium", "优先级（high/medium/low）")
	taskAddCmd.Flags().Int64VarP(&taskAddWeekGoal, "week", "w", 0, "关联周目标 ID")
	taskAddCmd.Flags().BoolVar(&taskAddHasWeek, "has-week", false, "内部使用：标记 --week 已设置")
	taskCmd.AddCommand(taskAddCmd)

	// list
	taskListCmd.Flags().StringVarP(&taskListStatus, "status", "s", "pending", "状态过滤（pending/done/all）")
	taskListCmd.Flags().StringVarP(&taskListDue, "due", "d", "", "截止日期过滤（today/week/overdue 或 YYYY-MM-DD）")
	taskListCmd.Flags().Int64VarP(&taskListWeekGoal, "week", "w", 0, "按周目标 ID 过滤")
	taskListCmd.Flags().BoolVar(&taskListHasWeek, "has-week", false, "内部使用：标记 --week 已设置")
	taskCmd.AddCommand(taskListCmd)

	// done / undone
	taskCmd.AddCommand(taskDoneCmd)
	taskCmd.AddCommand(taskUndoneCmd)

	// edit
	taskEditCmd.Flags().StringVarP(&taskEditTitle, "title", "t", "", "新标题")
	taskEditCmd.Flags().StringVarP(&taskEditDue, "due", "d", "", "新截止日期")
	taskEditCmd.Flags().StringVarP(&taskEditPriority, "priority", "p", "", "新优先级")
	taskEditCmd.Flags().BoolVar(&taskEditClearDue, "clear-due", false, "清除截止日期")
	taskCmd.AddCommand(taskEditCmd)

	// delete
	taskDeleteCmd.Flags().BoolVarP(&taskDeleteForce, "force", "f", false, "跳过确认")
	taskCmd.AddCommand(taskDeleteCmd)

	// link
	taskCmd.AddCommand(taskLinkCmd)
}
