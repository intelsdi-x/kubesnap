/*
http://www.apache.org/licenses/LICENSE-2.0.txt


Copyright 2016 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package exchange groups together pieces of data shared between publisher
// components and some common configuration.
package exchange

import (
	"sync"
	"time"

	cadv "github.com/google/cadvisor/info/v1"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/logcontrol"
)

// StatsRequest is a struct representing HTTP request for stats.
type StatsRequest struct {
	// ContainerName is name of the container to retrieve stats for
	// (defaults to root or '/').
	ContainerName string `json:"containerName,omitempty"`

	// NumStats defines amount of stats to return per container, maximum.
	// Ignored if start and end time is given.
	NumStats int `json:"num_stats,omitempty"`

	// Start defines time window limiting the stats to fetch per container.
	// Defaults to beginning of time.
	Start time.Time `json:"start,omitempty"`

	// End defines time window limiting the stats to fetch per container.
	// Defaults to current time.
	End time.Time `json:"end,omitempty"`

	// Subcontainers controls depth of query, allowing to fetch stats for
	// subcontainers.
	Subcontainers bool `json:"subcontainers,omitempty"`
}

// MetricMemory constitutes a metric storage shared between processor
// and server parts of the plugin.
// MetricMemory is equipped with RW lock that should be obtained separately for
// read and update operations.
type MetricMemory struct {
	sync.RWMutex
	// ContainerMap refers to all containers known to publisher.
	ContainerMap map[string]*cadv.ContainerInfo
	// PendingMetrics holds custom metrics that await publishing.
	PendingMetrics map[string]map[string][]cadv.MetricVal
}

// SystemConfig contains all configuration items extracted from plugin's
// config in task manifest.
type SystemConfig struct {
	// StatsDepth limits the maximum number of stats that should be
	//buffered for a container. Disabled if zero. Evaluated in combination
	//with StatsSpan.
	StatsDepth int
	// StatsSpan limits the maximum time span of stats that should
	//be buffered for a container. Stats will be limited by this limit and
	//the value of StatsDepth.
	StatsSpan time.Duration
	// ServerAddr is an address that the server should bind to.
	ServerAddr string
	// ServerPort is a port number that the server should listen at.
	ServerPort int
	// VerboseAt holds a list of logger name prefixes (field 'at')
	//that should be set to verbose (Debug level).
	VerboseAt []string
	// SilentAt holds a list of logger name prefixes (field 'at')
	//that should be set to silent (Warning level).
	SilentAt []string
	// MuteAt holds a list of logger name prefixes (field 'at')
	//that should be set to mute (Error level).
	MuteAt []string
}

// Subsystem is a common interface for all publisher subsystems
type Subsystem struct {
	config *SystemConfig
	memory *MetricMemory
}

// LogControl is a controller for loggers shared between publisher subsystems.
// Each subsystem should this package first, and wire its logger.
var LogControl = &logcontrol.LogControl{}

// NewMetricMemory return new instance of metric memory to be shared between
// subsystems.
func NewMetricMemory() *MetricMemory {
	return &MetricMemory{
		ContainerMap:   map[string]*cadv.ContainerInfo{},
		PendingMetrics: map[string]map[string][]cadv.MetricVal{},
	}
}

// NewSystemConfig returns zero-valued instance of SystemConfig.
func NewSystemConfig() *SystemConfig {
	return &SystemConfig{}
}
