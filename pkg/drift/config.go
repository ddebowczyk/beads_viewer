package drift

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config contains drift detection thresholds
type Config struct {
	// DensityWarningPct triggers warning when density increases by this percentage
	DensityWarningPct float64 `yaml:"density_warning_pct" json:"density_warning_pct"`

	// DensityInfoPct triggers info when density increases by this percentage
	DensityInfoPct float64 `yaml:"density_info_pct" json:"density_info_pct"`

	// NodeGrowthInfoPct triggers info when node count changes by this percentage
	NodeGrowthInfoPct float64 `yaml:"node_growth_info_pct" json:"node_growth_info_pct"`

	// EdgeGrowthInfoPct triggers info when edge count changes by this percentage
	EdgeGrowthInfoPct float64 `yaml:"edge_growth_info_pct" json:"edge_growth_info_pct"`

	// BlockedIncreaseThreshold triggers warning when blocked count increases by this amount
	BlockedIncreaseThreshold int `yaml:"blocked_increase_threshold" json:"blocked_increase_threshold"`

	// ActionableDecreaseWarningPct triggers warning when actionable decreases by this pct
	ActionableDecreaseWarningPct float64 `yaml:"actionable_decrease_warning_pct" json:"actionable_decrease_warning_pct"`

	// ActionableIncreaseInfoPct triggers info when actionable changes by this pct
	ActionableIncreaseInfoPct float64 `yaml:"actionable_increase_info_pct" json:"actionable_increase_info_pct"`

	// PageRankChangeWarningPct triggers warning when PageRank changes by this pct
	PageRankChangeWarningPct float64 `yaml:"pagerank_change_warning_pct" json:"pagerank_change_warning_pct"`
}

// DefaultConfig returns sensible default thresholds
func DefaultConfig() *Config {
	return &Config{
		DensityWarningPct:            50,  // 50% increase triggers warning
		DensityInfoPct:               20,  // 20% increase triggers info
		NodeGrowthInfoPct:            25,  // 25% node change triggers info
		EdgeGrowthInfoPct:            25,  // 25% edge change triggers info
		BlockedIncreaseThreshold:     5,   // 5+ more blocked issues triggers warning
		ActionableDecreaseWarningPct: 30,  // 30% decrease in actionable triggers warning
		ActionableIncreaseInfoPct:    20,  // 20% change in actionable triggers info
		PageRankChangeWarningPct:     50,  // 50% PageRank change triggers warning
	}
}

// ConfigFilename is the default config filename
const ConfigFilename = "drift.yaml"

// ConfigPath returns the default config path for a project
func ConfigPath(projectDir string) string {
	return filepath.Join(projectDir, ".bv", ConfigFilename)
}

// LoadConfig loads drift configuration from .bv/drift.yaml
// Returns default config if file doesn't exist
func LoadConfig(projectDir string) (*Config, error) {
	path := ConfigPath(projectDir)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults if no config file
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading drift config: %w", err)
	}

	config := DefaultConfig() // Start with defaults
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("parsing drift config: %w", err)
	}

	// Validate loaded config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid drift config: %w", err)
	}

	return config, nil
}

// SaveConfig saves drift configuration to .bv/drift.yaml
func SaveConfig(projectDir string, config *Config) error {
	// Validate before saving
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	path := ConfigPath(projectDir)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("encoding drift config: %w", err)
	}

	// Add header comment
	header := "# Drift detection thresholds\n# See: bv --help for drift detection options\n\n"
	content := header + string(data)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing drift config: %w", err)
	}

	return nil
}

// Validate checks that config values are sensible
func (c *Config) Validate() error {
	if c.DensityWarningPct < 0 || c.DensityWarningPct > 1000 {
		return fmt.Errorf("density_warning_pct must be between 0 and 1000")
	}
	if c.DensityInfoPct < 0 || c.DensityInfoPct > c.DensityWarningPct {
		return fmt.Errorf("density_info_pct must be between 0 and density_warning_pct")
	}
	if c.NodeGrowthInfoPct < 0 || c.NodeGrowthInfoPct > 1000 {
		return fmt.Errorf("node_growth_info_pct must be between 0 and 1000")
	}
	if c.EdgeGrowthInfoPct < 0 || c.EdgeGrowthInfoPct > 1000 {
		return fmt.Errorf("edge_growth_info_pct must be between 0 and 1000")
	}
	if c.BlockedIncreaseThreshold < 0 {
		return fmt.Errorf("blocked_increase_threshold must be non-negative")
	}
	if c.ActionableDecreaseWarningPct < 0 || c.ActionableDecreaseWarningPct > 100 {
		return fmt.Errorf("actionable_decrease_warning_pct must be between 0 and 100")
	}
	if c.ActionableIncreaseInfoPct < 0 || c.ActionableIncreaseInfoPct > 1000 {
		return fmt.Errorf("actionable_increase_info_pct must be between 0 and 1000")
	}
	if c.PageRankChangeWarningPct < 0 || c.PageRankChangeWarningPct > 1000 {
		return fmt.Errorf("pagerank_change_warning_pct must be between 0 and 1000")
	}
	return nil
}

// ExampleConfig returns an example configuration with comments
func ExampleConfig() string {
	return `# Drift detection thresholds configuration
# All percentage values are relative to baseline values

# Graph density thresholds (higher density = more interconnected)
density_warning_pct: 50    # Warn if density increases by 50%+
density_info_pct: 20       # Info if density increases by 20%+

# Node and edge count thresholds
node_growth_info_pct: 25   # Info if nodes change by 25%+
edge_growth_info_pct: 25   # Info if edges change by 25%+

# Issue status thresholds
blocked_increase_threshold: 5    # Warn if 5+ more issues are blocked
actionable_decrease_warning_pct: 30  # Warn if actionable drops 30%+
actionable_increase_info_pct: 20     # Info if actionable changes 20%+

# Metric change thresholds
pagerank_change_warning_pct: 50  # Warn if PageRank changes 50%+
`
}
