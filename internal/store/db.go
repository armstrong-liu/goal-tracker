// Package store 提供数据访问层，是整个应用中唯一执行 SQL 的地方。
// cmd 层和 tui 层必须通过 store 提供的方法访问数据。
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // 纯 Go 实现的 SQLite 驱动，无需 CGO
)

// initSQL 内嵌的初始建表脚本，对应 migrations/001_init.sql。
// 内嵌是为了编译后单二进制即可运行，不依赖外部 SQL 文件。
const initSQL = `
CREATE TABLE IF NOT EXISTS year_goals (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT NOT NULL,
    year        INTEGER NOT NULL,
    description TEXT DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'active',
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS quarter_goals (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    title        TEXT NOT NULL,
    year         INTEGER NOT NULL,
    quarter      INTEGER NOT NULL CHECK(quarter IN (1,2,3,4)),
    description  TEXT DEFAULT '',
    year_goal_id INTEGER,
    status       TEXT NOT NULL DEFAULT 'active',
    created_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (year_goal_id) REFERENCES year_goals(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS week_goals (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    title            TEXT NOT NULL,
    year             INTEGER NOT NULL,
    week             INTEGER NOT NULL CHECK(week BETWEEN 1 AND 53),
    quarter_goal_id  INTEGER,
    status           TEXT NOT NULL DEFAULT 'active',
    created_at       DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at       DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (quarter_goal_id) REFERENCES quarter_goals(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS tasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    title        TEXT NOT NULL,
    due_date     DATE,
    priority     TEXT NOT NULL DEFAULT 'medium',
    week_goal_id INTEGER,
    status       TEXT NOT NULL DEFAULT 'pending',
    created_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (week_goal_id) REFERENCES week_goals(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_due_date     ON tasks(due_date);
CREATE INDEX IF NOT EXISTS idx_tasks_status        ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_week_goal_id  ON tasks(week_goal_id);
CREATE INDEX IF NOT EXISTS idx_week_goals_year_week        ON week_goals(year, week);
CREATE INDEX IF NOT EXISTS idx_quarter_goals_year_quarter  ON quarter_goals(year, quarter);
`

// Store 封装数据库连接，提供各实体的数据访问方法。
type Store struct {
	db *sql.DB
}

// Open 打开（或创建）指定路径的 SQLite 数据库，并执行初始建表。
// 如果父目录不存在会自动创建。
func Open(dbPath string) (*Store, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	// modernc.org/sqlite 注册的驱动名为 "sqlite"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 测试连接是否有效
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// SQLite 推荐配置：开启外键约束
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("开启外键约束失败: %w", err)
	}

	// 执行初始建表
	if _, err := db.Exec(initSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化数据库表失败: %w", err)
	}

	return &Store{db: db}, nil
}

// Close 关闭数据库连接。
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// DB 暴露底层 *sql.DB，供 store 内部各文件使用（不对外暴露给 cmd/tui）。
func (s *Store) DB() *sql.DB {
	return s.db
}
