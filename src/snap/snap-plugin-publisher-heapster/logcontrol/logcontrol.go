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

// Package logcontrol offers utilities for managing hierarchy of loggers shared
// between multiple subsystems.
package logcontrol

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
)

// LogHandle is a logger that can be identified and have its threshold level
// controlled.
type LogHandle interface {
	// LogAt returns string identifying the logger
	LogAt() string
	// SetLevel sets logger threshold level
	SetLevel(level int)
	// Level returns logger threshold level
	GetLevel() int
}

// LogrusHandle implements LogHandle on top of logrus.Entry
type LogrusHandle logrus.Entry

// LogControl is a poor man's approach to hierarchical control
// of application verbosity. Every logger has to be wired with call to
// WireLogger() then verbosity may be selectively turned on with calls to
// SetLevel().
type LogControl struct {
	sync.Mutex
	loggers  []LogHandle
	updaters []func(log LogHandle)
}

// WireLogger registers logger for verbosity control. Logger
// to be wired should have data field 'at' set to some sort of qualified name.
// If some levels were already turned on with call to SetLevel() then all
// rules will be evaluated on new logger.
func (l *LogControl) WireLogger(log LogHandle) {
	l.Lock()
	defer l.Unlock()
	l.loggers = append(l.loggers, log)
	l.updateLogger(log)
}

// SetLevel registers and runs a rule to control wired loggers.
// Every logger that matches the prefix in logAt parameter will have
// level set to given one. Every new wired logger will also have this rule
// evaluated.
func (l *LogControl) SetLevel(logAt string, level int) {
	l.Lock()
	defer l.Unlock()
	l.updaters = append(l.updaters, func(log LogHandle) {
		if ckLogAt := log.LogAt(); strings.HasPrefix(ckLogAt, logAt) {
			log.SetLevel(level)
		}
	})
	for _, logger := range l.loggers {
		l.updateLogger(logger)
	}
}

// updateLogger runs all available updaters against given logger instance.
func (l *LogControl) updateLogger(log LogHandle) {
	for _, updater := range l.updaters {
		updater(log)
	}
}

// LogAt returns string identifying the logger reading fields of the wrapped
// logger
func (l *LogrusHandle) LogAt() string {
	if ckLogAt, haveLogAt := l.Data["at"]; haveLogAt {
		return fmt.Sprint(ckLogAt)
	}
	return ""
}

// SetLevel sets logger threshold level in the wrapped logger
func (l *LogrusHandle) SetLevel(level int) {
	l.Logger.Level = logrus.Level(level)
}

// GetLevel returns logger threshold level of the wrapped logger
func (l *LogrusHandle) GetLevel() int {
	return int(l.Logger.Level)
}
