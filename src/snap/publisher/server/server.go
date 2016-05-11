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

package server

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"

	"encoding/json"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"

	"github.com/intelsdi-x/kubesnap/src/snap/publisher/exchange"
	"sync"
)

var logger *log.Logger
var once sync.Once

func EnsureStarted(state *exchange.InnerState, port int) {
	once.Do(func() {
		go ServerFunc(state, port)
	})
}

func ServerFunc(state *exchange.InnerState, port int) {
	logger = log.New()
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("POST").Path("/stats/container").HandlerFunc(wrapper(state, Stats))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}

func Stats(state *exchange.InnerState, w http.ResponseWriter, r *http.Request) {
	_, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	//var stats exchange.StatsRequest
	//if err := json.Unmarshal(body, &stats); err != nil {
	//	logger.Errorf("got this error: %v \n", err)
	//	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	//	w.WriteHeader(422) // unprocessable entity
	//	if err := json.NewEncoder(w).Encode(err); err != nil {
	//		panic(err)
	//	}
	//	return
	//}

	//t := RepoCreateTodo(todo)
	state.RLock()
	defer state.RUnlock()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	res := state.DockerStorage
	if err := json.NewEncoder(w).Encode(res); err != nil {
		panic(err)
	}
}

func wrapper(state *exchange.InnerState, fu func(*exchange.InnerState, http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		Stats(state, w, r)
	}
}
