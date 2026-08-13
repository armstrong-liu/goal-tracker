package util

import (
	"os"
	"path/filepath"
)

// AppName 应用名，用于构造配置目录
const AppName = "goaltracker"

// DefaultDBFileName 默认数据库文件名
const DefaultDBFileName = "data.db"

// Config 全局配置
type Config struct {
	// DBPath 数据库文件路径
	DBPath string
}

// DefaultConfig 返回默认配置：数据库放在 ~/.goaltracker/data.db
func DefaultConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, "."+AppName)
	return &Config{
		DBPath: filepath.Join(dir, DefaultDBFileName),
	}, nil
}

// EnsureDBDir 确保数据库所在目录存在（不存在则创建）。
// 返回数据库文件路径。
func (c *Config) EnsureDBDir() error {
	dir := filepath.Dir(c.DBPath)
	return os.MkdirAll(dir, 0o755)
}
