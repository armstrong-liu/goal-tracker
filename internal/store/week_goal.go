package store

import (
	"database/sql"
	"fmt"
	"strings"

	"goal-tracker/internal/model"
)

// WeekGoalFilter 周目标查询过滤。
type WeekGoalFilter struct {
	Year           int
	Week           int
	QuarterGoalID  *int64
	Status         string // active / completed / all
}

// CreateWeekGoalInput 创建周目标的输入。
type CreateWeekGoalInput struct {
	Title         string
	Year          int
	Week          int
	QuarterGoalID *int64
}

// UpdateWeekGoalInput 更新周目标（nil 字段表示不更新）。
type UpdateWeekGoalInput struct {
	Title         *string
	QuarterGoalID *int64
	ClearLink     bool
}

// WeekGoalWithProgress 周目标 + 进度信息。
type WeekGoalWithProgress struct {
	model.WeekGoal
	TaskTotal     int  // 关联任务总数
	TaskDone      int  // 已完成任务数
	HasProgress   bool // 是否有关联任务（false 时进度无意义）
}

// Progress 返回进度百分比（0-100）。无关联任务时返回 -1。
func (w WeekGoalWithProgress) Progress() int {
	if !w.HasProgress || w.TaskTotal == 0 {
		return -1
	}
	return w.TaskDone * 100 / w.TaskTotal
}

// CreateWeekGoal 创建周目标。
func (s *Store) CreateWeekGoal(in CreateWeekGoalInput) (*model.WeekGoal, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, fmt.Errorf("周目标标题不能为空")
	}
	if in.Week < 1 || in.Week > 53 {
		return nil, fmt.Errorf("无效的周数: %d", in.Week)
	}
	var qgPtr any
	if in.QuarterGoalID != nil {
		qgPtr = *in.QuarterGoalID
	}
	res, err := s.db.Exec(
		`INSERT INTO week_goals (title, year, week, quarter_goal_id)
		 VALUES (?, ?, ?, ?)`,
		in.Title, in.Year, in.Week, qgPtr,
	)
	if err != nil {
		return nil, fmt.Errorf("插入周目标失败: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetWeekGoal(id)
}

// GetWeekGoal 按 ID 查询周目标。不存在返回 (nil, nil)。
func (s *Store) GetWeekGoal(id int64) (*model.WeekGoal, error) {
	row := s.db.QueryRow(
		`SELECT id, title, year, week, quarter_goal_id, status, created_at, updated_at
		 FROM week_goals WHERE id = ?`, id,
	)
	w, err := scanWeekGoal(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

// ListWeekGoals 按过滤条件查询周目标，按 week 升序。
func (s *Store) ListWeekGoals(filter WeekGoalFilter) ([]model.WeekGoal, error) {
	q := `SELECT id, title, year, week, quarter_goal_id, status, created_at, updated_at
	      FROM week_goals`
	var clauses, args = buildWhere(filter)
	if len(clauses) > 0 {
		q += " WHERE " + strings.Join(clauses, " AND ")
	}
	q += " ORDER BY year ASC, week ASC, id ASC"

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("查询周目标失败: %w", err)
	}
	defer rows.Close()

	var out []model.WeekGoal
	for rows.Next() {
		w, err := scanWeekGoal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

// ListWeekGoalsWithProgress 查询周目标并附带任务进度。
func (s *Store) ListWeekGoalsWithProgress(filter WeekGoalFilter) ([]WeekGoalWithProgress, error) {
	goals, err := s.ListWeekGoals(filter)
	if err != nil {
		return nil, err
	}
	out := make([]WeekGoalWithProgress, 0, len(goals))
	for _, g := range goals {
		w := WeekGoalWithProgress{WeekGoal: g}
		total, done, err := s.countTasksForWeek(g.ID)
		if err != nil {
			return nil, err
		}
		w.TaskTotal = total
		w.TaskDone = done
		w.HasProgress = total > 0
		out = append(out, w)
	}
	return out, nil
}

// countTasksForWeek 返回 (总数, 已完成数)。
func (s *Store) countTasksForWeek(weekGoalID int64) (int, int, error) {
	var total, done int
	if err := s.db.QueryRow(
		"SELECT COUNT(*) FROM tasks WHERE week_goal_id = ?", weekGoalID,
	).Scan(&total); err != nil {
		return 0, 0, err
	}
	if err := s.db.QueryRow(
		"SELECT COUNT(*) FROM tasks WHERE week_goal_id = ? AND status = 'done'",
		weekGoalID,
	).Scan(&done); err != nil {
		return 0, 0, err
	}
	return total, done, nil
}

// UpdateWeekGoal 更新周目标。
func (s *Store) UpdateWeekGoal(id int64, in UpdateWeekGoalInput) (*model.WeekGoal, error) {
	existing, err := s.GetWeekGoal(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("周目标 %d 不存在", id)
	}
	var (
		sets    []string
		args    []any
		changed bool
	)
	if in.Title != nil {
		if strings.TrimSpace(*in.Title) == "" {
			return nil, fmt.Errorf("标题不能为空")
		}
		sets = append(sets, "title = ?")
		args = append(args, *in.Title)
		changed = true
	}
	if in.ClearLink {
		sets = append(sets, "quarter_goal_id = NULL")
		changed = true
	} else if in.QuarterGoalID != nil {
		sets = append(sets, "quarter_goal_id = ?")
		args = append(args, *in.QuarterGoalID)
		changed = true
	}
	if !changed {
		return existing, nil
	}
	sets = append(sets, "updated_at = datetime('now')")
	args = append(args, id)
	if _, err := s.db.Exec("UPDATE week_goals SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
		return nil, fmt.Errorf("更新周目标失败: %w", err)
	}
	return s.GetWeekGoal(id)
}

// SetWeekGoalStatus 设置周目标状态。
func (s *Store) SetWeekGoalStatus(id int64, status string) (*model.WeekGoal, error) {
	if status != model.WeekGoalStatusActive && status != model.WeekGoalStatusCompleted {
		return nil, fmt.Errorf("无效状态: %s", status)
	}
	existing, err := s.GetWeekGoal(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("周目标 %d 不存在", id)
	}
	if _, err := s.db.Exec(
		"UPDATE week_goals SET status = ?, updated_at = datetime('now') WHERE id = ?",
		status, id,
	); err != nil {
		return nil, fmt.Errorf("更新状态失败: %w", err)
	}
	return s.GetWeekGoal(id)
}

// DeleteWeekGoal 删除周目标。
func (s *Store) DeleteWeekGoal(id int64) (bool, error) {
	res, err := s.db.Exec("DELETE FROM week_goals WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("删除周目标失败: %w", err)
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// LinkWeekGoal 关联周目标到季度目标。
func (s *Store) LinkWeekGoal(weekGoalID, quarterGoalID int64) (*model.WeekGoal, error) {
	existing, err := s.GetWeekGoal(weekGoalID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("周目标 %d 不存在", weekGoalID)
	}
	var exists int
	if err := s.db.QueryRow(
		"SELECT 1 FROM quarter_goals WHERE id = ?", quarterGoalID,
	).Scan(&exists); err == sql.ErrNoRows {
		return nil, fmt.Errorf("季度目标 %d 不存在", quarterGoalID)
	} else if err != nil {
		return nil, err
	}
	return s.UpdateWeekGoal(weekGoalID, UpdateWeekGoalInput{QuarterGoalID: &quarterGoalID})
}

// ---------- 内部辅助 ----------

func buildWhere(f WeekGoalFilter) ([]string, []any) {
	var clauses []string
	var args []any
	if f.Year > 0 {
		clauses = append(clauses, "year = ?")
		args = append(args, f.Year)
	}
	if f.Week > 0 {
		clauses = append(clauses, "week = ?")
		args = append(args, f.Week)
	}
	if f.QuarterGoalID != nil {
		clauses = append(clauses, "quarter_goal_id = ?")
		args = append(args, *f.QuarterGoalID)
	}
	if f.Status != "" && f.Status != "all" {
		clauses = append(clauses, "status = ?")
		args = append(args, f.Status)
	}
	return clauses, args
}

func scanWeekGoal(s scanner) (*model.WeekGoal, error) {
	var (
		w             model.WeekGoal
		quarterGoalID sql.NullInt64
		createdAt     sql.NullTime
		updatedAt     sql.NullTime
	)
	err := s.Scan(&w.ID, &w.Title, &w.Year, &w.Week, &quarterGoalID, &w.Status, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if quarterGoalID.Valid {
		id := quarterGoalID.Int64
		w.QuarterGoalID = &id
	}
	if createdAt.Valid {
		w.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		w.UpdatedAt = updatedAt.Time
	}
	return &w, nil
}
