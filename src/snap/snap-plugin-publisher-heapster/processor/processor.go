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

// Package processor contains all routines performing processing on
// incoming metrics.
package processor

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	cadv "github.com/google/cadvisor/info/v1"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/exchange"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/jsonutil"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/logcontrol"
	"github.com/intelsdi-x/snap/control/plugin"
)

const (
	dockerMetricPrefix = "/intel/docker"
)

// DefaultInstance wires together all data elements required to perform metrics
// processing.
type DefaultInstance struct {
	config *exchange.SystemConfig
	memory *exchange.MetricMemory
	stats  *processorStats
}

// processorStats holds diagnostic indicators for monitoring load on
// processor part of plugin.
// This type is equipped with RW lock that should be obtained separately for
// read and update operations.
type processorStats struct {
	sync.RWMutex
	MetricsRxTotal       int `json:"metrics_received_total"`
	MetricsRxLast        int `json:"metrics_received_last"`
	ContainersRxLast     int `json:"containers_received_last"`
	ContainersRxMax      int `json:"containers_received_max"`
	CustomMetricsRxTotal int `json:"custom_metric_values_received_total"`
	CustomMetricsRxLast  int `json:"custom_metric_values_received_last"`
	CustomMetricsDdTotal int `json:"custom_metric_values_discarded_total"`
	CustomMetricsDdLast  int `json:"custom_metric_values_discarded_last"`
	CustomSpecsRxTotal   int `json:"custom_metric_specs_received_total"`
	CustomSpecsRxLast    int `json:"custom_metric_specs_received_last"`
}

// processingContext holds data required for a processing run on a single
// batch of metrics.
type processingContext struct {
	*DefaultInstance
	statsContainersPcsdMap map[string]bool
	// number of custom metrics (values) that were stored in exposed container stats
	statsNumCustomMetricsRx int
	// number of custom metric specs that were added to containers
	statsNumCustomSpecsRx int
	// number of custom metric values that were discarded for being too old
	statsNumStaleMetricsDd int
}

type wrappedMetrics struct {
	rawMetrics []plugin.MetricType
}

type wrappedMetric struct {
	rawMetric *plugin.MetricType
}

// valueFilter is a tool for extracting values and retaining errors from
// functions returning a value and an error
type valueFilter struct {
	context      string
	errorTracker error
	errorLast    error
}

// Instance wires together all data elements required to perform metrics
// processing.
type Instance interface {
	// Config gets the system config instance referenced by this processor
	Config() *exchange.SystemConfig
	// Memory gets the metric memory instance referenced by this processor
	Memory() *exchange.MetricMemory
	// ProcessMetrics initiates a processing run on a batch of metrics.
	ProcessMetrics(rawMetrics []plugin.MetricType)
	// DeliverStatus provides a data structure reflecting state of
	// processor part for diagnostic purposes.
	DeliverStatus() interface{}
}

// MetricMap is a wrapper around map of metrics (possibly nested) allowing to
// parse its values into specific types
type MetricMap map[string]interface{}

var (
	log            *logrus.Entry
	dummyTimestamp time.Time
	hostBootTime   *time.Time
)

func init() {
	var l = logrus.New()
	log = l.WithField("at", "/processor/main")
	exchange.LogControl.WireLogger((*logcontrol.LogrusHandle)(log))
	dummyTimestamp, _ = ParseTime("2003-03-03T03:03:03Z")
}

// NewProcessor returns instance of ProcessorInstance referencing SystemConfig
// and MetricMemory instances given by caller.
var NewProcessor = func(config *exchange.SystemConfig, memory *exchange.MetricMemory) (Instance, error) {
	processor := DefaultInstance{
		config: config,
		memory: memory,
		stats:  &processorStats{},
	}
	return &processor, nil
}

// ParseTime parses string representation of time as produced by Time.String().
func ParseTime(str string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05Z07:00", str)
}

func getHostBootTime() time.Time {
	if hostBootTime != nil {
		return *hostBootTime
	}
	bootTime := time.Now()
	if uptimeBytes, err := ioutil.ReadFile("/proc/uptime"); err == nil {
		uptimeSplit := strings.Fields(string(uptimeBytes))
		if len(uptimeSplit) == 0 {
			goto onBootTimeSet
		}
		uptimeSeconds, err := strconv.ParseFloat(uptimeSplit[0], 64)
		if err != nil {
			goto onBootTimeSet
		}
		bootTime = bootTime.Add(time.Duration(-1 * int64(uptimeSeconds) * int64(time.Second)))
	}
onBootTimeSet:
	hostBootTime = &bootTime
	return *hostBootTime
}

// ProcessMetrics initiates a processing run on a batch of metrics.
// This function engages following locks:
// - Lock on metric memory (ProcessorInstance.memory),
// - Lock on processor stats (ProcessorInstance.stats).
func (p *DefaultInstance) ProcessMetrics(rawMetrics []plugin.MetricType) {
	p.memory.Lock()
	defer p.memory.Unlock()
	p.stats.Lock()
	defer p.stats.Unlock()

	ctx := processingContext{
		DefaultInstance:        p,
		statsContainersPcsdMap: map[string]bool{},
	}
	metrics := &wrappedMetrics{rawMetrics: rawMetrics}
	ctx.processMetrics(metrics)
}

// DeliverStatus provides a data structure reflecting state of
// processor part for diagnostic purposes.
func (p *DefaultInstance) DeliverStatus() interface{} {
	p.stats.RLock()
	defer p.stats.RUnlock()
	statsSnapshot := *p.stats
	return statsSnapshot
}

// Config gets the config referenced by this processor instance
func (p *DefaultInstance) Config() *exchange.SystemConfig {
	return p.config
}

// Memory gets the metric memory referenced by this processor instance
func (p *DefaultInstance) Memory() *exchange.MetricMemory {
	return p.memory
}

// Len returns number of metrics in the list
func (m *wrappedMetrics) Len() int {
	return len(m.rawMetrics)
}

// Item returns MetricHandle item created from raw metric in the list
func (m *wrappedMetrics) Item(index int) jsonutil.MetricHandle {
	item := &wrappedMetric{rawMetric: &(*m).rawMetrics[index]}
	return item
}

// Path returns object path identifying namespace of metric
func (m *wrappedMetric) Path() jsonutil.ObjectPath {
	rawPath := strings.TrimLeft(m.rawMetric.Namespace().String(), "/")
	path := jsonutil.ObjectPath{Literal: rawPath, Split: m.rawMetric.Namespace().Strings()}
	return path
}

// Data returns data carried by metric instance
func (m *wrappedMetric) Data() interface{} {
	return m.rawMetric.Data()
}

// RawMetric returns the underlying raw metric instance
func (m *wrappedMetric) RawMetric() interface{} {
	return m.rawMetric
}

func (p *processingContext) processMetrics(metrics jsonutil.MetricList) {
	mtree := jsonutil.RebuildObjectFromMetrics(metrics,
		func(path []string, m jsonutil.MetricHandle) interface{} {
			return m.RawMetric()
		})
	var containerPaths []string
	if dtree, err := jsonutil.NewObjWalker(mtree).Seek(dockerMetricPrefix); err == nil {
		// remove docker metrics from tree, remaining ones are custom metrics
		outerTree, _ := jsonutil.NewObjWalker(mtree).Seek(filepath.Dir(dockerMetricPrefix))
		delete(outerTree.(map[string]interface{}), dockerMetricPrefix)
		jsonutil.PruneEmptySubtrees(mtree)
		containerPaths = p.ingestDockersTree(dtree.(map[string]interface{}))
	}
	p.ingestCustomMetrics(mtree)
	if len(containerPaths) > 0 {
		for _, containerPath := range containerPaths {
			p.mergeCustomMetricsFor(containerPath)
			p.discardTooOldCustomValuesFor(containerPath)
		}
	}
	// update diagnostic info
	p.stats.MetricsRxLast = metrics.Len()
	p.stats.MetricsRxTotal += metrics.Len()
	if len(p.statsContainersPcsdMap) > p.stats.ContainersRxMax {
		p.stats.ContainersRxMax = len(p.statsContainersPcsdMap)
	}
	p.stats.ContainersRxLast = len(p.statsContainersPcsdMap)
	p.stats.CustomMetricsRxLast = p.statsNumCustomMetricsRx
	p.stats.CustomMetricsRxTotal += p.statsNumCustomMetricsRx
	p.stats.CustomSpecsRxLast = p.statsNumCustomSpecsRx
	p.stats.CustomSpecsRxTotal += p.statsNumCustomSpecsRx
	p.stats.CustomMetricsDdLast = p.statsNumStaleMetricsDd
	p.stats.CustomMetricsDdTotal += p.statsNumStaleMetricsDd
}

// ingestDockersTree processes a tree of metrics generated by
// docker collector plugin.
func (p *processingContext) ingestDockersTree(dtree map[string]interface{}) (containerPaths []string) {
	for dockerID, dockerMetrics := range dtree {
		id := dockerID
		if id == "root" {
			id = "/"
		}
		if _, haveContainer := p.memory.ContainerMap[id]; !haveContainer {
			log.WithField("id", id).Debug("building info structures for new container")
			p.memory.ContainerMap[id] = p.ingestDockerTree(id, dockerMetrics.(map[string]interface{}))
		}
		container := p.memory.ContainerMap[id]
		p.statsContainersPcsdMap[id] = true
		p.updateContainerStats(container, dockerMetrics.(map[string]interface{}))
		containerPaths = append(containerPaths, container.Name)
	}
	return containerPaths
}

func (p *processingContext) ingestDockerTree(dockerID string, dtree map[string]interface{}) (res *cadv.ContainerInfo) {
	res = &cadv.ContainerInfo{}
	res.Id = dockerID
	res.Name = filepath.Join("/", dockerID)
	p.ingestContainerSpec(dockerID, res, dtree)
	return res
}

func (p *processingContext) ingestContainerLabels(dockerID string, container *cadv.ContainerInfo, dtree map[string]interface{}) {
	vf := newValueFilter()
	defer func() {
		if metricErr, gotError := vf.anyError(); gotError {
			log.WithField("error", metricErr).Warning("1 couldn't handle all incoming metrics, had to use defaults")
		}
	}()
	vf.enter(fmt.Sprintf("%s/spec", container.Id))
	labelsRawObj := vf.filter1(jsonutil.NewObjWalker(dtree).Seek("/spec/labels"))
	if labelsRawObj == nil {
		return
	}
	labelsMap := MetricMap(labelsRawObj.(map[string]interface{})).FlattenValues()
	container.Labels = map[string]string{}
	for k := range labelsMap {
		v, _ := labelsMap.Get(k)
		if v != nil {
			k = strings.Replace(k,"_",".",-1)
			container.Labels[k] = fmt.Sprint(v)
		} else {
			container.Labels[k] = "null"
		}
	}
}

func (p *processingContext) ingestContainerSpec(dockerID string, container *cadv.ContainerInfo, dtree map[string]interface{}) {
	vf := newValueFilter()
	vf.enter(fmt.Sprintf("%s/spec", dockerID))
	defer func() {
		if metricErr, gotError := vf.anyError(); gotError {
			log.WithField("error", metricErr).Warning("couldn't handle all incoming metrics, had to use defaults")
		}
	}()
	spec := &container.Spec
	specRawObj, gotSpec := dtree["spec"]
	if gotSpec {
		p.ingestContainerLabels(dockerID, container, dtree)
		spec.Labels = container.Labels
		specMap := MetricMap(specRawObj.(map[string]interface{}))
		spec.CreationTime = vf.filter1(specMap.GetTimeFromStr("creation_time", dummyTimestamp)).(time.Time)
		spec.Image = vf.filter1(specMap.GetStr("image_name", "")).(string)
	} else if dockerID == "/" {
		// ok to have no /spec branch for root container
		spec.CreationTime = getHostBootTime()
	}
	vf.enter(fmt.Sprintf("%s/memory_stats", dockerID))
	if memStatsRawObj := vf.filter1(jsonutil.NewObjWalker(dtree).Seek("/stats/cgroups/memory_stats/stats")); memStatsRawObj != nil {
		spec.HasMemory = true
		memStatsMap := MetricMap(memStatsRawObj.(map[string]interface{}))
		spec.Memory.Limit = vf.filter1(memStatsMap.GetUint64("limit_in_bytes", 0)).(uint64)
		spec.Memory.SwapLimit = vf.filter1(memStatsMap.GetUint64("swap_limit_in_bytes", 0)).(uint64)
	} 

	spec.HasCpu = true
	spec.HasNetwork = true
	spec.HasFilesystem = true
	spec.HasCustomMetrics = true
}

// updateContainerStats fills Stats structure of ContainerInfo
// with values extracted from container metrics.
func (p *processingContext) updateContainerStats(container *cadv.ContainerInfo, metrics map[string]interface{}) {
	vf := newValueFilter()
	vf.enter(fmt.Sprintf("%s/spec", container.Id))
	defer func() {
		if metricErr, gotError := vf.anyError(); gotError {
			log.WithField("error", metricErr).Warning("couldn't handle all incoming metrics, had to use defaults")
		}
	}()
	// scan the metrics to discover timestamp of most recent metric
	var timestamp *time.Time
	objWalker := jsonutil.NewObjWalker(metrics)
	statsObj := vf.filter1(objWalker.Seek("/stats"))
	if statsObj == nil {
		return
	}
	objWalker.Walk("/stats", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if v, isMetric := info.Sys().(*plugin.MetricType); isMetric {
			if timestamp == nil || timestamp.Before(v.Timestamp()) {
				t := v.Timestamp()
				timestamp = &t
			}
		}
		return nil
	})
	stats := cadv.ContainerStats{}
	stats.Timestamp = *timestamp
	p.ingestCPUStats(container, &stats, statsObj.(map[string]interface{}))
	p.ingestMemoryStats(container, &stats, statsObj.(map[string]interface{}))
	p.ingestNetworkStats(container, &stats, statsObj.(map[string]interface{}))
	p.ingestConnectionStats(container, &stats, statsObj.(map[string]interface{}))
	p.ingestFsStats(container, &stats, statsObj.(map[string]interface{}))
	p.makeRoomForStats(&container.Stats, &stats)
	container.Stats = append(container.Stats, &stats)
}

func (p *processingContext) ingestCPUStats(container *cadv.ContainerInfo, stats *cadv.ContainerStats, statsMap map[string]interface{}) {
	vf := newValueFilter()
	defer func() {
		if metricErr, gotError := vf.anyError(); gotError {
			log.WithField("error", metricErr).Warning("4 couldn't handle all incoming metrics, had to use defaults")
		}
	}()
	vf.enter(fmt.Sprintf("%s/cpu_stats", container.Id))
	usageObj := vf.filter1(jsonutil.NewObjWalker(statsMap).Seek("/cgroups/cpu_stats/cpu_usage"))
        if usageObj != nil {
                cpuUsage := &stats.Cpu.Usage
		usageMap := MetricMap(usageObj.(map[string]interface{}))
		cpuUsage.System = vf.filter1(usageMap.GetUint64("usage_in_kernelmode", 0)).(uint64)
		cpuUsage.User = vf.filter1(usageMap.GetUint64("usage_in_usermode", 0)).(uint64)
		cpuUsage.Total = vf.filter1(usageMap.GetUint64("total_usage", 0)).(uint64)
	}
	perCPUObj := vf.filter1(jsonutil.NewObjWalker(statsMap).Seek("/cgroups/cpu_stats/cpu_usage/percpu_usage"))
	if perCPUObj == nil {
		return
	}
	cpuMap := map[string]uint64{}
	cpuMax := -1
	vf.enter(fmt.Sprintf("%s/percpu_usage", container.Id))
	cpuUsage := &stats.Cpu.Usage
	perCPUMap := MetricMap(perCPUObj.(map[string]interface{})).FlattenValues()
	for k := range perCPUMap {
		vf.enter(fmt.Sprintf("%s/percpu_usage", container.Id))
		cpuNum := vf.filter1(strconv.Atoi(k)).(int)
		if _, gotError := vf.lastError(); gotError {
			continue
		}
		vf.enter(fmt.Sprintf("%s/percpu_usage/%s", container.Id, cpuNum))
		if cpuNum > cpuMax {
			cpuMax = cpuNum
		}
		cpuMap[k] = vf.filter1(perCPUMap.GetUint64(k, 0)).(uint64)
	}
	for i := 0; i <= cpuMax; i++ {
		if usage, gotUsage := cpuMap[strconv.Itoa(i)]; gotUsage {
			cpuUsage.PerCpu = append(cpuUsage.PerCpu, usage)
		} else {
			cpuUsage.PerCpu = append(cpuUsage.PerCpu, 0)
		}
	}
}

func (p *processingContext) ingestMemoryStats(container *cadv.ContainerInfo, stats *cadv.ContainerStats, statsMap map[string]interface{}) {
	vf := newValueFilter()
	defer func() {
		if metricErr, gotError := vf.anyError(); gotError {
			log.WithField("error", metricErr).Warning("5 couldn't handle all incoming metrics, had to use defaults")
		}
	}()
	vf.enter(fmt.Sprintf("%s/memory_stats", container.Id))
	memMetricsObj, _ := jsonutil.NewObjWalker(statsMap).Seek("/cgroups/memory_stats")
	memStatsObj := vf.filter1(jsonutil.NewObjWalker(statsMap).Seek("/cgroups/memory_stats/stats"))
	memUsageObj := vf.filter1(jsonutil.NewObjWalker(statsMap).Seek("/cgroups/memory_stats/usage"))
	if memMetricsObj == nil {
		return
	}
	memStats := &stats.Memory
	memMetricsMap := MetricMap(memMetricsObj.(map[string]interface{}))
	memStats.Cache = vf.filter1(memMetricsMap.GetUint64("cache", 0)).(uint64)
	if memUsageObj != nil {
		memUsageMap := MetricMap(memUsageObj.(map[string]interface{}))
		memStats.Usage = vf.filter1(memUsageMap.GetUint64("usage", 0)).(uint64)
		memStats.Failcnt = vf.filter1(memUsageMap.GetUint64("failcnt", 0)).(uint64)
	}
	if memStatsObj != nil {
		memStatsMap := MetricMap(memStatsObj.(map[string]interface{}))
		memStats.RSS = vf.filter1(memStatsMap.GetUint64("rss", 0)).(uint64)
		memStats.ContainerData.Pgfault = vf.filter1(memStatsMap.GetUint64("pgfault", 0)).(uint64)
		memStats.ContainerData.Pgmajfault = vf.filter1(memStatsMap.GetUint64("pgmajfault", 0)).(uint64)
	}
}

func (p *processingContext) ingestNetworkStats(container *cadv.ContainerInfo, stats *cadv.ContainerStats, statsMap map[string]interface{}) {
	vf := newValueFilter()
	defer func() {
		if metricErr, gotError := vf.anyError(); gotError {
			log.WithField("error", metricErr).Warning("couldn't handle all incoming metrics, had to use defaults")
		}
	}()
	vf.enter(fmt.Sprintf("%s/network", container.Id))
	networksObj := vf.filter1(jsonutil.NewObjWalker(statsMap).Seek("/network"))
	if networksObj == nil {
		return
	}
	ifaceMap := map[string]cadv.InterfaceStats{}
	ifaceNames := []string{}
	for name, netMetricsObj := range networksObj.(map[string]interface{}) {
		vf.enter(fmt.Sprintf("%s/network/%s", container.Id, name))
		netMetricsMap := MetricMap(netMetricsObj.(map[string]interface{}))
		if name != "total" {
			ifaceNames = append(ifaceNames, name)
		}
		iface := cadv.InterfaceStats{}
		iface.Name = name
		iface.RxBytes = vf.filter1(netMetricsMap.GetUint64("rx_bytes", 0)).(uint64)
		iface.RxPackets = vf.filter1(netMetricsMap.GetUint64("rx_packets", 0)).(uint64)
		iface.RxErrors = vf.filter1(netMetricsMap.GetUint64("rx_errors", 0)).(uint64)
		iface.RxDropped = vf.filter1(netMetricsMap.GetUint64("rx_dropped", 0)).(uint64)
		iface.TxBytes = vf.filter1(netMetricsMap.GetUint64("tx_bytes", 0)).(uint64)
		iface.TxPackets = vf.filter1(netMetricsMap.GetUint64("tx_packets", 0)).(uint64)
		iface.TxErrors = vf.filter1(netMetricsMap.GetUint64("tx_errors", 0)).(uint64)
		iface.TxDropped = vf.filter1(netMetricsMap.GetUint64("tx_dropped", 0)).(uint64)
		ifaceMap[name] = iface
	}
	vf.enter(fmt.Sprintf("%s/network", container.Id))
	if totalStats, gotTotal := ifaceMap["total"]; gotTotal {
		stats.Network.InterfaceStats = totalStats
	} else {
		log.WithField("context", vf.getContext()).Warning("found no /total interface")
	}
	sort.Strings(ifaceNames)
	for _, name := range ifaceNames {
		stats.Network.Interfaces = append(stats.Network.Interfaces, ifaceMap[name])
	}
}

func (p *processingContext) ingestConnectionStats(container *cadv.ContainerInfo, stats *cadv.ContainerStats, statsMap map[string]interface{}) {
	vf := newValueFilter()
	defer func() {
		if metricErr, gotError := vf.anyError(); gotError {
			log.WithField("error", metricErr).Warning("couldn't handle all incoming metrics, had to use defaults")
		}
	}()
	ingestTCPStats := func(dest *cadv.TcpStat, srcMap MetricMap) {
		dest.Close = vf.filter1(srcMap.GetUint64("close", 0)).(uint64)
		dest.CloseWait = vf.filter1(srcMap.GetUint64("close_wait", 0)).(uint64)
		dest.Closing = vf.filter1(srcMap.GetUint64("closing", 0)).(uint64)
		dest.Established = vf.filter1(srcMap.GetUint64("established", 0)).(uint64)
		dest.FinWait1 = vf.filter1(srcMap.GetUint64("fin_wait1", 0)).(uint64)
		dest.FinWait2 = vf.filter1(srcMap.GetUint64("fin_wait2", 0)).(uint64)
		dest.LastAck = vf.filter1(srcMap.GetUint64("last_ack", 0)).(uint64)
		dest.Listen = vf.filter1(srcMap.GetUint64("listen", 0)).(uint64)
		dest.SynRecv = vf.filter1(srcMap.GetUint64("syn_recv", 0)).(uint64)
		dest.SynSent = vf.filter1(srcMap.GetUint64("syn_sent", 0)).(uint64)
		dest.TimeWait = vf.filter1(srcMap.GetUint64("time_wait", 0)).(uint64)
	}
	vf.enter(fmt.Sprintf("%s/connection", container.Id))
	tcpObj := vf.filter1(jsonutil.NewObjWalker(statsMap).Seek("/connection/tcp"))
	if tcpObj != nil {
		vf.enter(fmt.Sprintf("%s/tcp", container.Id))
		tcpStats := &stats.Network.Tcp
		ingestTCPStats(tcpStats, MetricMap(tcpObj.(map[string]interface{})))
	}
	vf.enter(fmt.Sprintf("%s/connection", container.Id))
	tcp6Obj := vf.filter1(jsonutil.NewObjWalker(statsMap).Seek("/connection/tcp6"))
	if tcp6Obj != nil {
		vf.enter(fmt.Sprintf("%s/tcp6", container.Id))
		tcp6Stats := &stats.Network.Tcp6
		ingestTCPStats(tcp6Stats, MetricMap(tcp6Obj.(map[string]interface{})))
	}
}

func (p *processingContext) ingestFsStats(container *cadv.ContainerInfo, stats *cadv.ContainerStats, statsMap map[string]interface{}) {
	vf := newValueFilter()
	defer func() {
		if metricErr, gotError := vf.anyError(); gotError {
			log.WithField("error", metricErr).Warning("couldn't handle all incoming metrics, had to use defaults")
		}
	}()
	vf.enter(fmt.Sprintf("%s/filesystem", container.Id))
	fsObj := vf.filter1(jsonutil.NewObjWalker(statsMap).Seek("/filesystem"))
	if fsObj == nil {
		return
	}
	fsMap := map[string]cadv.FsStats{}
	devNames := []string{}
	for name, fsMetricsObj := range fsObj.(map[string]interface{}) {
		vf.enter(fmt.Sprintf("%s/filesystem/%s", container.Id, name))
		fsMetricsMap := MetricMap(fsMetricsObj.(map[string]interface{}))
		devNames = append(devNames, name)
		fs := cadv.FsStats{}
		fs.Device = name
		fsType, _ := fsMetricsMap.Get("type")
		if fsType != nil {
			fs.Type = fmt.Sprint(fsType)
		} else {
			fs.Type = "null"
		}
		fs.Limit = vf.filter1(fsMetricsMap.GetUint64("capacity", 0)).(uint64)
		fs.Usage = vf.filter1(fsMetricsMap.GetUint64("usage", 0)).(uint64)
		fs.BaseUsage = vf.filter1(fsMetricsMap.GetUint64("base_usage", 0)).(uint64)
		fs.Available = vf.filter1(fsMetricsMap.GetUint64("available", 0)).(uint64)
		fs.InodesFree = vf.filter1(fsMetricsMap.GetUint64("inodes_free", 0)).(uint64)
		fs.ReadsCompleted = vf.filter1(fsMetricsMap.GetUint64("reads_completed", 0)).(uint64)
		fs.ReadsMerged = vf.filter1(fsMetricsMap.GetUint64("reads_merged", 0)).(uint64)
		fs.SectorsRead = vf.filter1(fsMetricsMap.GetUint64("sectors_read", 0)).(uint64)
		fs.ReadTime = vf.filter1(fsMetricsMap.GetUint64("read_time", 0)).(uint64)
		fs.WritesCompleted = vf.filter1(fsMetricsMap.GetUint64("writes_completed", 0)).(uint64)
		fs.WritesMerged = vf.filter1(fsMetricsMap.GetUint64("writes_merged", 0)).(uint64)
		fs.SectorsWritten = vf.filter1(fsMetricsMap.GetUint64("sectors_written", 0)).(uint64)
		fs.WriteTime = vf.filter1(fsMetricsMap.GetUint64("write_time", 0)).(uint64)
		fs.IoInProgress = vf.filter1(fsMetricsMap.GetUint64("io_in_progress", 0)).(uint64)
		fs.IoTime = vf.filter1(fsMetricsMap.GetUint64("io_time", 0)).(uint64)
		fs.WeightedIoTime = vf.filter1(fsMetricsMap.GetUint64("weighted_io_time", 0)).(uint64)
		fsMap[name] = fs
	}
	sort.Strings(devNames)
	for _, name := range devNames {
		stats.Filesystem = append(stats.Filesystem, fsMap[name])
	}
}

// makeRoomForStats performs filtering and truncation on list of
// container stats so that incoming Stats element fits within configured range
// of stats (stats_depth and stats_span).
func (p *processingContext) makeRoomForStats(destList *[]*cadv.ContainerStats, stats *cadv.ContainerStats) {
	validOfs := 0
	statsList := *destList
	if p.config.StatsDepth > 0 && len(statsList) == p.config.StatsDepth {
		validOfs++
	}
	if p.config.StatsSpan <= 0 {
		if validOfs > 0 {
			statsList = statsList[:copy(statsList, statsList[validOfs:])]
			*destList = statsList
		}
		return
	}
	nuStamp := stats.Timestamp
	for validOfs < len(statsList) {
		ckStamp := statsList[validOfs].Timestamp
		span := nuStamp.Sub(ckStamp)
		if span <= p.config.StatsSpan {
			break
		}
		validOfs++
	}
	if validOfs > 0 {
		statsList = statsList[:copy(statsList, statsList[validOfs:])]
		*destList = statsList
	}
}

// Get extracts Data value from metric entry within map
func (m MetricMap) Get(key string) (interface{}, bool) {
	if v, gotValue := m[key]; gotValue {
		return v.(*plugin.MetricType).Data(), true
	}
	return nil, false
}

// GetTimeFromStr parses map entry as instance of Time, parsing defValue if
// value is missing, and returns optional parsing errors.
func (m MetricMap) GetTimeFromStr(key string, defValue time.Time) (time.Time, error) {
	if v, gotValue := m.Get(key); gotValue {
		if vstr, gotStr := v.(string); gotStr {
			return ParseTime(vstr)
		}
		return time.Time{}, fmt.Errorf("can't handle metric map entry '%s' as string input for time parser", key)
	}
	return defValue, nil
}

// GetStr parses map entry as instance of string, returning defValue if entry
// is missing; return error if underlying value is not a string
func (m MetricMap) GetStr(key, defValue string) (string, error) {
	if v, gotValue := m.Get(key); gotValue {
		if res, gotStr := v.(string); gotStr {
			return res, nil
		}
		return "", fmt.Errorf("can't handle metric map entry '%s' as string", key)
	}
	return defValue, nil
}

// GetUint64 parses map entry as instance of uint64, falls back to default value
// if entry is missing, and returns optional parsing errors.
func (m MetricMap) GetUint64(key string, defValue uint64) (uint64, error) {
	if v, gotValue := m.Get(key); gotValue {
		if res, gotUint := v.(uint64); gotUint {
			return res, nil
		}
		return 0, fmt.Errorf("can't handle metric map entry '%s' as uint64", key)
	}
	return defValue, nil
}

// FlattenValues returns a new map instance with keys mapping directly to their
// value, so {a: {value: 5}, b: {value: -1}} becomes {a:5, b:-1}.
func (m MetricMap) FlattenValues() MetricMap {
	res := map[string]interface{}{}
	for k, vObj := range m {
		if vMap, isMap := vObj.(map[string]interface{}); !isMap {
			continue
		} else {
			if _, gotValue := vMap["value"]; !gotValue {
				continue
			}
			res[k] = vMap["value"]
		}
	}
	return MetricMap(res)
}

// newValueFilter creates new instance of the value filter referencing given
// error variable.
func newValueFilter() *valueFilter {
	res := valueFilter{}
	return &res
}

// filter1 extracts 1 value from the given value, error tuple, retaining error
// if no previous error was set.
// Function clears last invocation error before filtering. Filtering status may
// be checked with a call to lastError().
func (v *valueFilter) filter1(value interface{}, err error) interface{} {
	v.clearLast()
	if err != nil && v.errorTracker == nil {
		if v.context != "" {
			outputErr := fmt.Errorf("%s: %v", v.context, err)
			v.errorTracker = outputErr
		} else {
			v.errorTracker = err
		}
	}
	return value
}

// getContext returns context currently set for this filter
func (v *valueFilter) getContext() string {
	return v.context
}

// enter sets new context in this filter, and returns pointer to same instance
func (v *valueFilter) enter(newContext string) *valueFilter {
	v.context = newContext
	return v
}

// clearLast clears last invocation error
func (v *valueFilter) clearLast() {
	v.errorLast = nil
}

// anyError returns tracked error and true, if any error was collected during
// filtering
func (v *valueFilter) anyError() (error, bool) {
	if v.errorTracker != nil {
		return v.errorTracker, true
	}
	return nil, false
}

// lastError returns error, true for last invocation of filterX() method, if that
// one has collected any error.
func (v *valueFilter) lastError() (error, bool) {
	if v.errorLast != nil {
		return v.errorLast, true
	}
	return nil, false
}
