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
	Redis    RedisConfig  `yaml:"redis"`
	OAuth2   OAuth2Config `yaml:"oauth2"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port string `yaml:"port"`
}

// DBConfig holds database settings.
type DBConfig struct {
	Path string `yaml:"path"`
}

// RedisConfig holds Redis connection settings for the token store.
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// OAuth2Config holds OAuth2 settings.
type OAuth2Config struct {
	AccessTokenExp  string         `yaml:"access_token_exp"`
	RefreshTokenExp string         `yaml:"refresh_token_exp"`
	Clients         []ClientConfig `yaml:"clients"`
}

// ClientConfig represents a registered OAuth2 client.
type ClientConfig struct {
	ID     string `yaml:"id"`
	Secret string `yaml:"secret"`
	Domain string `yaml:"domain"`
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

// GetAccessTokenDuration parses the access token expiry string into a time.Duration.
func (c *OAuth2Config) GetAccessTokenDuration() (time.Duration, error) {
	return time.ParseDuration(c.AccessTokenExp)
}

// GetRefreshTokenDuration parses the refresh token expiry string into a time.Duration.
func (c *OAuth2Config) GetRefreshTokenDuration() (time.Duration, error) {
	return time.ParseDuration(c.RefreshTokenExp)
}

func (c *Config) validate() error {
	if c.Server.Port == "" {
		c.Server.Port = "8080"
	}
	if c.Database.Path == "" {
		c.Database.Path = "gatheryourdeals.db"
	}
	if c.Redis.Addr == "" {
		c.Redis.Addr = "localhost:6379"
	}
	if c.OAuth2.AccessTokenExp == "" {
		c.OAuth2.AccessTokenExp = "1h"
	}
	if c.OAuth2.RefreshTokenExp == "" {
		c.OAuth2.RefreshTokenExp = "168h"
	}
	if len(c.OAuth2.Clients) == 0 {
		return fmt.Errorf("at least one OAuth2 client must be configured")
	}
	for i, client := range c.OAuth2.Clients {
		if client.ID == "" {
			return fmt.Errorf("client at index %d has no ID", i)
		}
	}
	return nil
}
