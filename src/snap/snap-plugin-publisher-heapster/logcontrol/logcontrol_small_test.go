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
	"strings"
	"testing"

	"github.com/Sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
)

type DummyLog struct {
	level int
	logAt string
}

func NewDummyLog(logAt string, level ...int) *DummyLog {
	if len(level) >= 1 {
		return &DummyLog{logAt: logAt, level: level[0]}
	}
	return &DummyLog{logAt: logAt, level: 0}
}

func (l *DummyLog) LogAt() string {
	return l.logAt
}

func (l *DummyLog) SetLevel(level int) {
	l.level = level
}

func (l *DummyLog) GetLevel() int {
	return l.level
}

func TestLogControl_WireLogger(t *testing.T) {
	Convey("LogControl should never fail", t, func() {
		logControl := &LogControl{}
		Convey("when WireLogger is called once", func() {
			So(func() {
				logControl.WireLogger(NewDummyLog("uno"))
			}, ShouldNotPanic)
		})
		Convey("or any number of times (3)", func() {
			So(func() {
				logControl.WireLogger(NewDummyLog("uno"))
				logControl.WireLogger(NewDummyLog("dos"))
				logControl.WireLogger(NewDummyLog("tres"))
			}, ShouldNotPanic)
		})
	})
}

func TestLogControl_SetLevel(t *testing.T) {
	Convey("When multiple wired loggers form naming hierarchy", t, func() {
		logControl := &LogControl{}
		var loglist []*DummyLog

		forLevel := func(level int) []string {
			var res []string
			for _, logger := range loglist {
				if logger.GetLevel() == level {
					res = append(res, logger.LogAt())
				}
			}
			return res
		}

		loglist = append(loglist,
			NewDummyLog("/uno", 0),
			NewDummyLog("/uno/foo", 0),
			NewDummyLog("/uno/foo/yes", 0),
			NewDummyLog("/uno/bar", 0),
			NewDummyLog("/dos", 0))
		for _, logger := range loglist {
			logControl.WireLogger(logger)
		}
		Convey("LogControl should SetLevel on all related loggers", func() {
			logControl.SetLevel("/", 3)
			So(forLevel(3), ShouldResemble, strings.Split("/uno,/uno/foo,/uno/foo/yes,/uno/bar,/dos", ","))
			So(forLevel(0), ShouldBeEmpty)
		})
		Convey("LogControl should NOT set level on unrelated loggers", func() {
			logControl.SetLevel("/uno/foo", 2)
			So(forLevel(2), ShouldResemble, strings.Split("/uno/foo,/uno/foo/yes", ","))
			So(forLevel(0), ShouldResemble, strings.Split("/uno,/uno/bar,/dos", ","))
		})
		Convey("LogControl should NOT touch any level if no loggers match", func() {
			logControl.SetLevel("/TRES", 1)
			So(forLevel(1), ShouldBeEmpty)
			So(forLevel(0), ShouldResemble, strings.Split("/uno,/uno/foo,/uno/foo/yes,/uno/bar,/dos", ","))
		})
	})
}

func TestLogrusHandle_GetLevel(t *testing.T) {
	Convey("Instances of LogrusHandle", t, func() {
		log := logrus.New().WithField("at", "/publisher")
		log.Logger.Level = logrus.WarnLevel
		handle := (*LogrusHandle)(log)
		Convey("should correctly report the level of underlying logger", func() {
			So(handle.GetLevel(), ShouldEqual, int(logrus.WarnLevel))
			log.Logger.Level = logrus.DebugLevel
			So(handle.GetLevel(), ShouldEqual, int(logrus.DebugLevel))
		})
	})
}

func TestLogrusHandle_SetLevel(t *testing.T) {
	Convey("Instances of LogrusHandle", t, func() {
		log := logrus.New().WithField("at", "/publisher")
		log.Logger.Level = logrus.WarnLevel
		handle := (*LogrusHandle)(log)
		handle.SetLevel(int(logrus.DebugLevel))
		Convey("should correctly update the level of underlying logger", func() {
			So(log.Logger.Level, ShouldEqual, logrus.DebugLevel)
			handle.SetLevel(int(logrus.ErrorLevel))
			So(log.Logger.Level, ShouldEqual, logrus.ErrorLevel)
		})
	})
}

func TestLogrusHandle_GetAndSetLevel(t *testing.T) {
	Convey("Instances of LogrusHandle", t, func() {
		log := logrus.New().WithField("at", "/publisher")
		log.Level = logrus.WarnLevel
		handle := (*LogrusHandle)(log)
		handle.SetLevel(int(logrus.PanicLevel))
		Convey("should consistently report the level set on them", func() {
			So(handle.GetLevel(), ShouldEqual, int(logrus.PanicLevel))
			handle.SetLevel(int(logrus.DebugLevel))
			So(handle.GetLevel(), ShouldEqual, int(logrus.DebugLevel))
		})
	})
}

func TestLogrusHandle_LogAt(t *testing.T) {
	Convey("Instances of LogrusHandle", t, func() {
		Convey("with an 'at' field (logAt) set", func() {
			log := logrus.New().WithField("at", "/work")
			handle := (*LogrusHandle)(log)
			Convey("should correctly report their 'at' field", func() {
				So(handle.LogAt(), ShouldEqual, "/work")
			})
		})
		Convey("with UNSET 'at' field (logAt) set", func() {
			log := logrus.New().WithFields(logrus.Fields{})
			handle := (*LogrusHandle)(log)
			Convey("should correctly report their 'at' field to be empty", func() {
				So(handle.LogAt(), ShouldEqual, "")
			})
		})
	})
}
