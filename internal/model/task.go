package model

import "time"

// Task 表示一个 TODO 任务。
// 对应数据库表 tasks。
type Task struct {
	ID         int64      `json:"id"`
	Title      string     `json:"title"`
	DueDate    *time.Time `json:"due_date"`    // 截止日期，可为空
	Priority   string     `json:"priority"`    // high / medium / low
	WeekGoalID *int64     `json:"week_goal_id"` // 关联周目标，可为空
	Status     string     `json:"status"`      // pending / done
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// 任务状态常量
const (
	TaskStatusPending = "pending"
	TaskStatusDone    = "done"
)

// 优先级常量
const (
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// IsValidStatus 判断任务状态是否合法
func (t Task) IsValidStatus() bool {
	return t.Status == TaskStatusPending || t.Status == TaskStatusDone
}

// IsValidPriority 判断优先级是否合法
func (t Task) IsValidPriority() bool {
	switch t.Priority {
	case PriorityHigh, PriorityMedium, PriorityLow:
		return true
	default:
		return false
	}
}
