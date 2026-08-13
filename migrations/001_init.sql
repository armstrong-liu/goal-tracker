-- 001_init.sql
-- Goal Tracker 初始建表脚本
-- 严格对应设计文档第 4 章数据模型设计

-- 表1：年度目标
CREATE TABLE IF NOT EXISTS year_goals (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT NOT NULL,
    year        INTEGER NOT NULL,
    description TEXT DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'active',
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- 表2：季度目标
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

-- 表3：周目标
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

-- 表4：TODO 任务
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

-- 索引（性能优化）
CREATE INDEX IF NOT EXISTS idx_tasks_due_date     ON tasks(due_date);
CREATE INDEX IF NOT EXISTS idx_tasks_status        ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_week_goal_id  ON tasks(week_goal_id);
CREATE INDEX IF NOT EXISTS idx_week_goals_year_week        ON week_goals(year, week);
CREATE INDEX IF NOT EXISTS idx_quarter_goals_year_quarter  ON quarter_goals(year, quarter);
