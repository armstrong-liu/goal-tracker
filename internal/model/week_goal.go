package model

import "time"

// WeekGoal 表示一个周目标，对应数据库表 week_goals。
type WeekGoal struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	Year          int       `json:"year"`           // ISO 周年
	Week          int       `json:"week"`           // ISO 周数 1-53
	QuarterGoalID *int64    `json:"quarter_goal_id"` // 关联季度目标，可为空
	Status        string    `json:"status"`         // active / completed
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// 周目标状态常量
const (
	WeekGoalStatusActive    = "active"
	WeekGoalStatusCompleted = "completed"
)

// IsValidStatus 判断周目标状态是否合法
func (w WeekGoal) IsValidStatus() bool {
	return w.Status == WeekGoalStatusActive || w.Status == WeekGoalStatusCompleted
}
