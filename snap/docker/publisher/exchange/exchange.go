package exchange

import (
	"github.com/intelsdi-x/snap/control/plugin"
	"sync"
	"time"
)

//type Status struct {
//	Message string `json:"msg"`
//	Data    *StatsRequest
//	Count   int
//	Names   map[string]string
//}

type StatsRequest struct {
	// The name of the container for which to request stats.
	// Default: /
	ContainerName string `json:"containerName,omitempty"`

	// Max number of stats to return.
	// If start and end time are specified this limit is ignored.
	// Default: 60
	NumStats int `json:"num_stats,omitempty"`

	// Start time for which to query information.
	// If omitted, the beginning of time is assumed.
	Start time.Time `json:"start,omitempty"`

	// End time for which to query information.
	// If omitted, current time is assumed.
	End time.Time `json:"end,omitempty"`

	// Whether to also include information from subcontainers.
	// Default: false.
	Subcontainers bool `json:"subcontainers,omitempty"`
}

type InnerState struct {
	sync.RWMutex
	MetricsReceived int
	RecentMetrics   map[string]plugin.MetricType
	DockerPaths     map[string]string
	DockerStorage   map[string]interface{}
}
