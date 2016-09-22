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

package exchange

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewMetricMemory(t *testing.T) {
	Convey("When NewMetricMemory is invoked", t, func() {
		memory := NewMetricMemory()
		Convey("it should return valid metric memory instance", func() {
			So(memory, ShouldNotBeNil)
			Convey("with all fields initialized to non-nil values", func() {
				So(memory.ContainerMap, ShouldNotBeNil)
				So(memory.PendingMetrics, ShouldNotBeNil)
			})
		})
	})
}

func TestNewSystemConfig(t *testing.T) {
	Convey("When NewSystemConfig is invoked", t, func() {
		config := NewSystemConfig()
		Convey("it should return valid system config instance", func() {
			So(config, ShouldNotBeNil)
		})
	})
}
