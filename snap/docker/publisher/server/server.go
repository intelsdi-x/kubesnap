package server

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"

	"encoding/json"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"

	//"marcintao/goworks/api/exchange"
	"github.com/intelsdi-x/kubesnap/snap/docker/publisher/exchange"
	"sync"
)

var logger *log.Logger
var once sync.Once

func EnsureStarted(state *exchange.InnerState) {
	once.Do(func() {
		go ServerFunc(state)
	})
}

func ServerFunc(state *exchange.InnerState) {
	logger = log.New()
	logger.Println("Hey! HOWAbout a server?")
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", Index)
	router.HandleFunc("/todos", TodoIndex)
	router.HandleFunc("/todos/{todoId}", TodoShow)
	//router.Methods("POST").Path("/stats/container").HandlerFunc(Stats)
	router.Methods("POST").Path("/stats/container").HandlerFunc(wrapper(state, Stats))
	log.Fatal(http.ListenAndServe(":43080", router))
}

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Welcome!")
}

func TodoIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Todo Index!")
}

func TodoShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	todoId := vars["todoId"]
	fmt.Fprintln(w, "Todo show:", todoId)
}

func Stats(state *exchange.InnerState, w http.ResponseWriter, r *http.Request) {
	//var todo Todo
	_, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	logger.Println("Hey! IAMHere!")
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
	//count := 0
	//names := map[string]string{}
	//func() {
	//	state.RLock()
	//	defer state.RUnlock()
	//	count = state.MetricsReceived
	//	for k, _ := range state.RecentMetrics {
	//		names[k] = "@"
	//	}
	//}()
	//data := ""
	//{
	//	jsonData, _ := json.MarshalIndent(stats, "  ", "- ")
	//	data = string(jsonData)
	//}
	//res := exchange.Status{"Yes Sir, I'm okay Sir", &stats, count, names}
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
