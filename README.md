# 🎯 Goal Tracker

> 一个终端原生的**个人目标与任务管理工具**，帮助你管理从"日常 TODO"到"年度目标"的完整目标层级。
>
> **核心理念：每一个日常 TODO，都能向上追溯到它服务的年度目标。**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

## ✨ 特性

- 🖥️ **终端原生**：TUI 交互界面 + CLI 命令行，双模式
- 🎯 **四级目标层级**：年度目标 → 季度目标 → 周目标 → 任务
- 📊 **进度自动传导**：任务完成 → 周目标进度更新 → 季度/年度进度联动
- 📅 **智能日期**：支持 `today`/`tomorrow`/`monday` 等关键词
- 📤 **Markdown 导出**：一键生成周报/季度回顾
- 💾 **单文件部署**：纯 Go 实现，编译后零依赖
- 🔒 **数据本地化**：所有数据存在 `~/.goaltracker/data.db`

---

## 📦 安装

### 方式一：下载预编译版本

从 [Releases](../../releases) 页面下载对应平台的二进制文件，重命名为 `gt`（或 `gt.exe`），放入 `PATH` 即可。

### 方式二：从源码编译

```bash
git clone <repo-url>
cd goal-tracker
go build -o gt .
```

要求：[Go](https://golang.org/dl/) 1.21+。

---

## 🚀 快速开始

### 1. 启动 TUI 界面

```bash
gt
```

进入交互式终端界面，用 `1234` 或 `Tab` 切换视图，`j/k` 移动光标，`Space` 完成任务，`a` 添加，`q` 退出。

### 2. 用 CLI 建立目标体系

```bash
# 设定年度方向
gt year add "晋升P7" -d "本年度核心目标"

# 拆解到季度
gt quarter add "项目A上线" -g 1

# 拆解到本周
gt week add "完成技术方案" -q 1

# 落实到任务
gt task add "写技术文档" -w 1 --due today -p high

# 执行
gt task done 1
```

### 3. 查看进度

```bash
gt today          # 今日概览
gt week view      # 本周目标 + 进度
gt quarter view   # 季度目标 + 进度
gt year view      # 年度目标 + 进度
```

---

## 📖 命令参考

### 全局

| 命令 | 说明 |
|------|------|
| `gt` | 启动 TUI 交互界面 |
| `gt today` | 今日概览（过期 + 今日到期 + 统计）|
| `gt review` | 周回顾 |
| `gt export` | 导出 Markdown |
| `gt --version` | 显示版本 |
| `gt --help` | 帮助 |

### 任务管理（`gt task`）

```bash
gt task add <标题> [flags]      # 添加任务
gt task list [flags]            # 查看任务列表
gt task done <id>               # 标记完成
gt task undone <id>             # 恢复为待办
gt task edit <id> [flags]       # 编辑任务
gt task delete <id> [-f]        # 删除任务
gt task link <task_id> <week_goal_id>  # 关联到周目标
```

**`task add` / `task list` 常用 flags：**

| Flag | 缩写 | 说明 | 取值 |
|------|------|------|------|
| `--due` | `-d` | 截止日期 | `today` / `tomorrow` / `monday`..`sunday` / `YYYY-MM-DD` |
| `--priority` | `-p` | 优先级 | `high` / `medium` / `low`（默认 medium）|
| `--week` | `-w` | 关联周目标 ID | 数字 |
| `--status` | `-s` | 状态过滤（list） | `pending` / `done` / `all` |

### 周目标（`gt week`）

```bash
gt week add <标题> [-y 年] [-w 周] [-q 季度目标ID]
gt week view [-y 年] [-w 周]
gt week done <id>
gt week edit <id> [-t 标题]
gt week delete <id> [-f]
```

### 季度目标（`gt quarter`）

```bash
gt quarter add <标题> [-y 年] [-q 季度] [-g 年度目标ID] [-d 描述]
gt quarter view [-y 年] [-q 季度]
gt quarter done <id>
gt quarter edit <id> [flags]
gt quarter delete <id> [-f]
```

> `quarter view` 默认显示**全年所有季度目标**（按季度分组、当前季度标注 `← 当前`）；加 `-q` 只看指定季度。

### 年度目标（`gt year`）

```bash
gt year add <标题> [-y 年] [-d 描述]
gt year view [-y 年]
gt year done <id>
gt year edit <id> [-t 标题] [-d 描述]
gt year delete <id> [-f]
```

### 导出（`gt export`）

```bash
gt export [--scope 范围] [-o 文件]
```

| scope | 导出内容 |
|-------|---------|
| `today` | 今日到期 + 过期任务 + 统计 |
| `week` | 本周目标及其任务（默认）|
| `quarter` | 本季度目标及其下周目标 |
| `year` | 本年度目标及其下季度目标 |
| `all` | 完整层级（年→季→周→任务）|

示例：

```bash
gt export                          # 本周回顾输出到屏幕
gt export -o weekly.md             # 导出到文件
gt export --scope all -o full.md   # 完整导出
```

---

## 🎨 TUI 快捷键

| 按键 | 作用 |
|------|------|
| `1` `2` `3` `4` | 切换主视图（今日/周/季/年）|
| `Tab` / `Shift+Tab` | 切换下一个/上一个视图 |
| `↑` `↓` 或 `j` `k` | 上下移动选中项 |
| `←` `→` 或 `h` `l` | 周/季视图切换上一/下一周（季度），当前期带标注 |
| `Space` | 切换完成状态（**任务和周/季/年目标都支持**）|
| `Enter` | 查看详情面板（**完整标题不截断**，含上级目标信息）|
| `a` | 添加新任务（今日视图）|
| `x` | 删除当前项（带确认）|
| `?` | 显示快捷键提示 |
| `q` 或 `Ctrl+C` | 退出（详情面板打开时按任意键仅关闭面板）|

---

## 🏗️ 进度计算规则

所有进度都是**动态计算**的，不存储：

| 层级 | 进度公式 |
|------|---------|
| 周目标 | 关联任务中已完成数 / 关联任务总数 |
| 季度目标 | 关联周目标中已完成数 / 关联周目标总数 |
| 年度目标 | 关联季度目标中已完成数 / 关联季度目标总数 |

无关联子项时显示 `—`（不计算）。

**完整传导链**：完成 4 个任务 → 周目标进度 100% → 标记周目标完成 → 季度进度更新 → 标记季度完成 → 年度进度更新。

---

## 📁 数据存储

- **位置**：`~/.goaltracker/data.db`（SQLite）
- **自定义**：`gt --db /path/to/db.db <command>`
- **备份**：直接复制 `.db` 文件即可

---

## 🛠️ 技术栈

| 组件 | 选型 |
|------|------|
| 语言 | Go 1.21+ |
| CLI 框架 | [cobra](https://github.com/spf13/cobra) |
| TUI 框架 | [bubbletea](https://github.com/charmbracelet/bubbletea) |
| 样式 | [lipgloss](https://github.com/charmbracelet/lipgloss) |
| 数据库 | [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)（纯 Go，无需 CGO）|

---

## 📂 项目结构

```
goal-tracker/
├── main.go                      # 入口
├── internal/
│   ├── cmd/                     # CLI 命令（cobra）
│   │   ├── root.go              # 根命令 + TUI 启动
│   │   ├── task.go              # gt task
│   │   ├── week.go              # gt week
│   │   ├── quarter.go           # gt quarter
│   │   ├── year.go              # gt year
│   │   ├── today.go             # gt today
│   │   ├── review.go            # gt review
│   │   ├── export.go            # gt export
│   │   └── ui.go                # CLI 输出美化
│   ├── tui/                     # TUI 界面（bubbletea）
│   │   ├── model.go             # 主 Model（Elm 架构）
│   │   ├── views.go             # 四个视图渲染
│   │   ├── table.go             # 通用表格渲染
│   │   ├── styles.go            # 样式定义
│   │   └── helpers.go           # 辅助函数
│   ├── store/                   # 数据访问层（唯一执行 SQL 的地方）
│   │   ├── db.go                # SQLite 连接 + 建表
│   │   ├── task.go              # Task CRUD
│   │   ├── week_goal.go         # WeekGoal CRUD + 进度
│   │   ├── quarter_goal.go      # QuarterGoal CRUD + 进度
│   │   └── year_goal.go         # YearGoal CRUD + 进度
│   ├── model/                   # 领域模型（纯数据结构）
│   └── util/                    # 工具函数（日期、配置）
└── migrations/
    └── 001_init.sql             # 建表脚本
```

---

## 📝 许可证

MIT
