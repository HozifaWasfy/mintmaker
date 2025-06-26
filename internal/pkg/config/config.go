// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"errors"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/konflux-ci/mintmaker/internal/pkg/constant"
)

const MMConfigMapName = "mintmaker-controller-configmap"

type ConfigReader struct {
	MaxParallelPipelineruns string `yaml:"max-parallel-pipelineruns"`
	GhTokenValidity         string `yaml:"github-token-validity-mins"`
	GhTokenUsageWindow      string `yaml:"github-teoken-usage-window-mins"`
}

type Config struct {
	MaxParallelPipelineruns int
	GhTokenValidity         time.Duration
	GhTokenUsageWindow      time.Duration
	GhTokenRenewThreshold   time.Duration
}

var globalConfig *Config

func DefaultConfig() *Config {
	GhTokenValidity := 60 * time.Minute
	GhTokenUsageWindow := 30 * time.Minute

	return &Config{
		MaxParallelPipelineruns: 40,
		GhTokenValidity:         GhTokenValidity,
		GhTokenUsageWindow:      GhTokenUsageWindow,
		GhTokenRenewThreshold:   GhTokenValidity - GhTokenUsageWindow,
	}
}

func LoadConfig(client client.Client) (*Config, error) {
	configReader := &ConfigReader{}
	defaultConfig := DefaultConfig()
	config := &Config{}

	configMap := &corev1.ConfigMap{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Namespace: constant.MintMakerNamespaceName,
		Name:      MMConfigMapName,
	}, configMap)
	if err != nil {
		return defaultConfig, err
	}

	if err := yaml.Unmarshal([]byte(configMap.Data["controller-config"]), configReader); err != nil {
		return defaultConfig, err
	}

	if parsed, err := strconv.Atoi(configReader.MaxParallelPipelineruns); err == nil && parsed > 0 {
		config.MaxParallelPipelineruns = parsed
	} else {
		config.MaxParallelPipelineruns = defaultConfig.MaxParallelPipelineruns
	}

	if parsed, err := time.ParseDuration(configReader.GhTokenValidity); err == nil && parsed > 0 {
		config.GhTokenValidity = parsed
	} else {
		config.GhTokenValidity = defaultConfig.GhTokenValidity
	}

	if parsed, err := time.ParseDuration(configReader.GhTokenUsageWindow); err == nil && parsed > 0 {
		config.GhTokenUsageWindow = parsed
	} else {
		config.GhTokenUsageWindow = defaultConfig.GhTokenUsageWindow
	}

	if config.GhTokenUsageWindow >= config.GhTokenValidity {
		config.GhTokenValidity = defaultConfig.GhTokenValidity
		config.GhTokenUsageWindow = defaultConfig.GhTokenUsageWindow
		return config, errors.New("GitHub token usage window must be less than token validity")
	}

	config.GhTokenRenewThreshold = config.GhTokenValidity - config.GhTokenUsageWindow

	return config, nil
}

// Will not return empty configs but error for logging purposses
func InitGlobalConfig(client client.Client) error {
	config, err := LoadConfig(client)
	globalConfig = config
	return err
}

func GetConfig() *Config {
	return globalConfig
}

// Get testing config
func GetTestConfig() Config {
	return *DefaultConfig()
}
