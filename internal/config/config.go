package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the full server configuration.
type Config struct {
	Server   ServerConfig `yaml:"server"`
	Database DBConfig     `yaml:"database"`
	Auth     AuthConfig   `yaml:"auth"`
	Log      LogConfig    `yaml:"log"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port string `yaml:"port"`
}

// DBConfig holds database settings.
type DBConfig struct {
	Path string `yaml:"path"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Dir       string `yaml:"dir"`
	MaxSizeMB int    `yaml:"max_size_mb"`
}

// AuthConfig holds JWT authentication settings.
// The JWT secret is intentionally NOT stored in the YAML file.
// Set the GYD_JWT_SECRET environment variable instead.
type AuthConfig struct {
	AccessTokenExp  string `yaml:"access_token_exp"`
	RefreshTokenExp string `yaml:"refresh_token_exp"`
}

// Load reads the config from a YAML file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// JWTSecret reads the JWT signing secret from the GYD_JWT_SECRET environment variable.
// Returns an error if it is not set or is too short to be secure.
func (c *Config) JWTSecret() ([]byte, error) {
	secret := os.Getenv("GYD_JWT_SECRET")
	if len(secret) < 32 {
		return nil, fmt.Errorf("GYD_JWT_SECRET must be set and at least 32 characters long")
	}
	return []byte(secret), nil
}

// GetAccessTokenDuration parses the access token expiry string into a time.Duration.
func (c *AuthConfig) GetAccessTokenDuration() (time.Duration, error) {
	return time.ParseDuration(c.AccessTokenExp)
}

// GetRefreshTokenDuration parses the refresh token expiry string into a time.Duration.
func (c *AuthConfig) GetRefreshTokenDuration() (time.Duration, error) {
	return time.ParseDuration(c.RefreshTokenExp)
}

func (c *Config) validate() error {
	if c.Server.Port == "" {
		c.Server.Port = "8080"
	}
	if c.Database.Path == "" {
		c.Database.Path = "gatheryourdeals.db"
	}
	if c.Auth.AccessTokenExp == "" {
		c.Auth.AccessTokenExp = "1h"
	}
	if c.Auth.RefreshTokenExp == "" {
		c.Auth.RefreshTokenExp = "168h"
	}
	if c.Log.Dir == "" {
		c.Log.Dir = "logs"
	}
	if c.Log.MaxSizeMB <= 0 {
		c.Log.MaxSizeMB = 10
	}
	return nil
}
