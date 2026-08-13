package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"goal-tracker/internal/model"
)

// TaskFilter 用于 ListTasks 的查询过滤。
type TaskFilter struct {
	Status     string // pending / done / all（空或 all 表示不过滤）
	DueDate    *time.Time
	DueBefore  *time.Time // 含义：截止日期 < 此值（用于查"过期"）
	WeekGoalID *int64
}

// CreateTaskInput 创建任务的输入参数。
type CreateTaskInput struct {
	Title      string
	DueDate    *time.Time
	Priority   string // 默认 medium
	WeekGoalID *int64
}

// UpdateTaskInput 更新任务的输入参数。
// 所有字段为指针类型，nil 表示不更新该字段。
type UpdateTaskInput struct {
	Title      *string
	DueDate    *time.Time // 传 &time.Time{} 表示清空；传 nil 表示不更新
	ClearDue   bool       // true 表示把 due_date 设为 NULL
	Priority   *string
	WeekGoalID *int64
	ClearLink  bool // true 表示把 week_goal_id 设为 NULL
}

// CreateTask 创建一个任务，返回新建的任务（含 ID）。
func (s *Store) CreateTask(in CreateTaskInput) (*model.Task, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, fmt.Errorf("任务标题不能为空")
	}
	priority := in.Priority
	if priority == "" {
		priority = model.PriorityMedium
	}
	if !isValidPriority(priority) {
		return nil, fmt.Errorf("无效的优先级: %s", priority)
	}

	// 规范化 due_date 到零点
	var duePtr any
	if in.DueDate != nil {
		d := truncateToDate(*in.DueDate)
		duePtr = d
	}
	var weekGoalPtr any
	if in.WeekGoalID != nil {
		weekGoalPtr = *in.WeekGoalID
	}

	res, err := s.db.Exec(
		`INSERT INTO tasks (title, due_date, priority, week_goal_id, status)
		 VALUES (?, ?, ?, ?, 'pending')`,
		in.Title, duePtr, priority, weekGoalPtr,
	)
	if err != nil {
		return nil, fmt.Errorf("插入任务失败: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取任务 ID 失败: %w", err)
	}

	return s.GetTask(id)
}

// GetTask 按 ID 查询单个任务。不存在返回 (nil, nil)。
func (s *Store) GetTask(id int64) (*model.Task, error) {
	row := s.db.QueryRow(
		`SELECT id, title, due_date, priority, week_goal_id, status, created_at, updated_at
		 FROM tasks WHERE id = ?`, id,
	)
	t, err := scanTask(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ListTasks 根据过滤条件查询任务列表。
// 结果按"过期优先、截止日期升序、优先级降序"排序。
func (s *Store) ListTasks(filter TaskFilter) ([]model.Task, error) {
	q := "SELECT id, title, due_date, priority, week_goal_id, status, created_at, updated_at FROM tasks"
	var (
		clauses []string
		args    []any
	)

	if filter.Status != "" && filter.Status != "all" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.DueDate != nil {
		d := truncateToDate(*filter.DueDate)
		clauses = append(clauses, "due_date = ?")
		args = append(args, d)
	}
	if filter.DueBefore != nil {
		d := truncateToDate(*filter.DueBefore)
		clauses = append(clauses, "(due_date IS NOT NULL AND due_date < ?)")
		args = append(args, d)
	}
	if filter.WeekGoalID != nil {
		clauses = append(clauses, "week_goal_id = ?")
		args = append(args, *filter.WeekGoalID)
	}

	if len(clauses) > 0 {
		q += " WHERE " + strings.Join(clauses, " AND ")
	}

	// 排序：未完成在前 → 有截止日期按升序 → 优先级降序
	q += " ORDER BY CASE status WHEN 'done' THEN 1 ELSE 0 END," +
		" CASE WHEN due_date IS NULL THEN 1 ELSE 0 END," +
		" due_date ASC," +
		" CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END," +
		" id ASC"

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("查询任务列表失败: %w", err)
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// UpdateTask 按 ID 更新任务。返回更新后的任务。
func (s *Store) UpdateTask(id int64, in UpdateTaskInput) (*model.Task, error) {
	// 先确认存在
	existing, err := s.GetTask(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("任务 %d 不存在", id)
	}

	var (
		sets   []string
		args   []any
		changed bool
	)

	if in.Title != nil {
		if strings.TrimSpace(*in.Title) == "" {
			return nil, fmt.Errorf("任务标题不能为空")
		}
		sets = append(sets, "title = ?")
		args = append(args, *in.Title)
		changed = true
	}
	if in.ClearDue {
		sets = append(sets, "due_date = NULL")
		changed = true
	} else if in.DueDate != nil {
		d := truncateToDate(*in.DueDate)
		sets = append(sets, "due_date = ?")
		args = append(args, d)
		changed = true
	}
	if in.Priority != nil {
		if !isValidPriority(*in.Priority) {
			return nil, fmt.Errorf("无效的优先级: %s", *in.Priority)
		}
		sets = append(sets, "priority = ?")
		args = append(args, *in.Priority)
		changed = true
	}
	if in.ClearLink {
		sets = append(sets, "week_goal_id = NULL")
		changed = true
	} else if in.WeekGoalID != nil {
		sets = append(sets, "week_goal_id = ?")
		args = append(args, *in.WeekGoalID)
		changed = true
	}

	if !changed {
		// 没有字段需要更新，直接返回原数据
		return existing, nil
	}

	sets = append(sets, "updated_at = datetime('now')")
	args = append(args, id)

	q := "UPDATE tasks SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	if _, err := s.db.Exec(q, args...); err != nil {
		return nil, fmt.Errorf("更新任务失败: %w", err)
	}
	return s.GetTask(id)
}

// SetTaskStatus 设置任务状态（pending / done）。
func (s *Store) SetTaskStatus(id int64, status string) (*model.Task, error) {
	if status != model.TaskStatusPending && status != model.TaskStatusDone {
		return nil, fmt.Errorf("无效的任务状态: %s", status)
	}
	existing, err := s.GetTask(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("任务 %d 不存在", id)
	}
	if _, err := s.db.Exec(
		"UPDATE tasks SET status = ?, updated_at = datetime('now') WHERE id = ?",
		status, id,
	); err != nil {
		return nil, fmt.Errorf("更新任务状态失败: %w", err)
	}
	return s.GetTask(id)
}

// LinkTask 将任务关联到周目标。
func (s *Store) LinkTask(taskID, weekGoalID int64) (*model.Task, error) {
	existing, err := s.GetTask(taskID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("任务 %d 不存在", taskID)
	}
	// 校验周目标存在（外键也会拦，但提前给友好错误）
	var exists int
	if err := s.db.QueryRow(
		"SELECT 1 FROM week_goals WHERE id = ?", weekGoalID,
	).Scan(&exists); err == sql.ErrNoRows {
		return nil, fmt.Errorf("周目标 %d 不存在", weekGoalID)
	} else if err != nil {
		return nil, err
	}
	return s.UpdateTask(taskID, UpdateTaskInput{WeekGoalID: &weekGoalID})
}

// DeleteTask 删除任务。返回是否真的删除了一行。
func (s *Store) DeleteTask(id int64) (bool, error) {
	res, err := s.db.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("删除任务失败: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// CountTasks 返回任务总数（按过滤条件）。
func (s *Store) CountTasks(filter TaskFilter) (int, error) {
	q := "SELECT COUNT(*) FROM tasks"
	var (
		clauses []string
		args    []any
	)
	if filter.Status != "" && filter.Status != "all" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.WeekGoalID != nil {
		clauses = append(clauses, "week_goal_id = ?")
		args = append(args, *filter.WeekGoalID)
	}
	if len(clauses) > 0 {
		q += " WHERE " + strings.Join(clauses, " AND ")
	}
	var n int
	if err := s.db.QueryRow(q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// ---------- 内部工具 ----------

// scanner 兼容 *sql.Row 和 *sql.Rows 的 Scan 方法。
type scanner interface {
	Scan(dest ...any) error
}

func scanTask(s scanner) (*model.Task, error) {
	var (
		t          model.Task
		dueDate    sql.NullTime
		weekGoalID sql.NullInt64
		createdAt  sql.NullTime
		updatedAt  sql.NullTime
	)
	err := s.Scan(&t.ID, &t.Title, &dueDate, &t.Priority, &weekGoalID, &t.Status, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if dueDate.Valid {
		t.DueDate = ptrTime(dueDate.Time)
	}
	if weekGoalID.Valid {
		id := weekGoalID.Int64
		t.WeekGoalID = &id
	}
	if createdAt.Valid {
		t.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		t.UpdatedAt = updatedAt.Time
	}
	return &t, nil
}

func ptrTime(t time.Time) *time.Time { return &t }

func truncateToDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func isValidPriority(p string) bool {
	switch p {
	case model.PriorityHigh, model.PriorityMedium, model.PriorityLow:
		return true
	default:
		return false
	}
}
