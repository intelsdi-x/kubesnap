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

package jsonutil

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type JsonWalker struct {
	fs interface{}
}

type dummyFileInfo struct {
	name  string
	isDir bool
	sys   interface{}
}

var NotFound = errors.New("Path not found")

func (i *dummyFileInfo) Name() string {
	return i.name
}

func (i *dummyFileInfo) IsDir() bool {
	return i.isDir
}

func (i *dummyFileInfo) Size() int64 {
	return 0
}

func (i *dummyFileInfo) Mode() os.FileMode {
	if i.isDir {
		return os.ModeDir
	}
	return 0
}

func (i *dummyFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (i *dummyFileInfo) Sys() interface{} {
	return i.sys
}

// NewJsonWalker returns an iterator over contents of the json source.
//
// Iterator provided by NewJsonWalker supports the same semantics as standard
// `filepath.Walk`.
func NewJsonWalker(jsonSource string) (*JsonWalker, error) {
	walker := new(JsonWalker)
	var root interface{}
	if err := json.Unmarshal([]byte(jsonSource), &(root)); err != nil {
		return nil, err
	}
	walker.fs = root
	return walker, nil
}

// NewObjWalker returns an iterator over contents of a composite object.
//
// Iterator provided by NewObjWalker supports the same semantics as standard
// `filepath.Walk`.
//
// The only nodes that can be inspected in depth are generic map and generic
// array, namely `map[string]interface{}` and `[]interface{}`.
func NewObjWalker(root interface{}) *JsonWalker {
	walker := new(JsonWalker)
	walker.fs = root
	return walker
}

// Walk implements similar behavior to `filepath.Walk`.
func (w *JsonWalker) Walk(path string, walkFunc filepath.WalkFunc) error {
	node, err := seek(w.fs, path)
	if err != nil {
		return err
	}
	walk(node, path, walkFunc)
	return nil
}

// Seek walks through walker's target object until specific path is reached,
// returning handle to data at that location.
//
// Seek tries its best to find value at given path. Failure to reach the path
// is indicated with error value of `NotFound`.
func (w *JsonWalker) Seek(seekPath string) (interface{}, error) {
	return seek(w.fs, seekPath)
}

func seek(root interface{}, seekPath string) (interface{}, error) {
	var result interface{}
	resultSet := false
	walk(root, "/", func(path string, info os.FileInfo, _ error) error {
		if result != nil {
			return filepath.SkipDir
		} else if path == seekPath {
			result = info.Sys()
			resultSet = true
			return filepath.SkipDir
		}
		return nil
	})
	if resultSet {
		return result, nil
	}
	return nil, NotFound
}

func basename(path string) string {
	base := filepath.Base(path)
	if base == "." {
		return "/"
	}
	return base
}

func walk(node interface{}, path string, walkFunc filepath.WalkFunc) error {
	var err error
	if dirNode, isDir := node.(map[string]interface{}); isDir {
		err = walkFunc(path, &dummyFileInfo{basename(path), true, dirNode}, nil)
		if err == filepath.SkipDir {
			return nil
		}
		for k, subNode := range dirNode {
			err = walk(subNode, filepath.Join(path, k), walkFunc)
			if err == filepath.SkipDir {
				return nil
			}
		}
	} else if dirNode, isDir := node.([]interface{}); isDir {
		err = walkFunc(path, &dummyFileInfo{basename(path), true, dirNode}, nil)
		if err == filepath.SkipDir {
			return nil
		}
		for k, subNode := range dirNode {
			err = walk(subNode, filepath.Join(path, strconv.Itoa(k)), walkFunc)
			if err == filepath.SkipDir {
				return nil
			}
		}
	} else {
		err = walkFunc(path, &dummyFileInfo{basename(path), false, node}, nil)
		if err == filepath.SkipDir {
			return err
		}
	}
	return nil
}
