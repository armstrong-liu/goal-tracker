package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// TestOpenCreatesDatabaseAndTables 验证：
// 1. Open 能自动创建数据库目录和文件
// 2. 所有表都被创建
// 3. 索引被创建
func TestOpenCreatesDatabaseAndTables(t *testing.T) {
	// 用临时目录，测试结束自动清理
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "data.db") // 故意用嵌套目录，验证自动创建

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer s.Close()

	// 验证数据库文件被创建
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("数据库文件未被创建")
	}

	// 验证所有表存在
	tables := []string{"year_goals", "quarter_goals", "week_goals", "tasks"}
	for _, table := range tables {
		var name string
		err := s.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&name)
		if err == sql.ErrNoRows {
			t.Errorf("表 %q 未被创建", table)
		} else if err != nil {
			t.Fatalf("查询表 %q 失败: %v", table, err)
		}
	}

	// 验证索引存在
	indexes := []string{
		"idx_tasks_due_date",
		"idx_tasks_status",
		"idx_tasks_week_goal_id",
		"idx_week_goals_year_week",
		"idx_quarter_goals_year_quarter",
	}
	for _, idx := range indexes {
		var name string
		err := s.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='index' AND name=?",
			idx,
		).Scan(&name)
		if err == sql.ErrNoRows {
			t.Errorf("索引 %q 未被创建", idx)
		} else if err != nil {
			t.Fatalf("查询索引 %q 失败: %v", idx, err)
		}
	}
}

// TestInsertIntoAllTables 验证每张表都能正常插入数据（冒烟测试）
func TestInsertIntoAllTables(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "data.db")

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer s.Close()

	// 插入年度目标
	yearRes, err := s.db.Exec(
		`INSERT INTO year_goals (title, year) VALUES (?, ?)`,
		"年度目标测试", 2026,
	)
	if err != nil {
		t.Fatalf("插入 year_goals 失败: %v", err)
	}
	yearID, _ := yearRes.LastInsertId()

	// 插入季度目标（关联年度目标）
	qRes, err := s.db.Exec(
		`INSERT INTO quarter_goals (title, year, quarter, year_goal_id) VALUES (?, ?, ?, ?)`,
		"Q3目标", 2026, 3, yearID,
	)
	if err != nil {
		t.Fatalf("插入 quarter_goals 失败: %v", err)
	}
	quarterID, _ := qRes.LastInsertId()

	// 插入周目标（关联季度目标）
	wRes, err := s.db.Exec(
		`INSERT INTO week_goals (title, year, week, quarter_goal_id) VALUES (?, ?, ?, ?)`,
		"W32目标", 2026, 32, quarterID,
	)
	if err != nil {
		t.Fatalf("插入 week_goals 失败: %v", err)
	}
	weekID, _ := wRes.LastInsertId()

	// 插入任务（关联周目标）
	_, err = s.db.Exec(
		`INSERT INTO tasks (title, priority, week_goal_id) VALUES (?, ?, ?)`,
		"测试任务", "high", weekID,
	)
	if err != nil {
		t.Fatalf("插入 tasks 失败: %v", err)
	}

	// 验证数据能查出来
	var taskCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&taskCount)
	if err != nil {
		t.Fatalf("查询 tasks 数量失败: %v", err)
	}
	if taskCount != 1 {
		t.Errorf("tasks 数量 = %d, 期望 1", taskCount)
	}
}

// TestForeignKeyConstraint 验证外键约束开启
func TestForeignKeyConstraint(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "data.db")

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer s.Close()

	var fkEnabled int
	err = s.db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("查询 foreign_keys 失败: %v", err)
	}
	if fkEnabled != 1 {
		t.Error("外键约束未开启（期望 PRAGMA foreign_keys = 1）")
	}
}
