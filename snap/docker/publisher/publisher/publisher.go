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

package publisher

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"

	log "github.com/Sirupsen/logrus"

	"encoding/json"
	"github.com/intelsdi-x/kubesnap/snap/docker/publisher/exchange"
	"github.com/intelsdi-x/kubesnap/snap/docker/publisher/server"
	"github.com/intelsdi-x/kubesnap/snap/docker/publisher/util"
	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core/ctypes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	name       = "heapster"
	version    = 1
	pluginType = plugin.PublisherPluginType
)

const (
	dockerMetricPrefix = "/intel/linux/docker"
	defStatsDepth      = 10
	defServerPort      = 8777
	cfgStatsDepth      = "stats_depth"
	cfgServerPort      = "server_port"
)

type core struct {
	logger         *log.Logger
	state          *exchange.InnerState
	once           sync.Once
	statsDepth     int
	metricTemplate MetricTemplate
}

func NewInnerState() *exchange.InnerState {
	res := &exchange.InnerState{
		RecentMetrics: map[string]plugin.MetricType{},
		DockerPaths:   map[string]string{},
		DockerStorage: map[string]interface{}{},
	}
	return res
}

func NewCore() (*core, error) {
	log.SetOutput(os.Stderr)
	logger := log.New()
	core := core{
		state:      NewInnerState(),
		logger:     logger,
		statsDepth: defStatsDepth,
	}
	if err := core.loadMetricTemplate(); err != nil {
		return nil, err
	}
	return &core, nil
}

func (f *core) Publish(contentType string, content []byte, config map[string]ctypes.ConfigValue) error {
	//f.logger.Printf("Publishing, YOAH! pid: %s \n", os.Getpid())
	f.ensureInitialized(config)
	var metrics []plugin.MetricType

	switch contentType {
	case plugin.SnapGOBContentType:
		dec := gob.NewDecoder(bytes.NewBuffer(content))
		if err := dec.Decode(&metrics); err != nil {
			f.logger.Printf("Error decoding: error=%v content=%v", err, content)
			return err
		}
	default:
		f.logger.Printf("Error unknown content type '%v'", contentType)
		return errors.New(fmt.Sprintf("Unknown content type '%s'", contentType))
	}
	f.state.Lock()
	f.processMetrics(metrics)
	defer f.state.Unlock()
	for _, mt := range metrics {
		f.state.RecentMetrics[mt.Namespace().String()] = mt
	}
	f.state.MetricsReceived++
	return nil
}

func Meta() *plugin.PluginMeta {
	return plugin.NewPluginMeta(
		name, version, pluginType,
		[]string{plugin.SnapGOBContentType},
		[]string{plugin.SnapGOBContentType},
		plugin.ConcurrencyCount(999))
}

func (f *core) GetConfigPolicy() (*cpolicy.ConfigPolicy, error) {
	cp := cpolicy.New()
	p := cpolicy.NewPolicyNode()
	rule1, _ := cpolicy.NewIntegerRule(cfgServerPort, false, defServerPort)
	rule2, _ := cpolicy.NewIntegerRule(cfgStatsDepth, false, defStatsDepth)
	p.Add(rule1, rule2)
	cp.Add([]string{}, p)
	return cp, nil
}

func (f *core) ensureInitialized(config map[string]ctypes.ConfigValue) {
	f.once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				f.logger.Errorf("Caught an error: %s", r)
			}
		}()
		f.statsDepth = config[cfgStatsDepth].(ctypes.ConfigValueInt).Value
		server.EnsureStarted(f.state, config[cfgServerPort].(ctypes.ConfigValueInt).Value)
	})
}

type InMetrics []plugin.MetricType

func (m InMetrics) Len() int {
	return len(m)
}

func (m InMetrics) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m InMetrics) Less(i, j int) bool {
	l := m[i].Namespace().Strings()
	r := m[j].Namespace().Strings()
	for k := 0; k < len(l) && k < len(r); k++ {
		if delta := strings.Compare(l[k], r[k]); delta != 0 {
			return delta < 0
		}
	}
	return len(l) < len(r)
}

type MetricTemplate struct {
	source      string
	statsSource string
	mapToDocker map[string]string
}

func (f *core) loadMetricTemplate() error {
	var err error
	var source string
	if source, err = f.loadTemplateSource(); err != nil {
		return err
	}
	var templateRef interface{}
	// parse template once for test
	////FIXMED: REMOVEIT \/
	//f.logger.Println("Gonna parse template source")
	if err = json.Unmarshal([]byte(source), &templateRef); err != nil {
		return err
	}
	templateObj := templateRef.(map[string]interface{})
	extractMapping := func(obj interface{}) map[string]string {
		mapping := map[string]string{}
		tmplWalker := jsonutil.NewObjWalker(obj)
		tmplWalker.Walk("/", func(target string, info os.FileInfo, _ error) error {
			if source, isStr := info.Sys().(string); isStr &&
				strings.HasPrefix(source, "/") {
				mapping[source] = target
			}
			return nil
		})
		return mapping
	}
	statsListRef, _ := jsonutil.NewObjWalker(templateObj).Seek("/stats")
	statsList := statsListRef.([]interface{})
	////FIXMED: REMOVEIT \/
	//f.logger.Printf("Found stats list: %#v \n", statsList)
	var statsObj interface{}
	statsObj, statsList = statsList[0], statsList[1:]
	////FIXMED: REMOVEIT \/
	//f.logger.Printf("Stats obj: %#v \n", statsObj)
	map[string]interface{}(templateObj)["stats"] = statsList
	////FIXMED: REMOVEIT \/
	//{
	//	templateJson, _ := json.MarshalIndent(templateObj, "", "  ")
	//	f.logger.Printf("Updated template obj: %s \n", string(templateJson))
	//}
	statsTemplate, _ := json.Marshal(statsObj)
	dockerTemplate, _ := json.Marshal(templateObj)
	f.metricTemplate = MetricTemplate{
		source:      string(dockerTemplate),
		statsSource: string(statsTemplate),
		mapToDocker: extractMapping(statsObj)}
	////FIXMED: REMOVEIT \/
	//{
	//	f.logger.Printf("err: %v; Complete  MetricTemplate instance: \n\tsource: %s,\n\tstatsSource: %s,\n\tmapping: %s \n", err, f.metricTemplate.source, f.metricTemplate.statsSource, f.metricTemplate.mapToDocker)
	//}
	////FIXMED: REMOVEIT \/
	//{
	//	var dockerObj map[string]interface{}
	//	err := json.Unmarshal([]byte(f.metricTemplate.source), &dockerObj)
	//	f.logger.Printf("sample docker obj: %#v, err: %v", dockerObj, err)
	//}

	return nil
}

func (f *core) loadTemplateSource() (string, error) {
	templateSrc := builtinMetricTemplate
	return templateSrc, nil
}

func (f *core) extractDockerIdAndPath(metric *plugin.MetricType) (string, string, bool) {
	ns := metric.Namespace().String()
	if !strings.HasPrefix(ns, dockerMetricPrefix) {
		return "", "", false
	}
	tailSplit := strings.Split(strings.TrimLeft(strings.TrimPrefix(ns, dockerMetricPrefix), "/"), "/")
	id := tailSplit[0]
	path := "/" + id
	return id, path, true
}

func (f *core) processMetrics(metrics []plugin.MetricType) {
	sort.Sort(InMetrics(metrics))
	dockerPaths := f.state.DockerPaths
	dockerStorage := f.state.DockerStorage
	temporaryStats := map[string]map[string]interface{}{}
	fetchObjectForDocker := func(id, path string, metric *plugin.MetricType) map[string]interface{} {
		//TODO: support the docker tree
		if dockerObj, gotIt := dockerStorage[path]; gotIt {
			dockerMap := dockerObj.(map[string]interface{})
			return dockerMap
		} else {
			dockerPaths[path] = id
			var dockerMap map[string]interface{}
			json.Unmarshal([]byte(f.metricTemplate.source), &dockerMap)
			dockerMap["name"] = path
			dockerMap["id"] = id

			dockerStorage[path] = dockerMap
			return dockerMap
		}
	}
	fetchObjectForDockerStats := func(id, path string, metric *plugin.MetricType) (map[string]interface{}, bool) {
		var statsObj map[string]interface{}
		var haveStats bool
		if statsObj, haveStats = temporaryStats[path]; haveStats {
			return statsObj, true
		} else if metric != nil {
			json.Unmarshal([]byte(f.metricTemplate.statsSource), &statsObj)
			statsObj["timestamp"] = metric.Timestamp().String()
			temporaryStats[path] = statsObj
			return statsObj, true
		} else {
			return statsObj, false
		}
	}
	validateDockerMetric := func(path, ns string) (string, bool) {
		for sourcePath, _ := range f.metricTemplate.mapToDocker {
			if strings.HasSuffix(ns, sourcePath) {
				return sourcePath, true
			}
		}
		customPath := ns[strings.LastIndex(ns, path)+len(path):]
		return customPath, false
	}
	insertIntoStats := func(dockerPath string, statsObj map[string]interface{}, metric *plugin.MetricType) {
		ns := metric.Namespace().String()
		////FIXMED: REMOVEIT \/
		//f.logger.Printf("  called for %s/...%s/%s \n", dockerPath, filepath.Base(filepath.Dir(ns)), filepath.Base(ns))
		if sourcePath, isExpectedMetric := validateDockerMetric(dockerPath, ns); isExpectedMetric {
			targetPath := f.metricTemplate.mapToDocker[sourcePath]
			metricParent, _ := jsonutil.NewObjWalker(statsObj).Seek(filepath.Dir(targetPath))
			metricParentMap := metricParent.(map[string]interface{})
			metricParentMap[filepath.Base(targetPath)] = metric.Data()
			f.logger.Printf("  inserting  %s => %s for %s \n", sourcePath, targetPath, dockerPath)
		} else {
			//TODO: handle custom metrics
			//snapMetricsList, _ := jsonutil.NewObjWalker(statsObj).Seek("/stats/custom_metrics/SNAP")
			//oneMetric := map[string]interface{} {}
			//oneMetric["name"] = dockerMetricPrefix + "/"+ sourcePath
			//oneMetric["type"] = "gauge"
			//oneMetric[""]
		}
	}
	mergeStatsForDocker := func(id, path string) {
		dockerObj := fetchObjectForDocker(id, path, nil)
		statsObj, haveStats := fetchObjectForDockerStats(id, path, nil)
		if !haveStats {
			return
		}
		statsList := dockerObj["stats"].([]interface{})
		if len(statsList) == f.statsDepth {
			statsList = statsList[:copy(statsList, statsList[1:])]
		}
		statsList = append(statsList, statsObj)
		dockerObj["stats"] = statsList
	}
	for _, mt := range metrics {
		if id, path, isDockerMetric := f.extractDockerIdAndPath(&mt); isDockerMetric {
			fetchObjectForDocker(id, path, &mt)
			statsObj, _ := fetchObjectForDockerStats(id, path, &mt)
			insertIntoStats(path, statsObj, &mt)
		}
	}
	for path, id := range dockerPaths {
		mergeStatsForDocker(id, path)
	}
}
