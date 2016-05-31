package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

var stressProc *os.Process

func LoadGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Current load is ", currentLoad, " CPU(s)\n")
	//if err := json.NewEncoder(w).Encode(currentLoad); err != nil {
	//	panic(err)
	//}
}

func LoadSet(w http.ResponseWriter, r *http.Request) {
	var load Load
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1024))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(body, &load); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	c := load.Load
	// tear down existing stress process group
	if stressProc != nil {
		pgid, err := syscall.Getpgid(stressProc.Pid)
		if err == nil {
			syscall.Kill(-pgid, 15)
		}
		stressProc = nil
	}
	// create new stress process
	cmd := exec.Command("/usr/bin/stress", "-c", strconv.Itoa(c))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Start()
	stressProc = cmd.Process
	currentLoad.Load = c

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, "Request Accepted!\n")
}
