package service

import "github.com/codingconcepts/env"

type Env struct {
	MaxConcurrentReconciles int `env:"MAX_CONCURRENT_RECONCILES" default:"5"`

	ClusterInternalDNS string `env:"CLUSTER_INTERNAL_DNS" default:"cluster.local"`
	MultiClusterDNS    string `env:"MULTI_CLUSTER_DNS" required:"false"`
	KloudliteDNS       string `env:"KLOUDLITE_DNS" required:"false"`
}

func LoadEnv() (*Env, error) {
	var ev Env
	if err := env.Set(&ev); err != nil {
		return nil, err
	}
	return &ev, nil
}
