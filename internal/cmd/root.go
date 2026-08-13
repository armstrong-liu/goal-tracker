// Package cmd 定义所有 CLI 命令（基于 cobra）。
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"goal-tracker/internal/store"
	"goal-tracker/internal/tui"
	"goal-tracker/internal/util"
)

// Version 当前版本号。后续可通过 ldflags 注入。
const Version = "v0.1.0"

// globalDBPath 全局 --db flag 的值
var globalDBPath string

// rootCmd 根命令：gt
var rootCmd = &cobra.Command{
	Use:   "gt",
	Short: "🎯 Goal Tracker — 个人目标与任务管理工具",
	Long: `Goal Tracker (gt) 是一个终端原生的目标与任务管理工具，
帮助你管理从"日常 TODO"到"年度目标"的完整目标层级。

核心理念：每一个日常 TODO，都能向上追溯到它服务的年度目标。

运行 "gt"（无参数）启动 TUI 主界面。`,
	// 无参数运行时：启动 TUI 界面
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, err := ResolveDBPath()
		if err != nil {
			return err
		}
		s, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer s.Close()

		// 启动 Bubbletea TUI
		if err := tui.Run(s); err != nil {
			return fmt.Errorf("启动 TUI 失败: %w", err)
		}
		return nil
	},
}

// Execute 执行根命令。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// cobra 已经打印了错误信息，这里直接退出
		os.Exit(1)
	}
}

// GetRootCmd 返回根命令（用于测试或外部调用）。
func GetRootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	// 全局 flag：--db 指定数据库路径
	rootCmd.PersistentFlags().StringVar(
		&globalDBPath,
		"db",
		"",
		"数据库文件路径（默认 ~/.goaltracker/data.db）",
	)

	// --version / -v
	rootCmd.Flags().BoolP("version", "v", false, "显示版本号")
	rootCmd.Run = nil // 用 RunE，禁用默认 Run
	// 注：cobra 的 version 行为可以通过 SetVersionTemplate 自定义
	rootCmd.SetVersionTemplate(fmt.Sprintf("Goal Tracker %s\n", Version))
	rootCmd.Version = Version
}

// ResolveDBPath 解析数据库路径。
// 优先级：--db flag > 默认 ~/.goaltracker/data.db
func ResolveDBPath() (string, error) {
	if globalDBPath != "" {
		return globalDBPath, nil
	}
	cfg, err := util.DefaultConfig()
	if err != nil {
		return "", fmt.Errorf("获取默认配置失败: %w", err)
	}
	return cfg.DBPath, nil
}
