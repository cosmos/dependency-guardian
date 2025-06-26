package config

// Config represents the root configuration structure
type Config struct {
	Targets    TargetConfig    `yaml:"targets"`
	Patterns   PatternConfig   `yaml:"patterns"`
	Analysis   AnalysisConfig  `yaml:"analysis"`
	Critical   CriticalConfig  `yaml:"critical"`
}

// TargetConfig defines which high-level packages to analyze
type TargetConfig struct {
	HighLevelPackages []string `yaml:"high_level_packages"`
}

// PatternConfig defines include/exclude patterns for analysis
type PatternConfig struct {
	IgnorePatterns  []string `yaml:"ignore_patterns"`
	IncludePatterns []string `yaml:"include_patterns"`
}

// AnalysisConfig defines analysis behavior settings
type AnalysisConfig struct {
	MaxDepth           int `yaml:"max_depth"`
	MinImpactThreshold int `yaml:"min_impact_threshold"`
}

// CriticalConfig defines critical packages that require special attention
type CriticalConfig struct {
	Packages []string `yaml:"packages"`
} 