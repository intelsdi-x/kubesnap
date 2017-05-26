package main

import (
	"fmt"
	"encoding/json"
	"os"
	"io/ioutil"
	"bufio"
	"strings"
	"sort"
	"path/filepath"
)

type Paths []string
func (p Paths) Len() int {
	return len(p)
}

func (p Paths) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Paths) Less(i, j int) bool {
	l := strings.Split(p[i], "/")
	r := strings.Split(p[j], "/")
	for k := 0; k < len(l) && k < len(r); k++ {
		if diff := strings.Compare(l[k], r[k]); diff != 0 {
			return diff < 0
		}
	}
	return len(l) < len(r)
}

func main() {
	dontpanic := func(err error) {
		if err != nil {
			// okay, panic now
			panic(err)
		}
	}
	allb, err := ioutil.ReadAll(bufio.NewReader(os.Stdin))
	dontpanic(err)
	type NameValue struct {
		Path string
		Value interface{}
	}
	parseMetricList := func(data string) []NameValue {
		metricList := []NameValue {}
		scanner := bufio.NewScanner(strings.NewReader(data))

		for scanner.Scan() {
			//fmt.Println(scanner.Text())
			txt := scanner.Text()
			fields := strings.Split(txt, "|")
			name, values := fields[len(fields) -2], fields[len(fields) -1]
			var value interface{}
			decoder := json.NewDecoder(strings.NewReader(values))
			decoder.UseNumber()
			decoder.Decode(&value)
			metricList = append(metricList, NameValue{name, value})
		}
		return metricList
	}
	rebuildObjectFromStats := func(metrics []NameValue) map[string]interface{} {
		resObj := map[string]interface{} {}
		var mkdir func([]string, map[string]interface{}) map[string]interface{}
		mkdir = func(path []string, target map[string]interface{}) map[string]interface{} {
			child, haveChild := target[path[0]]
			if !haveChild {
				child = map[string]interface{}{}
				target[path[0]] = child
			}
			if len(path) > 1 {
				return mkdir(path[1:], child.(map[string]interface{}))
			}
			return child.(map[string]interface{})
		}
		snapshot := map[string]interface{} {}
		for _, nv := range metrics {
			snapshot[nv.Path] = nv.Value
		}
		// extract now unique list of paths and sort them
		paths := []string {}
		for k, _ := range snapshot {
			paths = append(paths, k)
		}
		sorted := Paths(paths)
		sort.Sort(sorted)

		for _, p := range paths {
			path := strings.Split(filepath.Dir(p), "/")
			leaf := filepath.Base(p)
			value := snapshot[p]
			target := mkdir(path, resObj)
			target[leaf] = value
		}
		return resObj
	}
	metrics := parseMetricList(string(allb))
	resObj := rebuildObjectFromStats(metrics)
	allb, err = json.MarshalIndent(resObj, "", "  ")
	dontpanic(err)
	fmt.Println(string(allb))
	return
}

