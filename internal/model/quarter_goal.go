package model

import "time"

// QuarterGoal 表示一个季度目标，对应数据库表 quarter_goals。
type QuarterGoal struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Year       int       `json:"year"`        // 如 2026
	Quarter    int       `json:"quarter"`     // 1-4
	Description string   `json:"description"`
	YearGoalID *int64    `json:"year_goal_id"` // 关联年度目标，可为空
	Status     string    `json:"status"`      // active / completed / archived
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// 季度目标状态常量
const (
	QuarterGoalStatusActive    = "active"
	QuarterGoalStatusCompleted = "completed"
	QuarterGoalStatusArchived  = "archived"
)

// IsValidStatus 判断季度目标状态是否合法
func (q QuarterGoal) IsValidStatus() bool {
	switch q.Status {
	case QuarterGoalStatusActive, QuarterGoalStatusCompleted, QuarterGoalStatusArchived:
		return true
	default:
		return false
	}
}
