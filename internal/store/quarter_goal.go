package store

import (
	"database/sql"
	"fmt"
	"strings"

	"goal-tracker/internal/model"
)

// QuarterGoalFilter 季度目标查询过滤。
type QuarterGoalFilter struct {
	Year       int
	Quarter    int
	YearGoalID *int64
	Status     string // active / completed / archived / all
}

// CreateQuarterGoalInput 创建季度目标的输入。
type CreateQuarterGoalInput struct {
	Title       string
	Year        int
	Quarter     int
	Description string
	YearGoalID  *int64
}

// UpdateQuarterGoalInput 更新季度目标。
type UpdateQuarterGoalInput struct {
	Title       *string
	Description *string
	YearGoalID  *int64
	ClearLink   bool
}

// QuarterGoalWithProgress 季度目标 + 进度。
type QuarterGoalWithProgress struct {
	model.QuarterGoal
	WeekTotal   int  // 关联周目标总数
	WeekDone    int  // 已完成周目标数
	HasProgress bool
}

// Progress 返回进度百分比，无关联返回 -1。
func (q QuarterGoalWithProgress) Progress() int {
	if !q.HasProgress || q.WeekTotal == 0 {
		return -1
	}
	return q.WeekDone * 100 / q.WeekTotal
}

// CreateQuarterGoal 创建季度目标。
func (s *Store) CreateQuarterGoal(in CreateQuarterGoalInput) (*model.QuarterGoal, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, fmt.Errorf("季度目标标题不能为空")
	}
	if in.Quarter < 1 || in.Quarter > 4 {
		return nil, fmt.Errorf("无效的季度: %d", in.Quarter)
	}
	var ygPtr any
	if in.YearGoalID != nil {
		ygPtr = *in.YearGoalID
	}
	res, err := s.db.Exec(
		`INSERT INTO quarter_goals (title, year, quarter, description, year_goal_id)
		 VALUES (?, ?, ?, ?, ?)`,
		in.Title, in.Year, in.Quarter, in.Description, ygPtr,
	)
	if err != nil {
		return nil, fmt.Errorf("插入季度目标失败: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetQuarterGoal(id)
}

// GetQuarterGoal 按 ID 查询。不存在返回 (nil, nil)。
func (s *Store) GetQuarterGoal(id int64) (*model.QuarterGoal, error) {
	row := s.db.QueryRow(
		`SELECT id, title, year, quarter, description, year_goal_id, status, created_at, updated_at
		 FROM quarter_goals WHERE id = ?`, id,
	)
	q, err := scanQuarterGoal(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return q, nil
}

// ListQuarterGoals 按过滤条件查询。
func (s *Store) ListQuarterGoals(filter QuarterGoalFilter) ([]model.QuarterGoal, error) {
	q := `SELECT id, title, year, quarter, description, year_goal_id, status, created_at, updated_at
	      FROM quarter_goals`
	clauses, args := buildQuarterWhere(filter)
	if len(clauses) > 0 {
		q += " WHERE " + strings.Join(clauses, " AND ")
	}
	q += " ORDER BY year DESC, quarter DESC, id ASC"

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("查询季度目标失败: %w", err)
	}
	defer rows.Close()

	var out []model.QuarterGoal
	for rows.Next() {
		q, err := scanQuarterGoal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *q)
	}
	return out, rows.Err()
}

// ListQuarterGoalsWithProgress 查询并附带进度。
func (s *Store) ListQuarterGoalsWithProgress(filter QuarterGoalFilter) ([]QuarterGoalWithProgress, error) {
	goals, err := s.ListQuarterGoals(filter)
	if err != nil {
		return nil, err
	}
	out := make([]QuarterGoalWithProgress, 0, len(goals))
	for _, g := range goals {
		entry := QuarterGoalWithProgress{QuarterGoal: g}
		total, done, err := s.countWeekGoalsForQuarter(g.ID)
		if err != nil {
			return nil, err
		}
		entry.WeekTotal = total
		entry.WeekDone = done
		entry.HasProgress = total > 0
		out = append(out, entry)
	}
	return out, nil
}

// countWeekGoalsForQuarter 返回 (周目标总数, 已完成数)。
func (s *Store) countWeekGoalsForQuarter(qgID int64) (int, int, error) {
	var total, done int
	if err := s.db.QueryRow(
		"SELECT COUNT(*) FROM week_goals WHERE quarter_goal_id = ?", qgID,
	).Scan(&total); err != nil {
		return 0, 0, err
	}
	if err := s.db.QueryRow(
		"SELECT COUNT(*) FROM week_goals WHERE quarter_goal_id = ? AND status = 'completed'",
		qgID,
	).Scan(&done); err != nil {
		return 0, 0, err
	}
	return total, done, nil
}

// UpdateQuarterGoal 更新季度目标。
func (s *Store) UpdateQuarterGoal(id int64, in UpdateQuarterGoalInput) (*model.QuarterGoal, error) {
	existing, err := s.GetQuarterGoal(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("季度目标 %d 不存在", id)
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
	if in.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *in.Description)
		changed = true
	}
	if in.ClearLink {
		sets = append(sets, "year_goal_id = NULL")
		changed = true
	} else if in.YearGoalID != nil {
		sets = append(sets, "year_goal_id = ?")
		args = append(args, *in.YearGoalID)
		changed = true
	}
	if !changed {
		return existing, nil
	}
	sets = append(sets, "updated_at = datetime('now')")
	args = append(args, id)
	if _, err := s.db.Exec("UPDATE quarter_goals SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
		return nil, fmt.Errorf("更新季度目标失败: %w", err)
	}
	return s.GetQuarterGoal(id)
}

// SetQuarterGoalStatus 设置季度目标状态。
func (s *Store) SetQuarterGoalStatus(id int64, status string) (*model.QuarterGoal, error) {
	switch status {
	case model.QuarterGoalStatusActive,
		model.QuarterGoalStatusCompleted,
		model.QuarterGoalStatusArchived:
	default:
		return nil, fmt.Errorf("无效状态: %s", status)
	}
	existing, err := s.GetQuarterGoal(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("季度目标 %d 不存在", id)
	}
	if _, err := s.db.Exec(
		"UPDATE quarter_goals SET status = ?, updated_at = datetime('now') WHERE id = ?",
		status, id,
	); err != nil {
		return nil, fmt.Errorf("更新状态失败: %w", err)
	}
	return s.GetQuarterGoal(id)
}

// DeleteQuarterGoal 删除季度目标。
func (s *Store) DeleteQuarterGoal(id int64) (bool, error) {
	res, err := s.db.Exec("DELETE FROM quarter_goals WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("删除季度目标失败: %w", err)
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// LinkQuarterGoal 关联季度目标到年度目标。
func (s *Store) LinkQuarterGoal(qID, yID int64) (*model.QuarterGoal, error) {
	existing, err := s.GetQuarterGoal(qID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("季度目标 %d 不存在", qID)
	}
	var exists int
	if err := s.db.QueryRow(
		"SELECT 1 FROM year_goals WHERE id = ?", yID,
	).Scan(&exists); err == sql.ErrNoRows {
		return nil, fmt.Errorf("年度目标 %d 不存在", yID)
	} else if err != nil {
		return nil, err
	}
	return s.UpdateQuarterGoal(qID, UpdateQuarterGoalInput{YearGoalID: &yID})
}

// ---------- 内部辅助 ----------

func buildQuarterWhere(f QuarterGoalFilter) ([]string, []any) {
	var clauses []string
	var args []any
	if f.Year > 0 {
		clauses = append(clauses, "year = ?")
		args = append(args, f.Year)
	}
	if f.Quarter > 0 {
		clauses = append(clauses, "quarter = ?")
		args = append(args, f.Quarter)
	}
	if f.YearGoalID != nil {
		clauses = append(clauses, "year_goal_id = ?")
		args = append(args, *f.YearGoalID)
	}
	if f.Status != "" && f.Status != "all" {
		clauses = append(clauses, "status = ?")
		args = append(args, f.Status)
	}
	return clauses, args
}

func scanQuarterGoal(s scanner) (*model.QuarterGoal, error) {
	var (
		q          model.QuarterGoal
		yearGoalID sql.NullInt64
		createdAt  sql.NullTime
		updatedAt  sql.NullTime
	)
	err := s.Scan(&q.ID, &q.Title, &q.Year, &q.Quarter, &q.Description,
		&yearGoalID, &q.Status, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if yearGoalID.Valid {
		id := yearGoalID.Int64
		q.YearGoalID = &id
	}
	if createdAt.Valid {
		q.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		q.UpdatedAt = updatedAt.Time
	}
	return &q, nil
}
