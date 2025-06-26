package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// DefaultConfigName is the default name of the config file
const DefaultConfigName = ".dependency-guardian.yml"

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Targets: TargetConfig{
			// By default, consider all packages as high-level
			HighLevelPackages: []string{
				"**",
			},
		},
		Patterns: PatternConfig{
			// Only ignore test files by default
			IgnorePatterns: []string{
				"*_test.go",
			},
			IncludePatterns: []string{},
		},
		Analysis: AnalysisConfig{
			MaxDepth:           10,  // Increased depth
			MinImpactThreshold: 0,   // Show all impacts
		},
		Critical: CriticalConfig{
			Packages: []string{},
		},
	}
}

// LoadConfig loads the configuration.
// If a specific configFilePath is provided, it is used.
// If configFilePath is empty, it looks for the default config file in repoPath.
func LoadConfig(repoPath, configFilePath string) (*Config, error) {
	config := DefaultConfig()

	var loadPath string
	explicitPathProvided := configFilePath != ""

	if explicitPathProvided {
		loadPath = configFilePath
	} else {
		loadPath = filepath.Join(repoPath, DefaultConfigName)
	}

	data, err := os.ReadFile(loadPath)
	if err != nil {
		if os.IsNotExist(err) {
			if explicitPathProvided {
				// User specified a file that doesn't exist. This is an error.
				return nil, fmt.Errorf("config file not found at specified path: %s", loadPath)
			}
			// Default file doesn't exist. This is fine, use defaults.
			zap.S().Infow("no default config file found, using default configuration", "path", loadPath)
			return config, nil
		}
		// Some other file reading error.
		return nil, fmt.Errorf("failed to read config file %s: %w", loadPath, err)
	}

	// Parse config file
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", loadPath, err)
	}

	return config, nil
}

// IsHighLevelPackage checks if a package matches any of the high-level package patterns
func (c *Config) IsHighLevelPackage(pkgPath string) bool {
	// If no high-level packages are defined, consider everything a target.
	if len(c.Targets.HighLevelPackages) == 0 {
		return true
	}

	for _, pattern := range c.Targets.HighLevelPackages {
		if matched, _ := doublestar.Match(pattern, pkgPath); matched {
			return true
		}
	}
	return false
}

// IsCriticalPackage checks if a package matches any of the critical package patterns
func (c *Config) IsCriticalPackage(pkgPath string) bool {
	for _, pattern := range c.Critical.Packages {
		if matched, _ := doublestar.Match(pattern, pkgPath); matched {
			return true
		}
	}
	return false
}

// ShouldIgnorePackage checks if a package should be ignored based on ignore patterns
func (c *Config) ShouldIgnorePackage(pkgPath string) bool {
	// Only ignore test files and explicitly ignored patterns
	for _, pattern := range c.Patterns.IgnorePatterns {
		if matched, _ := doublestar.Match(pattern, pkgPath); matched {
			return true
		}
	}
	return false
} 