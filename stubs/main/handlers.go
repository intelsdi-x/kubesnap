package main

import (
	"fmt"
	"net/http"
	"io/ioutil"
)

func ContainerStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	ret, _ := ioutil.ReadFile("data_stub")
	fmt.Fprint(w, string(ret))
}
