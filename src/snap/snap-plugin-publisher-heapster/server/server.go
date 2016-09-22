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

// Package server contains functions and types building server for getting
// the metrics.
package server

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	cadv "github.com/google/cadvisor/info/v1"
	"github.com/gorilla/mux"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/exchange"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/jsonutil"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/logcontrol"
)

const maxBodySizeBytes = 1048576

var (
	log *logrus.Entry
	// jsonCodec refers to package wide JSON codec implementation
	serverJSONCodec jsonutil.JSONCodec = &jsonutil.StdJSONCodec{}
)

// HTTPDriver offers subset of server features needed to drive stats server
type HTTPDriver interface {
	// AddRoute creates a route with specific handler. On a failure the
	// error will indicate what path has failed.
	AddRoute(methods []string, path string, handler http.HandlerFunc) error
	// ListenAndServe starts server polling loop
	ListenAndServe(serverAddr string) error
}

// DefaultHTTPDriver is a default implementation of http driver, building on
// standard http package server.
type DefaultHTTPDriver struct {
	router *mux.Router
}

// AddRoute creates a route with specific handler
func (d *DefaultHTTPDriver) AddRoute(methods []string, path string, handler http.HandlerFunc) error {
	d.router.Methods(methods...).Path(path).HandlerFunc(handler)
	return nil
}

// ListenAndServe starts server polling loop
func (d *DefaultHTTPDriver) ListenAndServe(serverAddr string) error {
	err := http.ListenAndServe(serverAddr, d.router)
	return err
}

// StatusPublisherFunc describes a function that intends to publish
// diagnostic status available below  /_status/ branch.
// Status publisher should return a simple object valid for JSON marshalling.
type StatusPublisherFunc func() interface{}

// Context wires data structures and configuration necessary for managing
// a stats server.
type Context interface {
	// Config returns the system config instance referenced by this server
	Config() *exchange.SystemConfig
	// Memory returns the metric memory instance referenced by this server
	Memory() *exchange.MetricMemory
	// AddStatusPublisher registers a function intended to report
	// diagnostic status for some part of the plugin.
	// Function will be invoked in response to request for /_status/[name].
	AddStatusPublisher(name string, statusPublisher StatusPublisherFunc) error
	// Start starts the server's listening thread (separate goroutine).
	Start() error
}

// DefaultContext is a default implementation of a stats server context.
type DefaultContext struct {
	// config refers to plugin's config.
	config *exchange.SystemConfig
	// memory refers to stored metrics available for report.
	memory *exchange.MetricMemory
	// stats refers to internal diagnostic stats of server part.
	stats *stats
	// driver is an instance supposed to configure and launch server
	driver HTTPDriver
}

type stats struct {
	sync.RWMutex
	StatsTxMax   int `json:"stats_tx_max"`
	StatsTxTotal int `json:"stats_tx_total"`
	StatsTxLast  int `json:"stats_tx_last"`
	StatsDdMax   int `json:"stats_dd_max"`
	StatsDdTotal int `json:"stats_dd_total"`
	StatsDdLast  int `json:"stats_dd_last"`
}

// statsListType is a wrapper around slice of ContainerStats to
// facilitate stats sorting.
type statsListType []*cadv.ContainerStats

// newHTTPDriver is a constructor function delivering a http driver to build and
// launch http server.
var newHTTPDriver = func() HTTPDriver {
	d := DefaultHTTPDriver{}
	d.router = mux.NewRouter().StrictSlash(true)
	return &d
}

// Len fulfills Interface from sort package.
func (s statsListType) Len() int {
	return len(s)
}

// Swap fulfills Interface from sort package.
func (s statsListType) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less fulfills Interface from sort package.
// Orders Stats elements in reverse order, putting most recent items first.
func (s statsListType) Less(i, j int) bool {
	l, r := s[i], s[j]
	return !l.Timestamp.Before(r.Timestamp)
}

func init() {
	var l = logrus.New()
	log = l.WithField("at", "/server")
	exchange.LogControl.WireLogger((*logcontrol.LogrusHandle)(log))
}

func newDefaultContext(config *exchange.SystemConfig, memory *exchange.MetricMemory) DefaultContext {
	ctx := DefaultContext{
		config: config,
		memory: memory,
		stats:  &stats{},
	}
	return ctx
}

// NewServer builds a server instance referencing SystemConfig and
// MetricMemory instances passed by caller.
var NewServer = func(config *exchange.SystemConfig, memory *exchange.MetricMemory) (Context, error) {
	server := newDefaultContext(config, memory)
	if err := server.setup(); err != nil {
		log.WithField("error", err).Error("Server setup failed")
		return nil, err
	}
	return &server, nil
}

// AddStatusPublisher registers a function intended to report
// diagnostic status for some part of the plugin.
// Function will be invoked in response to request for /_status/[name].
func (s *DefaultContext) AddStatusPublisher(name string, statusPublisher StatusPublisherFunc) error {
	if err := s.driver.AddRoute([]string{"GET"}, fmt.Sprintf("/_status/%s", name), func(w http.ResponseWriter, r *http.Request) {
		s.serveStatusWrapper(name, statusPublisher, w, r)
	}); err != nil {
		return fmt.Errorf("server: failed to add status publisher '%s'; err: %v", name, err)
	}
	return nil
}

// Start starts the server's listening thread (separate goroutine).
func (s *DefaultContext) Start() error {
	go func() {
		if err := s.listen(); err != nil {
			log.WithField("error", err).Error("Server routine exited with error")
			return
		}
	}()
	return nil
}

// Config returns the system config instance referenced by this server
func (s *DefaultContext) Config() *exchange.SystemConfig {
	return s.config
}

// Memory returns the metric memory instance referenced by this server
func (s *DefaultContext) Memory() *exchange.MetricMemory {
	return s.memory
}

// setup performs wiring of http router for the server part.
func (s *DefaultContext) setup() error {
	log.Debug("Setting up http server")
	s.driver = newHTTPDriver()
	if err := s.driver.AddRoute([]string{"POST"}, "/stats/container/", s.containerStats); err != nil {
		return fmt.Errorf("server: failed to add route for conatiner stats; err: %v", err)
	}
	return s.AddStatusPublisher("server", func() interface{} {
		s.stats.RLock()
		defer s.stats.RUnlock()
		statsCopy := *s.stats
		return statsCopy
	})
}

// listen runs a blocking call to http server's listening function.
func (s *DefaultContext) listen() error {
	listenAddr := fmt.Sprintf("%s:%d", s.config.ServerAddr, s.config.ServerPort)
	log.WithField("address", listenAddr).Info("Starting server")
	return s.driver.ListenAndServe(listenAddr)
}

// containerStats delivers diagnostic indicators on work performed by
// server part.
func (s *DefaultContext) containerStats(w http.ResponseWriter, r *http.Request) {
	log.Info("/stats/container was invoked")
	buffer := &bytes.Buffer{}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, maxBodySizeBytes))
	if err != nil {
		log.WithField("error", err).Error("Failed to read request body")
		s.reportError(w, err, http.StatusInternalServerError)
		return
	}
	var statsJSON map[string]interface{}
	if err := serverJSONCodec.Unmarshal(body, &statsJSON); err != nil {
		log.WithField("error", err).Error("Failed to decode request")
		s.reportError(w, err, http.StatusInternalServerError)
		return
	}
	var stats exchange.StatsRequest
	serverJSONCodec.Unmarshal(body, &stats)
	if _, gotStart := statsJSON["start"]; !gotStart {
		stats.Start = time.Time{}
	}
	if _, gotEnd := statsJSON["end"]; !gotEnd {
		stats.End = time.Now()
	}

	res := s.buildStatsResponse(&stats)

	if err := serverJSONCodec.Encode(buffer, res); err != nil {
		log.WithField("error", err).Error("Failed to encode response")
		s.reportError(w, err, http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, buffer); err != nil {
		log.WithField("error", err).Error("Failed to deliver response")
		s.reportError(w, err, http.StatusInternalServerError)
	}
}

// serveStatusWrapper invokes and publishes status object provided by
// statusPublisher function.
func (s *DefaultContext) serveStatusWrapper(name string, statusPublisher StatusPublisherFunc, w http.ResponseWriter, r *http.Request) {
	log.WithField("publisher_name", name).Info("serving status from registered publisher")
	statusObject := statusPublisher()
	buffer := &bytes.Buffer{}
	if err := serverJSONCodec.Encode(buffer, statusObject); err != nil {
		log.WithFields(logrus.Fields{"publisher_name": name, "error": err}).Error("failed to encode status from registered publisher")
		s.reportError(w, err, http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, buffer)
}

func (s *DefaultContext) reportError(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statusCode)
	if err := serverJSONCodec.Encode(w, err); err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}
}

// copyForUpdate builds a copy of ContainerInfo structure that can be
// processed independently of data in metric memory. Some internal fields are
// copied to avoid overwriting metric memory.
func copyForUpdate(container *cadv.ContainerInfo) *cadv.ContainerInfo {
	res := *container
	res.Stats = append(make([]*cadv.ContainerStats, 0, len(res.Stats)), res.Stats...)
	return &res
}

// buildStatsResponse builds a response to request for container
// stats. This function engages following locks:
// - RLock on metric storage (ServerContext.memory),
// - Lock on server stats (ServerContext.stats).
func (s *DefaultContext) buildStatsResponse(request *exchange.StatsRequest) interface{} {
	s.memory.RLock()
	defer s.memory.RUnlock()
	s.stats.Lock()
	defer s.stats.Unlock()
	res := map[string]*cadv.ContainerInfo{}
	statsTx := 0
	statsDd := 0
	for name, info := range s.memory.ContainerMap {
		info2 := copyForUpdate(info)
		sortedStats := statsListType(info2.Stats)
		sort.Sort(sortedStats)
		filteredStats := make(statsListType, 0, len(sortedStats))
		for i, statsItem := range sortedStats {
			if statsItem.Timestamp.Before(request.Start) ||
				statsItem.Timestamp.After(request.End) {
				statsDd++
				continue
			}
			filteredStats = append(filteredStats, statsItem)
			statsTx++
			if request.NumStats > 0 && len(filteredStats) >= request.NumStats {
				statsDd += (len(sortedStats) - i - 1)
				break
			}
		}
		info2.Stats = filteredStats
		res[name] = info2
	}
	// update the statistics
	if statsDd > s.stats.StatsDdMax {
		s.stats.StatsDdMax = statsDd
	}
	s.stats.StatsDdLast = statsDd
	s.stats.StatsDdTotal += statsDd
	if statsTx > s.stats.StatsTxMax {
		s.stats.StatsTxMax = statsTx
	}
	s.stats.StatsTxLast = statsTx
	s.stats.StatsTxTotal += statsTx

	return res
}
