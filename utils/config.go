package utils

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type LBStrategy int

// define enum
const (
	RoundRobin LBStrategy = iota
	LeastConnected
)

func GetLBStrategy(strategy string) LBStrategy {
	switch strategy {
	case "least-connected":
		return LeastConnected
	default:
		return RoundRobin
	}
}

type Config struct {
	Port           int      `yaml:"lb_port"`
	MaxAttempLimit int      `yaml:"max_attempt_limit"`
	Backends       []string `yaml:"backends"`
	Strategy       string   `yaml:"strategy"`
}

const MAX_LB_ATTEMPTS int = 3

// reads and parses the load balancer configuration from the "config.yaml" file.
// It returns a pointer to a Config struct representing the parsed configuration.
func GetLBConfig() (*Config, error) {
	var config Config
	configFile, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		return nil, err
	}

	if len(config.Backends) == 0 {
		return nil, errors.New("backend host expected, none provided")
	}

	if config.Port == 0 {
		return nil, errors.New("load balancer port not found")
	}

	return &config, nil
}
