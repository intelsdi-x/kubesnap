// +build small

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
	"errors"
	"testing"
	"time"

	"github.com/intelsdi-x/snap-plugin-publisher-heapster/exchange"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/mox"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/processor"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/server"
	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/core"
	"github.com/intelsdi-x/snap/core/ctypes"
	. "github.com/smartystreets/goconvey/convey"
)

type dummyPublishContext struct {
	report    []string
	extractor func(metric *plugin.MetricType) (string, bool)
	config    *exchange.SystemConfig
	processor processor.Instance
}

type mockServerContext struct {
	mox.CallMock
}

type mockProcessorInstance struct {
	mox.CallMock
}

func TestNewPublisher(t *testing.T) {
	Convey("When NewPublisher is invoked", t, func() {
		Convey("it should never fail", func() {
			So(func() {
				NewPublisher()
			}, ShouldNotPanic)
		})
		instance := NewPublisher()
		Convey("it should return non-nil result", func() {
			So(instance, ShouldNotBeNil)
			Convey("with all fields initialized", func() {
				So(instance.initOnce, ShouldNotBeNil)
				So(instance.config, ShouldNotBeNil)
				So(instance.metricMemory, ShouldNotBeNil)
			})
		})
	})
}

func TestMeta(t *testing.T) {
	Convey("When Meta is invoked", t, func() {
		Convey("it should never fail", func() {
			So(func() {
				Meta()
			}, ShouldNotPanic)
		})
		meta := Meta()
		Convey("it should return non-nil result", func() {
			So(meta, ShouldNotBeNil)
			Convey("with correct values for plugin metadata", func() {
				So(meta.Name, ShouldEqual, pluginName)
				So(meta.Version, ShouldEqual, pluginVersion)
				So(meta.Type, ShouldEqual, pluginType)
			})
		})
	})
}

func TestNewPublishContext(t *testing.T) {
	Convey("When NewPublishContext is invoked", t, func() {
		dummyConfig := &exchange.SystemConfig{}
		dummyProcessor := &processor.DefaultInstance{}
		Convey("it should never fail", func() {
			So(func() {
				newPublishContext(dummyConfig, dummyProcessor)
			}, ShouldNotPanic)
		})
		Convey("it should produce non-nil result", func() {
			context := newPublishContext(dummyConfig, dummyProcessor)
			So(context, ShouldNotBeNil)
			Convey("with all fields initialized", func() {
				So(context.Config(), ShouldEqual, dummyConfig)
				So(context.Processor(), ShouldEqual, dummyProcessor)
			})
		})
	})
}

func TestInstance_Publish(t *testing.T) {
	var buf bytes.Buffer
	metrics := []plugin.MetricType{
		*plugin.NewMetricType(core.NewNamespace("mocks", "mock"), time.Now(), nil, "", 1),
	}
	enc := gob.NewEncoder(&buf)
	enc.Encode(metrics)
	config := map[string]ctypes.ConfigValue{}
	Convey("With new publisher instance", t, func() {
		publisher := NewPublisher()
		// skip publisher default initialization
		publisher.initOnce.Do(func() {})
		dummyContext := newDummyPublishContextReportingNamespaces()
		var oldPublishContext func(_ *exchange.SystemConfig, _ processor.Instance) publishContext
		oldPublishContext, newPublishContext = newPublishContext, func(_ *exchange.SystemConfig, _ processor.Instance) publishContext {
			return dummyContext
		}
		Convey("requested to publish encoded metrics", func() {
			err := publisher.Publish(plugin.SnapGOBContentType, buf.Bytes(), config)
			Convey("the Publish call should not fail", func() {
				So(err, ShouldBeNil)
				Convey("but metrics should be properly decoded and passed for processing", func() {
					So(dummyContext.report, ShouldResemble, []string{"/mocks/mock"})
				})
			})
		})
		Reset(func() {
			newPublishContext = oldPublishContext
		})
	})
}

func TestInstance_GetConfigPolicy(t *testing.T) {
	Convey("Working with new publisher instance", t, func() {
		publisher := NewPublisher()
		Convey("GetConfigPolicy should create without errors a non-nil policy", func() {
			policy, err := publisher.GetConfigPolicy()
			So(err, ShouldBeNil)
			So(policy, ShouldNotBeNil)
		})
	})
}

func TestInstance_ensureInitialized(t *testing.T) {
	Convey("Working with new publisher instance", t, func() {
		publisher := NewPublisher()
		config := ConfigMap{}
		var oldProcessorCtor func(config *exchange.SystemConfig, memory *exchange.MetricMemory) (processor.Instance, error)
		var oldServerCtor func(config *exchange.SystemConfig, memory *exchange.MetricMemory) (server.Context, error)
		mockProcessor := &mockProcessorInstance{}
		mockServer := &mockServerContext{}
		processorCount := 0
		serverCount := 0
		oldProcessorCtor, processor.NewProcessor = processor.NewProcessor, func(config *exchange.SystemConfig, memory *exchange.MetricMemory) (processor.Instance, error) {
			processorCount++
			return mockProcessor, nil
		}
		oldServerCtor, server.NewServer = server.NewServer, func(config *exchange.SystemConfig, memory *exchange.MetricMemory) (server.Context, error) {
			serverCount++
			return mockServer, nil
		}
		Convey("initialization invokes all subsystems", func() {
			publisher.ensureInitialized(config)
			So(publisher.config.ServerPort, ShouldNotEqual, 0)
			So(processorCount, ShouldBeGreaterThan, 0)
			So(serverCount, ShouldBeGreaterThan, 0)
		})
		Convey("initialization takes place only once", func() {
			publisher.ensureInitialized(config)
			publisher.ensureInitialized(config)
			So(processorCount, ShouldEqual, 1)
			So(serverCount, ShouldEqual, 1)
		})
		Convey("errors during initialization are propagated to calling code", func() {
			Convey("including plugin config initialization error", func() {
				config[cfgStatsSpan] = ctypes.ConfigValueStr{Value: "9.9.9"}
				err := publisher.ensureInitialized(config)
				So(err, ShouldNotBeNil)
			})
			Convey("including processor constructor error", func() {
				processor.NewProcessor = func(config *exchange.SystemConfig, memory *exchange.MetricMemory) (processor.Instance, error) {
					return nil, errors.New("processor ctor failed")
				}
				err := publisher.ensureInitialized(config)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "processor ctor failed")
			})
			Convey("including server constructor error", func() {
				server.NewServer = func(config *exchange.SystemConfig, memory *exchange.MetricMemory) (server.Context, error) {
					return nil, errors.New("server ctor failed")
				}
				err := publisher.ensureInitialized(config)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "server ctor failed")
			})
			Convey("including server/ AddStatusPublisher error", func() {
				mockServer.AddInterceptor(func(funcName string, _ []interface{}, result *mox.Results) bool {
					if funcName == "AddStatusPublisher" {
						(*result)[0] = errors.New("AddStatusPublisher failed")
						return true
					}
					return false
				})
				err := publisher.ensureInitialized(config)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "AddStatusPublisher failed")
			})
			Convey("including server/ Start error", func() {
				mockServer.AddInterceptor(func(funcName string, _ []interface{}, result *mox.Results) bool {
					if funcName == "Start" {
						(*result)[0] = errors.New("Start failed")
						return true
					}
					return false
				})
				err := publisher.ensureInitialized(config)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Start failed")
			})
			Reset(func() {
				processor.NewProcessor = oldProcessorCtor
				server.NewServer = oldServerCtor
			})
		})
		Reset(func() {
			processor.NewProcessor = oldProcessorCtor
			server.NewServer = oldServerCtor
		})
	})
}

func TestDefaultPublishContext_publish(t *testing.T) {
	expMetrics := []plugin.MetricType{
		*plugin.NewMetricType(core.NewNamespace("mocks", "mock"), time.Now(), nil, "", 1),
		*plugin.NewMetricType(core.NewNamespace("mocks", "rock"), time.Now(), nil, "", 99),
	}
	Convey("Given a default publish context", t, func() {
		Convey("and a batch of sample metrics", func() {
			mockProcessor := &mockProcessorInstance{}
			ctx := &defaultPublishContext{processor: mockProcessor}
			Convey("a call to publish should not fail", func() {
				So(func() {
					ctx.publish(expMetrics)
				}, ShouldNotPanic)
				Convey("after invoking a processor", func() {
					So(mockProcessor.GetAllCalled(), ShouldResemble, []string{"ProcessMetrics"})
					Convey("with unchanged batch of metrics passed for processing", func() {
						entries := mockProcessor.GetCallsOf("ProcessMetrics")
						So(len(entries), ShouldEqual, 1)
						entry := entries[0]
						actMetrics := entry[1].([]plugin.MetricType)
						So(len(actMetrics), ShouldEqual, 2)
					})
				})
			})
		})
	})
}

func TestConfigMap(t *testing.T) {
	Convey("Given a config map filled with values", t, func() {
		m := ConfigMap{
			"foo": ctypes.ConfigValueInt{Value: 5},
			"bar": ctypes.ConfigValueStr{Value: "bonk"}}
		Convey("Getter methods correctly report existing entries", func() {
			So(m.GetInt("foo", -1), ShouldEqual, 5)
			So(m.GetStr("bar", "XX"), ShouldEqual, "bonk")
		})
		Convey("Getter methods correctly report defaults for missing entries", func() {
			So(m.GetInt("NOFOO", -1), ShouldEqual, -1)
			So(m.GetStr("NOBAR", "XX"), ShouldEqual, "XX")
		})
	})
}

func TestInitSystemConfig(t *testing.T) {
	makeInt := func(v int) ctypes.ConfigValueInt {
		return ctypes.ConfigValueInt{Value: v}
	}
	makeStr := func(v string) ctypes.ConfigValueStr {
		return ctypes.ConfigValueStr{Value: v}
	}
	Convey("Working in heapster package", t, func() {
		Convey("having empty configuration for plugin", func() {
			configMap := ConfigMap{}
			Convey("system config is correctly initialized with defaults", func() {
				config := exchange.NewSystemConfig()
				initSystemConfig(config, configMap)
				So(config.StatsDepth, ShouldEqual, defStatsDepth)
				So(config.ServerAddr, ShouldEqual, defServerAddr)
				So(config.ServerPort, ShouldEqual, defServerPort)
				So(config.VerboseAt, ShouldBeEmpty)
				So(config.SilentAt, ShouldBeEmpty)
				So(config.MuteAt, ShouldBeEmpty)
				So(config.StatsSpan, ShouldEqual, defStatsSpan)
			})
		})
		Convey("having specific configuration for plugin", func() {
			configMap := ConfigMap{
				cfgStatsDepth: makeInt(100),
				cfgServerAddr: makeStr("statserver"),
				cfgServerPort: makeInt(33933),
				cfgVerboseAt:  makeStr("/server /heapster"),
				cfgSilentAt:   makeStr("/processor/main /processor/custom"),
				cfgMuteAt:     makeStr("/processor/custom /server"),
				cfgStatsSpan:  makeStr("123m"),
			}
			Convey("system config is correctly initialized with parsed values", func() {
				config := exchange.NewSystemConfig()
				initSystemConfig(config, configMap)
				So(config.StatsDepth, ShouldEqual, 100)
				So(config.ServerAddr, ShouldEqual, "statserver")
				So(config.ServerPort, ShouldEqual, 33933)
				So(config.VerboseAt, ShouldResemble, []string{"/server", "/heapster"})
				So(config.SilentAt, ShouldResemble, []string{"/processor/main", "/processor/custom"})
				So(config.MuteAt, ShouldResemble, []string{"/processor/custom", "/server"})
				So(config.StatsSpan, ShouldEqual, 123*time.Minute)
			})
		})
	})
}

func newDummyPublishContextReportingNamespaces() *dummyPublishContext {
	return &dummyPublishContext{
		config: exchange.NewSystemConfig(),
		extractor: func(metric *plugin.MetricType) (string, bool) {
			return metric.Namespace().String(), true
		},
	}
}

func (p *dummyPublishContext) publish(metrics []plugin.MetricType) {
	for _, m := range metrics {
		if entry, accepted := p.extractor(&m); accepted {
			p.report = append(p.report, entry)
		}
	}
}

func (d *dummyPublishContext) Config() *exchange.SystemConfig {
	return d.config
}

func (d *dummyPublishContext) Processor() processor.Instance {
	return d.processor
}

func (c *mockServerContext) Config() *exchange.SystemConfig {
	res := c.Called("Config", 1)
	return res[0].(*exchange.SystemConfig)
}

func (c *mockServerContext) Memory() *exchange.MetricMemory {
	res := c.Called("Memory", 1)
	return res[0].(*exchange.MetricMemory)
}

func (c *mockServerContext) AddStatusPublisher(name string, statusPublisher server.StatusPublisherFunc) error {
	res := c.Called("AddStatusPublisher", 1, name, statusPublisher)
	return res.Error(0)
}

func (c *mockServerContext) Start() error {
	res := c.Called("Start", 1)
	return res.Error(0)
}

func (p *mockProcessorInstance) Config() *exchange.SystemConfig {
	res := p.Called("Config", 1)
	return res[0].(*exchange.SystemConfig)
}

func (p *mockProcessorInstance) Memory() *exchange.MetricMemory {
	res := p.Called("Memory", 1)
	return res[0].(*exchange.MetricMemory)
}

func (p *mockProcessorInstance) ProcessMetrics(rawMetrics []plugin.MetricType) {
	p.Called("ProcessMetrics", 0, rawMetrics)
}

func (p *mockProcessorInstance) DeliverStatus() interface{} {
	res := p.Called("DeliverStatus", 1)
	return res[0]
}
