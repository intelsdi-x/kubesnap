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

package processor

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	cadv "github.com/google/cadvisor/info/v1"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/exchange"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/jsonutil"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/logcontrol"
	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/core"
)

const (
	customMetricName          = "custom_metric_name"
	customMetricType          = "custom_metric_type"
	customMetricFormat        = "custom_metric_format"
	customMetricUnits         = "custom_metric_units"
	customMetricContainerPath = "custom_metric_container_path"

	defCustomMetricType          = "gauge"
	defCustomMetricFormat        = "int"
	defCustomMetricUnits         = "none"
	defCustomMetricContainerPath = "/"
)

var clog *logrus.Entry

func init() {
	var l = logrus.New()
	clog = l.WithField("at", "/processor/custom")
	exchange.LogControl.WireLogger((*logcontrol.LogrusHandle)(clog))
}

// mergeCustomMetricsFor flushes pending custom metric values into
// container object identified by containerPath.
// Pending metric values are copied into container's Stats element that has
// timestamp greater than or equal to each value's timestamp.
func (p *processingContext) mergeCustomMetricsFor(containerPath string) {
	containerValuesMap, gotContainerValuesMap := p.memory.PendingMetrics[containerPath]
	if !gotContainerValuesMap {
		return
	}
	container := p.memory.ContainerMap[containerPath]
	for _, statsElem := range container.Stats {
		refStamp := statsElem.Timestamp
		for metricName, valueList := range containerValuesMap {
			filteredPendingValues := make([]cadv.MetricVal, 0, len(valueList))
			for _, value := range valueList {
				if !value.Timestamp.Before(refStamp) {
					filteredPendingValues = append(filteredPendingValues, value)
					continue
				}
				if statsElem.CustomMetrics == nil {
					statsElem.CustomMetrics = map[string][]cadv.MetricVal{}
				}
				statsElem.CustomMetrics[metricName] = append(statsElem.CustomMetrics[metricName], value)
				p.statsNumCustomMetricsRx++
			}
			containerValuesMap[metricName] = filteredPendingValues
		}
	}
}

// discardTooOldCustomValuesFor removes pending custom metric values
// that will never be flushed.
// Pending custom values that are older than any of Stats elements are discarded
// from pending values for given container.
func (p *processingContext) discardTooOldCustomValuesFor(containerPath string) {
	containerValuesMap, gotContainerValuesMap := p.memory.PendingMetrics[containerPath]
	if !gotContainerValuesMap {
		return
	}
	container := p.memory.ContainerMap[containerPath]
	var oldestStamp *time.Time
	for _, statsElem := range container.Stats {
		refStamp := statsElem.Timestamp
		if oldestStamp == nil || refStamp.Before(*oldestStamp) {
			oldestStamp = &refStamp
		}
	}
	if oldestStamp == nil {
		return
	}
	for metricName, valueList := range containerValuesMap {
		for i, value := range valueList {
			if value.Timestamp.Before(*oldestStamp) {
				valueList = append(valueList[:i], valueList[i+1:]...)
			} else {
				p.statsNumStaleMetricsDd++
			}
		}
		containerValuesMap[metricName] = valueList
	}
}

// ingestCustomMetrics scans tree of metrics for items that are
// tagged as custom.
// Extracted custom metric Specs are reported in a list.
func (p *processingContext) ingestCustomMetrics(mtree map[string]interface{}) {
	jsonutil.NewObjWalker(mtree).Walk("/", func(path string, info os.FileInfo, err error) error {
		if metric, isMetric := info.Sys().(*plugin.MetricType); isMetric {
			p.ingestCustomMetricsFrom(metric)
		}
		return nil
	})
}

// ingestCustomMetricsFrom extracts custom metric Spec item(s) basing
// on information associated with given metric instance.
// Discovered metric Spec's are inserted into container's Spec structure to
// serve as blueprints for actual custom metric values.
func (p *processingContext) ingestCustomMetricsFrom(metric *plugin.MetricType) {
	containerPath, specs, validCustomMetric := p.extractCustomMetrics(metric)
	if !validCustomMetric {
		return
	}
	container, knowncontainer := p.memory.ContainerMap[containerPath]
	if !knowncontainer {
		return
	}
	p.insertIntoCustomMetrics(containerPath, container, specs, metric)
}

// insertIntoCustomMetrics inserts custom metric specs into
// container's Spec structure and the custom metric values into pending list.
func (p *processingContext) insertIntoCustomMetrics(containerPath string, container *cadv.ContainerInfo, specs []cadv.MetricSpec, metric *plugin.MetricType) {
	values := p.extractCustomValues(metric, specs)
	metricList := &container.Spec.CustomMetrics
	dbgValuesIn := []string{}
	numExtracted := 0
	for _, spec := range specs {
		foundSpec := false
		for _, ckSpec := range *metricList {
			if ckSpec.Name == spec.Name {
				foundSpec = true
				break
			}
		}
		if !foundSpec {
			*metricList = append(*metricList, spec)
			p.statsNumCustomSpecsRx++
		}
		customVal, validVal := values[spec.Name]
		if !validVal {
			clog.WithField("metric_name", spec.Name).Debug("found no instance value for custom metric", spec.Name)
			continue
		}
		dbgValuesIn = append(dbgValuesIn, spec.Name)

		// find room for custom metrics
		dockerValuesMap, gotDockerValuesMap := p.memory.PendingMetrics[containerPath]
		if !gotDockerValuesMap {
			dockerValuesMap = map[string][]cadv.MetricVal{}
			p.memory.PendingMetrics[containerPath] = dockerValuesMap
		}
		statsList, _ := dockerValuesMap[spec.Name]
		statsList = append(statsList, customVal)
		numExtracted++
		dockerValuesMap[spec.Name] = statsList
	}
	if numExtracted > 0 {
		clog.WithFields(logrus.Fields{"num_extracted": numExtracted, "value_names": dbgValuesIn}).Debug("extracted custom metrics")
	}
}

// extractCustomMetrics extracts custom metric Spec item(s) basing
// on information associated with given metric instance.
// Metric Data may represent a map. In that case each individual map entry
// will be represented by distinct metric Spec in the resulting list.
func (p *processingContext) extractCustomMetrics(metric *plugin.MetricType) (containerPath string, specs []cadv.MetricSpec, valid bool) {
	copyMetric := func(metric plugin.MetricType, nsSuffix string) *plugin.MetricType {
		res := metric
		res.Tags_ = make(map[string]string, len(metric.Tags_))
		for k, v := range metric.Tags_ {
			res.Tags_[k] = v
		}
		res.Namespace_ = append(core.NewNamespace(), metric.Namespace_...)
		if nsSuffix != "" {
			res.Namespace_ = core.Namespace(res.Namespace_).AddStaticElement(nsSuffix)
		}
		return &res
	}
	var valueMap map[string]float64
	var isMap bool
	if valueMap, isMap = metric.Data().(map[string]float64); !isMap {
		containerPath1, spec1, valid1 := p.extractOneCustomMetric(metric)
		if !valid1 {
			return "", specs, false
		}
		return containerPath1, []cadv.MetricSpec{spec1}, true
	}
	specs = make([]cadv.MetricSpec, 0, len(valueMap))
	valid = false
	containerPath = ""
	for k, v := range valueMap {
		nuMetric := copyMetric(*metric, k)
		nuMetric.Data_ = v
		containerPath1, spec1, valid1 := p.extractOneCustomMetric(nuMetric)
		if containerPath == "" {
			containerPath = containerPath1
		}
		valid = valid || valid1
		specs = append(specs, spec1)
	}
	if valid {
		return containerPath, specs, true
	}
	return "", specs[:0], false
}

// extractOneCustomMetric extracts one custom metric Spec basing on
// information associated with given metric instance.
// Function expects metric Data to be assignable to primitive types - Float or
// Int.
func (p *processingContext) extractOneCustomMetric(metric *plugin.MetricType) (containerPath string, spec cadv.MetricSpec, valid bool) {
	tags := metric.Tags()
	ns := metric.Namespace()
	containerPath = ""
	spec = cadv.MetricSpec{
		Type:   defCustomMetricType,
		Format: defCustomMetricFormat,
	}
	var haveName, haveType, haveFormat, haveUnits, havecontainerPath bool
	if spec.Name, haveName = tags[customMetricName]; !haveName {
		spec.Name = strings.Join(ns.Strings(), "/")
	}
	tmpTag := ""
	if tmpTag, haveType = tags[customMetricType]; haveType {
		spec.Type = cadv.MetricType(tmpTag)
	}
	if tmpTag, haveFormat = tags[customMetricFormat]; haveFormat {
		spec.Format = cadv.DataType(tmpTag)
	}
	if spec.Units, haveUnits = tags[customMetricUnits]; !haveUnits {
		spec.Units = defCustomMetricUnits
	}
	if containerPath, havecontainerPath = tags[customMetricContainerPath]; !havecontainerPath {
		containerPath = defCustomMetricContainerPath
	}
	if haveName || haveType || haveFormat || haveUnits || havecontainerPath {
		return containerPath, spec, true
	}
	return "", spec, false
}

// extractCustomValues extracts custom metric Values basing on metric
// instance and the input list of metric Specs.
func (p *processingContext) extractCustomValues(metric *plugin.MetricType, specs []cadv.MetricSpec) map[string]cadv.MetricVal {
	res := make(map[string]cadv.MetricVal, len(specs))
	if valueMap, isMap := metric.Data_.(map[string]float64); !isMap {
		clog.WithFields(logrus.Fields{"ns": metric.Namespace().String(), "value_type": reflect.TypeOf(metric.Data()), "value_specs": specs}).Debug("probing instance of custom value")
		value, ok := p.extractOneCustomValue(&specs[0], metric.Timestamp_, metric.Data_)
		if ok {
			res[specs[0].Name] = value
		}
	} else {
		for _, spec := range specs {
			value, ok := p.extractOneCustomValue(&spec, metric.Timestamp_, valueMap[filepath.Base(spec.Name)])
			clog.WithFields(logrus.Fields{"ns": metric.Namespace().String(), "value_key": spec.Name, "value_type": reflect.TypeOf(metric.Data()), "value_specs": specs}).Debug("probing instance of custom value within map")
			if ok {
				res[spec.Name] = value
			}
		}
	}
	return res
}

// extractOneCustomValue extracts a single custom metric Value basing
// on given metric spec and metric value.
func (p *processingContext) extractOneCustomValue(spec *cadv.MetricSpec, timeStamp time.Time, value interface{}) (cadv.MetricVal, bool) {
	customVal := cadv.MetricVal{Timestamp: timeStamp}
	switch spec.Format {
	case cadv.IntType:
		switch i := value.(type) {
		case int64:
			customVal.IntValue = int64(i)
		case uint64:
			customVal.IntValue = int64(i)
		case int32:
			customVal.IntValue = int64(i)
		case uint32:
			customVal.IntValue = int64(i)
		case uint16:
			customVal.IntValue = int64(i)
		case int16:
			customVal.IntValue = int64(i)
		case uint8:
			customVal.IntValue = int64(i)
		case int8:
			customVal.IntValue = int64(i)
		case uint:
			customVal.IntValue = int64(i)
		case int:
			customVal.IntValue = int64(i)
		default:
			clog.WithField("metric_name", spec.Name).Warn("custom metric cant be handled as IntValue")
			return customVal, false
		}
	case cadv.FloatType:
		switch i := value.(type) {
		case float32:
			customVal.FloatValue = float64(i)
		case float64:
			customVal.FloatValue = float64(i)
		case int64:
			customVal.FloatValue = float64(i)
		case uint64:
			customVal.FloatValue = float64(i)
		case int32:
			customVal.FloatValue = float64(i)
		case uint32:
			customVal.FloatValue = float64(i)
		case int16:
			customVal.FloatValue = float64(i)
		case uint16:
			customVal.FloatValue = float64(i)
		case int8:
			customVal.FloatValue = float64(i)
		case uint8:
			customVal.FloatValue = float64(i)
		case int:
			customVal.FloatValue = float64(i)
		case uint:
			customVal.FloatValue = float64(i)
		default:
			clog.WithField("metric_name", spec.Name).Warn("custom metric cant be handled as FloatValue")
			return customVal, false
		}
	}
	return customVal, true
}
