// +build small medium large

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

// Package mox delivers a functionality of building mocks for objects
package mox

import (
	"sync"
)

// CallMock is a very simple mock-support, focused on mocking calls and their
// effects
type CallMock struct {
	lock         sync.RWMutex
	Calls        [][]interface{}
	Interceptors []InterceptorFunc
}

// Results is a type returned by a mocking call to CallMock
type Results []interface{}

// InterceptorFunc is a type for functions that get invoked to intercept
// execution of mock functions.
// Interceptor may fill results instance and indicate in return if it worked.
type InterceptorFunc func(funcName string, entry []interface{}, result *Results) (responded bool)

// AddInterceptor adds function able to respond to mock calls, in front of other
// interceptors.
// First interceptor to return true will be the last one invoked.
func (c *CallMock) AddInterceptor(interceptor InterceptorFunc) {
	c.Interceptors = append(c.Interceptors, nil)
	copy(c.Interceptors[1:], c.Interceptors[0:])
	c.Interceptors[0] = interceptor
}

func (c *CallMock) Called(funcName string, numResults int, args ...interface{}) (result Results) {
	var entry []interface{}
	func() {
		c.lock.Lock()
		defer c.lock.Unlock()
		entry = append(append([]interface{}{}, funcName), args...)
		c.Calls = append(c.Calls, entry)
	}()
	result = make([]interface{}, numResults)
	for _, interceptor := range c.Interceptors {
		if interceptor(funcName, entry, &result) {
			break
		}
	}
	return result
}

// GetCallsOf returns invocation entries per each call of given function; each
// entry represents invocation arguments, with funcName stored at index 0.
func (c *CallMock) GetCallsOf(funcName string) (res [][]interface{}) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for _, entry := range c.Calls {
		if ckFuncName := entry[0].(string); ckFuncName == funcName {
			res = append(res, entry)
		}
	}
	return res
}

// GetAllCalled returns list of called function names without duplicates, in order
// of first appearance
func (c *CallMock) GetAllCalled() (funcNames []string) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	m := map[string]bool{}
	for _, entry := range c.Calls {
		funcName := entry[0].(string)
		if _, gotIt := m[funcName]; !gotIt {
			funcNames = append(funcNames, funcName)
			m[funcName] = true
		}
	}
	return funcNames
}

func (r Results) Error(index int) (res error) {
	if r[0] == nil {
		return res
	}
	return r[0].(error)
}

func (r Results) Int(index int) (res int) {
	if r[0] == nil {
		return res
	}
	return r[0].(int)
}
