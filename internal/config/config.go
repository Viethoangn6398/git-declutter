package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	AppName          = "git-declutter"
	DefaultRetention = 30 * 24 * time.Hour
	CacheTTL         = 5 * time.Minute
)

type Config struct {
	Protected []string        `yaml:"protected"`
	Recovery  RecoveryConfig  `yaml:"recovery"`
	Providers ProvidersConfig `yaml:"providers"`
	Cleanup   CleanupConfig   `yaml:"cleanup"`
}

type RecoveryConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Retention string `yaml:"retention"`
}

type ProvidersConfig struct {
	GitHub ProviderToggle `yaml:"github"`
	GitLab ProviderToggle `yaml:"gitlab"`
}

type ProviderToggle struct {
	Enabled bool `yaml:"enabled"`
}

type CleanupConfig struct {
	RequireRemoteDeleted bool `yaml:"requireRemoteDeleted"`
}

func Defaults() Config {
	return Config{
		Protected: []string{
			"main",
			"master",
			"develop",
			"development",
			"release/*",
			"production",
			"prod",
		},
		Recovery: RecoveryConfig{
			Enabled:   true,
			Retention: "30d",
		},
		Providers: ProvidersConfig{
			GitHub: ProviderToggle{Enabled: true},
			GitLab: ProviderToggle{Enabled: true},
		},
		Cleanup: CleanupConfig{
			RequireRemoteDeleted: true,
		},
	}
}

func (c Config) RetentionDuration() (time.Duration, bool, error) {
	raw := strings.TrimSpace(strings.ToLower(c.Recovery.Retention))
	if raw == "" {
		return DefaultRetention, false, nil
	}
	if raw == "forever" || raw == "never" {
		return 0, true, nil
	}
	d, err := ParseDuration(raw)
	if err != nil {
		return 0, false, err
	}
	return d, false, nil
}

func ParseDuration(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return 0, errors.New("empty duration")
	}
	if strings.HasSuffix(raw, "d") {
		n := strings.TrimSuffix(raw, "d")
		var days int
		if _, err := fmt.Sscanf(n, "%d", &days); err != nil {
			return 0, fmt.Errorf("invalid duration %q", raw)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(raw)
}

func GlobalPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, AppName, "config.yaml"), nil
}

func Load() (Config, error) {
	cfg := Defaults()
	path, err := GlobalPath()
	if err != nil {
		return cfg, nil
	}
	if err := mergeFile(path, &cfg); err != nil && !os.IsNotExist(err) {
		return cfg, err
	}
	return cfg, nil
}

func LoadWithRepo(repoRoot string) (Config, error) {
	cfg, err := Load()
	if err != nil {
		return cfg, err
	}
	if repoRoot == "" {
		return cfg, nil
	}
	for _, name := range []string{".gitdeclutter.yml", ".gitdeclutter.yaml"} {
		path := filepath.Join(repoRoot, name)
		if err := mergeFile(path, &cfg); err != nil && !os.IsNotExist(err) {
			return cfg, err
		}
	}
	return cfg, nil
}

func mergeFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}

func Save(cfg Config) error {
	path, err := GlobalPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func MatchesProtected(patterns []string, branch string) (bool, string) {
	for _, p := range patterns {
		if matchPattern(p, branch) {
			return true, p
		}
	}
	return false, ""
}

func matchPattern(pattern, name string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	if pattern == name {
		return true
	}
	ok, err := filepath.Match(pattern, name)
	return err == nil && ok
}
