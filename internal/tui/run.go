package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"goal-tracker/internal/store"
)

// Run 启动 TUI 程序。
func Run(s *store.Store) error {
	m := NewModel(s)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI 运行错误: %w", err)
	}
	return nil
}
