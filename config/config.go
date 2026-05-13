package config

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	pkglog "todo/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type TagGroup struct {
	Name string   `yaml:"name"`
	Tags []string `yaml:"tags"`
}

type Config struct {
	Port           int        `yaml:"port"`
	Token          string     `yaml:"token,omitempty"`      // kept for one-time migration to TokenHash
	TokenHash      string     `yaml:"token_hash,omitempty"` // SHA-256 hex of the bearer token
	TagGroups      []TagGroup `yaml:"tag_groups"`
	WorkerInterval int        `yaml:"worker_interval"` // seconds, default 30
	mu             sync.RWMutex
}

var cfg *Config

func todoHome() string {
	if h := os.Getenv("TODO_HOME"); h != "" {
		return h
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".todo")
	}
	return filepath.Join(home, ".todo")
}

func (c *Config) DBPath() string {
	return filepath.Join(todoHome(), "todo.db")
}

func Load() (*Config, error) {
	logger := pkglog.GetLogger(nil)
	dir := todoHome()
	if err := os.MkdirAll(dir, 0700); err != nil {
		logger.Error("cannot create config directory", zap.String("dir", dir), zap.Error(err))
		return nil, err
	}
	configPath := filepath.Join(dir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		c := defaultConfig()
		if err := c.Save(); err != nil {
			logger.Error("failed to save default config", zap.Error(err))
			return nil, fmt.Errorf("config save: %w", err)
		}
		logger.Info("default config created", zap.String("path", configPath))
		cfg = c
		return c, nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.Error("failed to read config file", zap.String("path", configPath), zap.Error(err))
		return nil, err
	}
	c := &Config{}
	if err := yaml.Unmarshal(data, c); err != nil {
		logger.Error("failed to parse config", zap.Error(err))
		return nil, fmt.Errorf("config parse: %w", err)
	}
	if c.Port == 0 {
		c.Port = 8080
	}
	if c.WorkerInterval <= 0 {
		c.WorkerInterval = 30
	}
	if c.TokenHash == "" && c.Token != "" {
		logger.Info("migrating plaintext token to hash")
		hash := sha256.Sum256([]byte(c.Token))
		c.TokenHash = hex.EncodeToString(hash[:])
		c.Token = ""
		if err := c.Save(); err != nil {
			logger.Error("failed to persist token hash migration", zap.Error(err))
		}
	}
	cfg = c
	return c, nil
}

func Get() *Config {
	return cfg
}

func (c *Config) HasToken() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.TokenHash != ""
}

func (c *Config) VerifyToken(token string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.TokenHash == "" {
		return true
	}
	hash := sha256.Sum256([]byte(token))
	expected, err := hex.DecodeString(c.TokenHash)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(hash[:], expected) == 1
}

func (c *Config) SetToken(token string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	hash := sha256.Sum256([]byte(token))
	c.TokenHash = hex.EncodeToString(hash[:])
	c.Token = ""
	return c.Save()
}

func (c *Config) Save() error {
	logger := pkglog.GetLogger(nil)
	configPath := filepath.Join(todoHome(), "config.yaml")
	data, err := yaml.Marshal(c)
	if err != nil {
		logger.Error("failed to marshal config", zap.Error(err))
		return err
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		logger.Error("failed to write config file", zap.String("path", configPath), zap.Error(err))
		return err
	}
	return nil
}

func defaultConfig() *Config {
	return &Config{
		Port:           8080,
		TagGroups:      []TagGroup{},
		WorkerInterval: 30,
	}
}
