package shared

import (
	"regexp"
	"time"
)

// URLPattern represents a compiled URL pattern
type URLPattern struct {
	Pattern     *regexp.Regexp
	Replacement string
}

// RuntimeConfig represents the configuration for a specific UrlPerformance CRD
// This is shared between the operator controller and the log processor
type RuntimeConfig struct {
	Key            string
	Namespace      string
	TargetName     string
	TargetKind     string
	WhitelistRegex []*regexp.Regexp
	IgnoredRegex   []*regexp.Regexp
	MergePaths     []string
	URLPatterns    []URLPattern
	CollectNTop    int
	Enabled        bool
	LastUpdated    time.Time
}

// ConfigManager interface for getting runtime configurations
// This allows the controller to provide configs to the log processor
type ConfigManager interface {
	GetConfig(key string) (*RuntimeConfig, bool)
	GetAllConfigs() []*RuntimeConfig
}
