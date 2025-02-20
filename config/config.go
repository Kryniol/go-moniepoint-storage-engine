package config

import "fmt"

type Mode string

const (
	Leader   Mode = "leader"
	Follower Mode = "follower"
)

type Config struct {
	HTTPPort     string
	DataPath     string
	Mode         Mode
	LeaderHost   string
	ReplicaHosts []string
}

func NewConfig(restPort, dataPath string, mode Mode, leaderHost string, replicaHosts []string) (*Config, error) {
	if mode != Leader && mode != Follower {
		return nil, fmt.Errorf("invalid mode: %s", mode)
	}

	if mode == Leader && leaderHost != "" {
		return nil, fmt.Errorf("cannot specify leader host in leader mode")
	}

	if mode == Follower && leaderHost == "" {
		return nil, fmt.Errorf("leader host must be specified in follower mode")
	}

	return &Config{
		HTTPPort:     restPort,
		DataPath:     dataPath,
		Mode:         mode,
		LeaderHost:   leaderHost,
		ReplicaHosts: replicaHosts,
	}, nil
}
