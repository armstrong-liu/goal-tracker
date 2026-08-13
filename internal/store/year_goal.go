package store

import (
	"database/sql"
	"fmt"
	"strings"

	"goal-tracker/internal/model"
)

// YearGoalFilter 年度目标查询过滤。
type YearGoalFilter struct {
	Year   int
	Status string // active / completed / archived / all
}

// CreateYearGoalInput 创建年度目标的输入。
type CreateYearGoalInput struct {
	Title       string
	Year        int
	Description string
}

// UpdateYearGoalInput 更新年度目标。
type UpdateYearGoalInput struct {
	Title       *string
	Description *string
}

// YearGoalWithProgress 年度目标 + 进度。
type YearGoalWithProgress struct {
	model.YearGoal
	QuarterTotal int  // 关联季度目标总数
	QuarterDone  int  // 已完成季度目标数
	HasProgress  bool
}

// Progress 返回进度百分比，无关联返回 -1。
func (y YearGoalWithProgress) Progress() int {
	if !y.HasProgress || y.QuarterTotal == 0 {
		return -1
	}
	return y.QuarterDone * 100 / y.QuarterTotal
}

// CreateYearGoal 创建年度目标。
func (s *Store) CreateYearGoal(in CreateYearGoalInput) (*model.YearGoal, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, fmt.Errorf("年度目标标题不能为空")
	}
	if in.Year < 2000 || in.Year > 2100 {
		return nil, fmt.Errorf("无效的年份: %d", in.Year)
	}
	res, err := s.db.Exec(
		`INSERT INTO year_goals (title, year, description)
		 VALUES (?, ?, ?)`,
		in.Title, in.Year, in.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("插入年度目标失败: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetYearGoal(id)
}

// GetYearGoal 按 ID 查询。不存在返回 (nil, nil)。
func (s *Store) GetYearGoal(id int64) (*model.YearGoal, error) {
	row := s.db.QueryRow(
		`SELECT id, title, year, description, status, created_at, updated_at
		 FROM year_goals WHERE id = ?`, id,
	)
	y, err := scanYearGoal(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return y, nil
}

// ListYearGoals 按过滤条件查询。
func (s *Store) ListYearGoals(filter YearGoalFilter) ([]model.YearGoal, error) {
	q := `SELECT id, title, year, description, status, created_at, updated_at
	      FROM year_goals`
	var (
		clauses []string
		args    []any
	)
	if filter.Year > 0 {
		clauses = append(clauses, "year = ?")
		args = append(args, filter.Year)
	}
	if filter.Status != "" && filter.Status != "all" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if len(clauses) > 0 {
		q += " WHERE " + strings.Join(clauses, " AND ")
	}
	q += " ORDER BY year DESC, id ASC"

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("查询年度目标失败: %w", err)
	}
	defer rows.Close()

	var out []model.YearGoal
	for rows.Next() {
		y, err := scanYearGoal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *y)
	}
	return out, rows.Err()
}

// ListYearGoalsWithProgress 查询并附带进度。
func (s *Store) ListYearGoalsWithProgress(filter YearGoalFilter) ([]YearGoalWithProgress, error) {
	goals, err := s.ListYearGoals(filter)
	if err != nil {
		return nil, err
	}
	out := make([]YearGoalWithProgress, 0, len(goals))
	for _, g := range goals {
		entry := YearGoalWithProgress{YearGoal: g}
		total, done, err := s.countQuarterGoalsForYear(g.ID)
		if err != nil {
			return nil, err
		}
		entry.QuarterTotal = total
		entry.QuarterDone = done
		entry.HasProgress = total > 0
		out = append(out, entry)
	}
	return out, nil
}

// countQuarterGoalsForYear 返回 (季度目标总数, 已完成数)。
func (s *Store) countQuarterGoalsForYear(yID int64) (int, int, error) {
	var total, done int
	if err := s.db.QueryRow(
		"SELECT COUNT(*) FROM quarter_goals WHERE year_goal_id = ?", yID,
	).Scan(&total); err != nil {
		return 0, 0, err
	}
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM quarter_goals
		 WHERE year_goal_id = ? AND status = 'completed'`,
		yID,
	).Scan(&done); err != nil {
		return 0, 0, err
	}
	return total, done, nil
}

// UpdateYearGoal 更新年度目标。
func (s *Store) UpdateYearGoal(id int64, in UpdateYearGoalInput) (*model.YearGoal, error) {
	existing, err := s.GetYearGoal(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("年度目标 %d 不存在", id)
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
	if !changed {
		return existing, nil
	}
	sets = append(sets, "updated_at = datetime('now')")
	args = append(args, id)
	if _, err := s.db.Exec("UPDATE year_goals SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
		return nil, fmt.Errorf("更新年度目标失败: %w", err)
	}
	return s.GetYearGoal(id)
}

// SetYearGoalStatus 设置年度目标状态。
func (s *Store) SetYearGoalStatus(id int64, status string) (*model.YearGoal, error) {
	switch status {
	case model.YearGoalStatusActive,
		model.YearGoalStatusCompleted,
		model.YearGoalStatusArchived:
	default:
		return nil, fmt.Errorf("无效状态: %s", status)
	}
	existing, err := s.GetYearGoal(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("年度目标 %d 不存在", id)
	}
	if _, err := s.db.Exec(
		"UPDATE year_goals SET status = ?, updated_at = datetime('now') WHERE id = ?",
		status, id,
	); err != nil {
		return nil, fmt.Errorf("更新状态失败: %w", err)
	}
	return s.GetYearGoal(id)
}

// DeleteYearGoal 删除年度目标。
func (s *Store) DeleteYearGoal(id int64) (bool, error) {
	res, err := s.db.Exec("DELETE FROM year_goals WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("删除年度目标失败: %w", err)
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ---------- 内部辅助 ----------

func scanYearGoal(s scanner) (*model.YearGoal, error) {
	var (
		y         model.YearGoal
		createdAt sql.NullTime
		updatedAt sql.NullTime
	)
	err := s.Scan(&y.ID, &y.Title, &y.Year, &y.Description,
		&y.Status, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if createdAt.Valid {
		y.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		y.UpdatedAt = updatedAt.Time
	}
	return &y, nil
}
