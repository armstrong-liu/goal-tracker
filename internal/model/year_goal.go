package model

import "time"

// YearGoal 表示一个年度目标，对应数据库表 year_goals。
type YearGoal struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Year        int       `json:"year"`        // 如 2026
	Description string    `json:"description"`
	Status      string    `json:"status"`      // active / completed / archived
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// 年度目标状态常量
const (
	YearGoalStatusActive    = "active"
	YearGoalStatusCompleted = "completed"
	YearGoalStatusArchived  = "archived"
)

// IsValidStatus 判断年度目标状态是否合法
func (y YearGoal) IsValidStatus() bool {
	switch y.Status {
	case YearGoalStatusActive, YearGoalStatusCompleted, YearGoalStatusArchived:
		return true
	default:
		return false
	}
}
