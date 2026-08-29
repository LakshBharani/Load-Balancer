package config

// AppConfig is the top-level YAML config shape.
//
// The high-level structure: backends define individual ip:port endpoints,
// clusters are named aliases for a group of backends, and rules match
// incoming client CIDR:port against a set of targets (backend ids or
// cluster names) balanced with a given strategy.
type AppConfig struct {
	HealthcheckAddr string              `yaml:"healthcheck_addr"`
	Backends        []BackendConfig     `yaml:"backends"`
	Clusters        map[string][]string `yaml:"clusters"`
	Rules           []RuleConfig        `yaml:"rules"`
}

type BackendConfig struct {
	ID string `yaml:"id"`
	IP string `yaml:"ip"`
}

type RuleConfig struct {
	Clients  []string `yaml:"clients"`
	Targets  []string `yaml:"targets"`
	Strategy Strategy `yaml:"strategy"`
}

// Strategy mirrors a tagged union: `type` selects the algorithm, with
// algorithm-specific fields alongside it (e.g. Adaptive's Coefficients/Alpha).
type Strategy struct {
	Type         string     `yaml:"type"`
	Coefficients [4]float64 `yaml:"coefficients"`
	Alpha        float64    `yaml:"alpha"`
}

const (
	StrategyRoundRobin       = "RoundRobin"
	StrategySourceIPHash     = "SourceIPHash"
	StrategyLeastConnections = "LeastConnections"
	StrategyAdaptive         = "Adaptive"
)

func defaultHealthcheckAddr() string {
	return "0.0.0.0:9000"
}

// Normalize fills in defaults left blank in the YAML source.
func (c *AppConfig) Normalize() {
	if c.HealthcheckAddr == "" {
		c.HealthcheckAddr = defaultHealthcheckAddr()
	}
}
