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

package heapster

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/exchange"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/logcontrol"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/processor"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/server"
	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core/ctypes"
)

const (
	pluginName    = "heapster"
	pluginVersion = 1
	pluginType    = plugin.PublisherPluginType
)

const (
	defStatsDepth   = 0
	defServerAddr   = ""
	defServerPort   = 8777
	defStatsSpanStr = "10m"
	defStatsSpan    = 10 * time.Minute
	cfgStatsDepth   = "stats_depth"
	cfgServerAddr   = "server_addr"
	cfgServerPort   = "server_port"
	cfgStatsSpan    = "stats_span"
	cfgVerboseAt    = "verbose_at"
	cfgSilentAt     = "silent_at"
	cfgMuteAt       = "mute_at"
)

// Instance references all data items publisher might need for its subsystems.
type Instance struct {
	initOnce     *sync.Once
	config       *exchange.SystemConfig
	metricMemory *exchange.MetricMemory
	processor    processor.Instance
}

// publishContext is an instance wrapping those data items necessary to perform
// metric publishing.
type publishContext interface {
	// publish a batch of metrics
	publish(metrics []plugin.MetricType)
	// Config gets the system config instance referenced by this publish context
	Config() *exchange.SystemConfig
	// Processor gets the processor instance referenced by this publish context
	Processor() processor.Instance
}

// defaultPublishContext is the default realization of publish context,
// contained in this code unit.
type defaultPublishContext struct {
	config    *exchange.SystemConfig
	processor processor.Instance
}

// ConfigMap is a wrapper on map of config items, supports getting values with
// defaults.
type ConfigMap map[string]ctypes.ConfigValue

var log *logrus.Entry

func init() {
	gob.Register(map[string]float64{})
	log = logrus.New().WithField("at", "/publisher")
	exchange.LogControl.WireLogger((*logcontrol.LogrusHandle)(log))
}

// NewPublisher returns new publisher instance with instance of SystemConfig
// and MetricMemory.
func NewPublisher() *Instance {
	return &Instance{initOnce: &sync.Once{},
		config:       exchange.NewSystemConfig(),
		metricMemory: exchange.NewMetricMemory()}
}

// Meta returns plugin metadata instance for this publisher.
func Meta() *plugin.PluginMeta {
	return plugin.NewPluginMeta(pluginName, pluginVersion, pluginType, []string{plugin.SnapGOBContentType}, []string{plugin.SnapGOBContentType}, plugin.ConcurrencyCount(99))
}

// newPublishContext returns new publishContext instance referencing
// instance of SystemConfig and processor given by caller.
var newPublishContext = func(config *exchange.SystemConfig, processor processor.Instance) publishContext {
	return &defaultPublishContext{
		config:    config,
		processor: processor,
	}
}

// Publish fulfills publisher plugin interface.
// Internally this method allocates publishContext and invokes its publish
// for batch of received metrics.
func (p *Instance) Publish(contentType string, content []byte, config map[string]ctypes.ConfigValue) error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Failure: %v\n", r)
			debug.PrintStack()
		}
	}()
	if initErr := p.ensureInitialized(ConfigMap(config)); initErr != nil {
		return fmt.Errorf("Publisher failed to initialize, error=%v\n", initErr)
	}
	ctx := newPublishContext(p.config, p.processor)

	var mts []plugin.MetricType

	switch contentType {
	case plugin.SnapGOBContentType:
		dec := gob.NewDecoder(bytes.NewBuffer(content))
		if err := dec.Decode(&mts); err != nil {
			return fmt.Errorf("Error decoding: %v", err)
		}
	default:
		return fmt.Errorf("Unknown content type '%s'", contentType)
	}
	ctx.publish(mts)
	return nil
}

// GetConfigPolicy fulfills publisher plugin interface.
func (p *Instance) GetConfigPolicy() (*cpolicy.ConfigPolicy, error) {
	cp := cpolicy.New()
	pn := cpolicy.NewPolicyNode()
	rule1, _ := cpolicy.NewStringRule(cfgServerAddr, false, defServerAddr)
	rule2, _ := cpolicy.NewIntegerRule(cfgServerPort, false, defServerPort)
	rule3, _ := cpolicy.NewIntegerRule(cfgStatsDepth, false, defStatsDepth)
	rule4, _ := cpolicy.NewStringRule(cfgStatsSpan, false, defStatsSpanStr)
	rule5, _ := cpolicy.NewStringRule(cfgVerboseAt, false, "")
	rule6, _ := cpolicy.NewStringRule(cfgSilentAt, false, "")
	rule7, _ := cpolicy.NewStringRule(cfgMuteAt, false, "")
	pn.Add(rule1, rule2, rule3, rule4, rule5, rule6, rule7)
	cp.Add([]string{}, pn)
	return cp, nil
}

// ensureInitialized runs one time initialization for subsystems of the
// publisher.
func (p *Instance) ensureInitialized(configMap ConfigMap) error {
	var serr error
	p.initOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Caught an error: %s", r)
				debug.PrintStack()
				panic(r)
			}
		}()
		if serr = initSystemConfig(p.config, configMap); serr != nil {
			return
		}
		if p.processor, serr = processor.NewProcessor(p.config, p.metricMemory); serr != nil {
			return
		}
		var serverCtx server.Context
		if serverCtx, serr = server.NewServer(p.config, p.metricMemory); serr != nil {
			return
		}
		if serr = serverCtx.AddStatusPublisher("processor", p.processor.DeliverStatus); serr != nil {
			return
		}
		if serr = serverCtx.Start(); serr != nil {
			return
		}
	})
	return serr
}

func (ctx *defaultPublishContext) publish(metrics []plugin.MetricType) {
	log.WithField("num_metrics", len(metrics)).Debug("received metrics to process")
	ctx.processor.ProcessMetrics(metrics)
}

// Config gets the system config instance referenced by this publish context
func (ctx *defaultPublishContext) Config() *exchange.SystemConfig {
	return ctx.config
}

// Processor gets the processor instance referenced by this publish context
func (ctx *defaultPublishContext) Processor() processor.Instance {
	return ctx.processor
}

// GetInt gets config value asserted as int if present, otherwise defValue.
func (m ConfigMap) GetInt(key string, defValue int) int {
	if value, gotIt := m[key]; gotIt {
		return value.(ctypes.ConfigValueInt).Value
	}
	return defValue
}

// GetStr gets config value asserted as string if present, otherwise defValue.
func (m ConfigMap) GetStr(key string, defValue string) string {
	if value, gotIt := m[key]; gotIt {
		return value.(ctypes.ConfigValueStr).Value
	}
	return defValue
}

func initSystemConfig(systemConfig *exchange.SystemConfig, configMap ConfigMap) error {
	statsSpanStr := configMap.GetStr(cfgStatsSpan, defStatsSpanStr)
	statsSpan, err := time.ParseDuration(statsSpanStr)
	if err != nil {
		return fmt.Errorf("invalid input for statsSpan: %v", err)
	}
	systemConfig.StatsSpan = statsSpan
	systemConfig.StatsDepth = configMap.GetInt(cfgStatsDepth, defStatsDepth)
	systemConfig.ServerAddr = configMap.GetStr(cfgServerAddr, defServerAddr)
	systemConfig.ServerPort = configMap.GetInt(cfgServerPort, defServerPort)
	systemConfig.VerboseAt = strings.Fields(configMap.GetStr(cfgVerboseAt, ""))
	systemConfig.SilentAt = strings.Fields(configMap.GetStr(cfgSilentAt, ""))
	systemConfig.MuteAt = strings.Fields(configMap.GetStr(cfgMuteAt, ""))
	for _, verboseAt := range systemConfig.VerboseAt {
		exchange.LogControl.SetLevel(verboseAt, int(logrus.DebugLevel))
	}
	for _, silentAt := range systemConfig.SilentAt {
		exchange.LogControl.SetLevel(silentAt, int(logrus.WarnLevel))
	}
	for _, muteAt := range systemConfig.MuteAt {
		exchange.LogControl.SetLevel(muteAt, int(logrus.ErrorLevel))
	}
	return nil
}
