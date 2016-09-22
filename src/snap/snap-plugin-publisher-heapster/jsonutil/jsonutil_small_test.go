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

// Package jsonutil offers utilities for building and traversing composite
// objects reconstructed from metrics.
package jsonutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// extractorFunc implements conditional extraction of nodes from JSON object,
// summarizing them in a string;
// returns true if item was extracted successfully.
type extractorFunc func(path string, info os.FileInfo) (bool, string)

// walkInspector implements a visitor for nodes in JSON object, extracting
// string summary for visited nodes.
type walkInspector struct {
	report    []string
	extractor extractorFunc
}

// dummyMetrics is a test type implementing MetricList; wraps a list of paths
type dummyMetrics []string

// dummyMetric is a test type implementing MetricHandle
type dummyMetric struct {
	item string
}

var (
	sampleJsonSrc = `{
		"uno":{
			"1":"dos",
			"2":42
		},
		"foo":"bar",
		"hop": {
			"lol": {
				"bonk":123
			},
			"rotfl": false
		}
	}
	`
	sampleJson    map[string]interface{}
	sampleMetrics MetricList
)

func init() {
	json.Unmarshal([]byte(sampleJsonSrc), &sampleJson)
	tmp := dummyMetrics(strings.Split("/uno/1,/uno/2,/foo,/hop/lol/bonk,/hop/rotfl", ","))
	sampleMetrics = &tmp
}

func makePaths(src string) (res objectPaths) {
	for _, ln := range strings.Split(src, "\n") {
		if ln == "" {
			continue
		}
		split := strings.Split(ln, "/")
		res = append(res, ObjectPath{Literal: ln, Split: split})
	}
	return res
}

func TestObjectPaths_Less(t *testing.T) {
	p := makePaths(`b/b/c
b/b/d
b/d
e/f
a/b/c`)
	Convey("Given a selection of ObjectPaths", t, func() {
		Convey("all of them should compare correctly to each other", func() {
			So(p.Less(0, 1), ShouldBeTrue)
			So(p.Less(1, 0), ShouldBeFalse)
			So(p.Less(0, 2), ShouldBeFalse)
			So(p.Less(2, 0), ShouldBeTrue)
			So(p.Less(0, 3), ShouldBeFalse)
			So(p.Less(3, 0), ShouldBeTrue)
			So(p.Less(0, 4), ShouldBeFalse)
			So(p.Less(4, 0), ShouldBeTrue)
			So(p.Less(0, 0), ShouldBeFalse)
			So(p.Less(1, 1), ShouldBeFalse)
			So(p.Less(2, 2), ShouldBeFalse)
			So(p.Less(3, 3), ShouldBeFalse)
			So(p.Less(4, 4), ShouldBeFalse)
		})
	})
}

func TestNewObjWalker(t *testing.T) {
	Convey("Given a nested map object", t, func() {
		Convey("NewObjWalker should not fail", func() {
			So(func() {
				NewObjWalker(sampleJson)
			}, ShouldNotPanic)
			Convey("but it should return a non-nil JSONWalker instance", func() {
				walker := NewObjWalker(sampleJson)
				So(walker, ShouldNotBeNil)
			})
		})
	})
}

func newWalkInspectorForPaths() *walkInspector {
	return newWalkInspector(func(path string, info os.FileInfo) (bool, string) {
		return true, path
	})
}

func newWalkInspector(extractor extractorFunc) *walkInspector {
	inspector := walkInspector{extractor: extractor}
	return &inspector
}

func (w *walkInspector) VisitPath(path string, info os.FileInfo, err error) error {
	if err == nil {
		if valid, entry := w.extractor(path, info); valid {
			w.report = append(w.report, entry)
		}
	}
	return nil
}

func (w walkInspector) Len() int {
	return len(w.report)
}

func (w walkInspector) Swap(i, j int) {
	w.report[i], w.report[j] = w.report[j], w.report[i]
}

func (w walkInspector) Less(i, j int) bool {
	l := NewObjectPath(w.report[i])
	r := NewObjectPath(w.report[j])
	return l.Diff(*r) <= 0
}

func (d *dummyMetrics) Len() int {
	return len(*d)
}

func (d *dummyMetrics) Item(index int) MetricHandle {
	return &dummyMetric{item: (*d)[index]}
}

func (d *dummyMetric) Path() ObjectPath {
	return *NewObjectPath(d.item)
}

func (d *dummyMetric) Data() interface{} {
	return d.item
}

func (d *dummyMetric) RawMetric() interface{} {
	return d.item
}

func extractPathsFromJSON(obj interface{}) []string {
	walker := NewObjWalker(obj)
	inspector := newWalkInspectorForPaths()
	walker.Walk("/", inspector.VisitPath)
	sort.Sort(inspector)
	return inspector.report
}

func rmFromJSON(obj interface{}, path string) error {
	parent := filepath.Dir(path)
	base := filepath.Base(path)
	walker := NewObjWalker(obj)
	parentObj, err := walker.Seek(parent)
	if err != nil {
		return err
	}
	parentMap := parentObj.(map[string]interface{})
	delete(parentMap, base)
	return nil
}

func TestNewObjectPath(t *testing.T) {
	Convey("Using NewObjectPath", t, func() {
		Convey("with a /-delimited path", func() {
			literal := "/usr/local/lib"
			Convey("should result in a correct mapping to an objectPath segments", func() {
				objPath := NewObjectPath(literal)
				So(objPath.Literal, ShouldEqual, literal)
				So(objPath.Split, ShouldResemble, []string{"usr", "local", "lib"})
			})
		})
	})
}

func TestJSONWalker_Walk(t *testing.T) {
	Convey("Given a JSON object to Walk", t, func() {
		walker := NewObjWalker(sampleJson)
		Convey("starting from a root path", func() {
			path := "/"
			Convey("JSONWalker should return no error", func() {
				inspector := newWalkInspectorForPaths()
				err := walker.Walk(path, inspector.VisitPath)
				Convey("and all paths in structure should be visited", func() {
					So(err, ShouldBeNil)
					sort.Sort(inspector)
					So(inspector.report, ShouldResemble, strings.Split(
						"/,/foo,/hop,/uno,/hop/lol,/hop/rotfl,/uno/1,/uno/2,/hop/lol/bonk", ","))
				})
			})
		})
		Convey("starting from an invalid (missing) path", func() {
			invalidPath := "/TRES"
			Convey("a JSONWalker should return error indicating that path as not found", func() {
				inspector := newWalkInspectorForPaths()
				err := walker.Walk(invalidPath, inspector.VisitPath)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "/TRES")
			})
		})
		Convey("JSONWalker should report names for visited nodes", func() {
			inspector := newWalkInspector(func(path string, info os.FileInfo) (bool, string) {
				return true, info.Name()
			})
			_ = walker.Walk("/", inspector.VisitPath)
			sort.Sort(inspector)
			Convey("so all names should be encountered during Walk", func() {
				So(inspector.report, ShouldResemble, strings.Split(
					"/,1,2,bonk,foo,hop,lol,rotfl,uno", ","))
			})
		})
		Convey("JSONWalker should indicate which nodes are dir(non-leaf) ones", func() {
			inspector := newWalkInspector(func(path string, info os.FileInfo) (bool, string) {
				if info.IsDir() {
					return true, path
				}
				return false, ""
			})
			_ = walker.Walk("/", inspector.VisitPath)
			sort.Sort(inspector)
			Convey("so it should be possible to select only dir nodes during Walk", func() {
				So(inspector.report, ShouldResemble, strings.Split(
					"/,/hop,/uno,/hop/lol", ","))
			})
		})
	})
}

func TestJSONWalker_Seek(t *testing.T) {
	Convey("Asked to seek into JSON object", t, func() {
		walker := NewObjWalker(sampleJson)
		Convey("JSONWalker should not return any error", func() {
			subObj, err := walker.Seek("/hop")
			So(subObj, ShouldNotBeNil)
			So(err, ShouldBeNil)
			Convey("and the resulting object should map to correct subobject", func() {
				paths := extractPathsFromJSON(subObj)
				So(paths, ShouldResemble, strings.Split(
					"/,/lol,/rotfl,/lol/bonk", ","))
			})
		})
		Convey("along an incorrect path (missing target)", func() {
			targetPath := "/TRES"
			Convey("JSONWalker should return error indicating path was not found", func() {
				_, err := walker.Seek(targetPath)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "/TRES")
			})
		})
	})
}

func TestPruneEmptySubtrees(t *testing.T) {
	Convey("Having a JSON object", t, func() {
		var another map[string]interface{}
		_ = json.Unmarshal([]byte(sampleJsonSrc), &another)
		Convey("which has no empty nodes, PruneEmptySubtrees should not modify it", func() {
			paths := extractPathsFromJSON(another)
			So(paths, ShouldResemble, strings.Split(
				"/,/foo,/hop,/uno,/hop/lol,/hop/rotfl,/uno/1,/uno/2,/hop/lol/bonk", ","))
		})
		Convey("which has some empty subtrees", func() {
			for _, path := range strings.Split("/hop/lol/bonk,/hop/rotfl,/uno/1,/uno/2", ",") {
				rmFromJSON(another, path)
			}
			Convey("PruneEmptySubtrees should remove all of them up to the root", func() {
				PruneEmptySubtrees(another)
				paths := extractPathsFromJSON(another)
				So(paths, ShouldResemble, strings.Split(
					"/,/foo", ","))
			})
		})
	})
}

func TestRebuildObjectFromMetrics(t *testing.T) {
	Convey("Given a list of metrics", t, func() {
		Convey("RebuildObjectFromMetrics should not fail", func() {
			So(func() {
				_ = RebuildObjectFromMetrics(sampleMetrics, func(_ []string, metric MetricHandle) interface{} {
					return metric.Data()
				})
			}, ShouldNotPanic)
			Convey("returned object should not be nil", func() {
				mtree := RebuildObjectFromMetrics(sampleMetrics, func(_ []string, metric MetricHandle) interface{} {
					return metric.Data()
				})
				So(mtree, ShouldNotBeNil)
				Convey("and its structure should reflect namespaces of all metrics", func() {
					paths := extractPathsFromJSON(mtree)
					So(paths, ShouldResemble, strings.Split(
						"/,/foo,/hop,/uno,/hop/lol,/hop/rotfl,/uno/1,/uno/2,/hop/lol/bonk", ","))
				})
			})
		})
	})
}
