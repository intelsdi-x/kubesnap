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

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cadv "github.com/google/cadvisor/info/v1"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/exchange"
	"github.com/intelsdi-x/snap-plugin-publisher-heapster/mox"
	. "github.com/smartystreets/goconvey/convey"
)

type mockHTTPDriver struct {
	mox.CallMock
}

func (m *mockHTTPDriver) AddRoute(methods []string, path string, handler http.HandlerFunc) error {
	res := m.Called("AddRoute", 1, methods, path, handler)
	return res.Error(0)
}

func (m *mockHTTPDriver) ListenAndServe(serverAddr string) error {
	res := m.Called("ListenAndServe", 1, serverAddr)
	return res.Error(0)
}

// nilReader is a reader for testing that returns errors for all ops
type nilReader struct{}

func (n *nilReader) Read(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func (n *nilReader) Close() error {
	return io.ErrUnexpectedEOF
}

type mockJSONCodec struct {
	mox.CallMock
}

func (c mockJSONCodec) Unmarshal(raw []byte, dest interface{}) error {
	res := c.Called("Unmarshal", 1, raw, dest)
	return res.Error(0)
}

func (c mockJSONCodec) Encode(writer io.Writer, obj interface{}) error {
	res := c.Called("Encode", 1, writer, obj)
	return res.Error(0)
}

type mockResponseWriter struct {
	mox.CallMock
}

func (w *mockResponseWriter) Header() http.Header {
	res := w.Called("Header", 1)
	var header http.Header
	if res[0] != nil {
		header = res[0].(http.Header)
	}
	return header
}

func (w *mockResponseWriter) Write(buf []byte) (int, error) {
	res := w.Called("Write", 2, buf)
	return res.Int(0), res.Error(1)
}

func (w *mockResponseWriter) WriteHeader(code int) {
	_ = w.Called("WriteHeader", 0, code)
}

func TestNewDefaultContext(t *testing.T) {
	Convey("Working with server package", t, func() {
		Convey("creating new server context should not fail", func() {
			So(func() {
				newDefaultContext(exchange.NewSystemConfig(), exchange.NewMetricMemory())
			}, ShouldNotPanic)
			Convey("but created object should have all fields initialized", func() {
				ctx := newDefaultContext(exchange.NewSystemConfig(), exchange.NewMetricMemory())
				So(ctx, ShouldNotBeNil)
				So(ctx.Config(), ShouldNotBeNil)
				So(ctx.Memory(), ShouldNotBeNil)
			})
		})
	})
}

func TestNewHTTPDriver(t *testing.T) {
	Convey("Working with server package", t, func() {
		Convey("creating new HTTP driver should not fail", func() {
			So(func() {
				newHTTPDriver()
			}, ShouldNotPanic)
			Convey("but should deliver a non-nil instance of driver", func() {
				driver := newHTTPDriver()
				So(driver, ShouldNotBeNil)
			})
		})
	})
}

func TestNewServer(t *testing.T) {
	config := exchange.NewSystemConfig()
	memory := exchange.NewMetricMemory()
	Convey("While configuring server subsystem", t, func() {
		mockDriver := &mockHTTPDriver{}
		oldHTTPDriverCtor := newHTTPDriver
		newHTTPDriver = func() HTTPDriver {
			return mockDriver
		}
		Convey("building new server instance should not fail", func() {
			server, err := NewServer(config, memory)
			So(err, ShouldBeNil)
			So(server, ShouldNotBeNil)
		})
		Convey("building new instance should fail if server setup terminates with error", func() {
			mockDriver.AddInterceptor(func(funcName string, _ []interface{}, result *mox.Results) bool {
				if funcName == "AddRoute" {
					(*result)[0] = errors.New("AddRoute failed")
					return true
				}
				return false
			})
			_, err := NewServer(config, memory)
			So(err, ShouldNotBeNil)
		})
		Reset(func() {
			newHTTPDriver = oldHTTPDriverCtor
		})
	})
}

func TestDefaultContext_AddStatusPublisher(t *testing.T) {
	config := exchange.NewSystemConfig()
	memory := exchange.NewMetricMemory()
	Convey("While configuring the server subsystem", t, func() {
		mockDriver := &mockHTTPDriver{}
		oldHTTPDriverCtor := newHTTPDriver
		newHTTPDriver = func() HTTPDriver {
			return mockDriver
		}
		Convey("adding status publisher should not fail", func() {
			var registeredRoutes []string
			mockDriver.AddInterceptor(func(funcName string, args []interface{}, result *mox.Results) bool {
				if funcName == "AddRoute" {
					registeredRoutes = append(registeredRoutes, args[2].(string))
					return true
				}
				return false
			})
			Convey("when adding a simple route", func() {
				server, _ := NewServer(config, memory)
				err := server.AddStatusPublisher("test", func() interface{} {
					return map[string]string{"status": "ok"}
				})
				So(err, ShouldBeNil)
				Convey("and new route should be created in the server", func() {
					So(registeredRoutes, ShouldContain, "/_status/test")
				})
			})
		})
		Convey("and adding a route that fails", func() {
			mockDriver.AddInterceptor(func(funcName string, args []interface{}, result *mox.Results) bool {
				if funcName == "AddRoute" && strings.HasSuffix(args[2].(string), "/test") {
					(*result)[0] = fmt.Errorf("AddRoute failed: %s", args[2].(string))
					return true
				}
				return false
			})
			Convey("adding status publisher should fail", func() {
				server, _ := NewServer(config, memory)
				err := server.AddStatusPublisher("test", func() interface{} {
					return map[string]string{"status": "ok"}
				})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "AddRoute failed")
				So(err.Error(), ShouldContainSubstring, "/test")
			})
		})
		Reset(func() {
			newHTTPDriver = oldHTTPDriverCtor
		})
	})
}

func TestDefaultContext_Start(t *testing.T) {
	config := exchange.NewSystemConfig()
	memory := exchange.NewMetricMemory()
	Convey("While configuring the server subsystem", t, func() {
		mockDriver := &mockHTTPDriver{}
		oldHTTPDriverCtor := newHTTPDriver
		newHTTPDriver = func() HTTPDriver {
			return mockDriver
		}
		// ListenAndServe is launched async, so add notifying-interceptor
		done := make(chan bool)
		mockDriver.AddInterceptor(func(funcName string, _ []interface{}, _ *mox.Results) bool {
			if funcName == "ListenAndServe" {
				done <- true
				return true
			}
			return false
		})
		Convey("and using default parameters", func() {
			server, _ := NewServer(config, memory)
			Convey("starting the server should not fail", func() {
				err := server.Start()
				<-done
				So(err, ShouldBeNil)
				Convey("and HTTP driver should launch server main loop", func() {
					So(mockDriver.GetAllCalled(), ShouldContain, "ListenAndServe")
				})
			})
		})
		Reset(func() {
			newHTTPDriver = oldHTTPDriverCtor
		})

	})
}

func TestDefaultContext_listen(t *testing.T) {
	config := exchange.NewSystemConfig()
	memory := exchange.NewMetricMemory()
	Convey("While configuring the server subsystem", t, func() {
		mockDriver := &mockHTTPDriver{}
		oldHTTPDriverCtor := newHTTPDriver
		newHTTPDriver = func() HTTPDriver {
			return mockDriver
		}
		Convey("and using HTTP driver setup that fails", func() {
			tmp := newDefaultContext(config, memory)
			server := &tmp
			server.setup()
			mockDriver.AddInterceptor(func(funcName string, args []interface{}, result *mox.Results) bool {
				if funcName == "ListenAndServe" {
					(*result)[0] = errors.New("ListenAndServe failed")
					return false
				}
				return false
			})
			Convey("starting the server listening routine should also fail", func() {
				err := server.listen()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "ListenAndServe failed")
			})
		})
		Reset(func() {
			newHTTPDriver = oldHTTPDriverCtor
		})

	})
}

func TestDefaultContext_containerStats(t *testing.T) {
	config := exchange.NewSystemConfig()
	memory := exchange.NewMetricMemory()
	Convey("While using the server subsystem", t, func() {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("POST", "./stats/container", strings.NewReader(`{"num_stats": 1}`))
		mockDriver := &mockHTTPDriver{}
		oldHTTPDriverCtor := newHTTPDriver
		newHTTPDriver = func() HTTPDriver {
			return mockDriver
		}
		tmp := newDefaultContext(config, memory)
		server := &tmp
		server.setup()
		oldJSONCodec := serverJSONCodec
		myMockJSONCodec := mockJSONCodec{}
		Convey("and issuing request for container stats", func() {
			Convey("the handler should not fail", func() {
				server.containerStats(w, r)
				w.Flush()
				So(w.Code, ShouldEqual, http.StatusOK)
				Convey("the response should represent valid JSON", func() {
					var stats map[string]interface{}
					err := json.Unmarshal(w.Body.Bytes(), &stats)
					So(err, ShouldBeNil)
				})
			})
			Convey("when there are some container entries in the metric memory", func() {
				server.memory.ContainerMap["/"] = makeDummyContainerInfo("/")
				server.memory.ContainerMap["/foo"] = makeDummyContainerInfo("f00")
				Convey("the handler should not fail", func() {
					server.containerStats(w, r)
					w.Flush()
					So(w.Code, ShouldEqual, http.StatusOK)
					Convey("the response should reflect entries from memory", func() {
						var stats map[string]map[string]interface{}
						err := json.Unmarshal(w.Body.Bytes(), &stats)
						So(err, ShouldBeNil)
						So(stats, ShouldContainKey, "/")
						So(stats, ShouldContainKey, "/foo")
						So(stats["/foo"]["id"], ShouldEqual, "f00")
						So(stats["/foo"]["name"], ShouldEqual, "f00")
					})
				})
			})
			Convey("when reading request fails", func() {
				r.Body = &nilReader{}
				Convey("the handler should fail too", func() {
					server.containerStats(w, r)
					w.Flush()
					So(w.Code, ShouldEqual, http.StatusInternalServerError)
				})
			})
			Convey("when decoding request fails", func() {
				serverJSONCodec = &myMockJSONCodec
				myMockJSONCodec.AddInterceptor(func(funcName string, _ []interface{}, result *mox.Results) bool {
					if funcName == "Unmarshal" {
						(*result)[0] = errors.New("decoding failed")
						return true
					}
					return false
				})
				Convey("the handler should also fail", func() {
					server.containerStats(w, r)
					w.Flush()
					So(w.Code, ShouldEqual, http.StatusInternalServerError)
				})
			})
			Convey("when encoding response fails", func() {
				serverJSONCodec = &myMockJSONCodec
				myMockJSONCodec.AddInterceptor(func(funcName string, _ []interface{}, result *mox.Results) bool {
					if funcName == "Encode" {
						(*result)[0] = errors.New("encoding failed")
						return true
					}
					return false
				})
				Convey("the handler should also fail", func() {
					server.containerStats(w, r)
					w.Flush()
					So(w.Code, ShouldEqual, http.StatusInternalServerError)
				})
			})
			Convey("when storing response fails", func() {
				statusCode := 0
				headers := http.Header{}
				mockWriter := &mockResponseWriter{}
				mockWriter.AddInterceptor(func(funcName string, args []interface{}, res *mox.Results) bool {
					if funcName == "Write" {
						(*res)[1] = errors.New("Write failed")
						return true
					}
					if funcName == "Header" {
						(*res)[0] = headers
						return true
					}
					if funcName == "WriteHeader" {
						statusCode = args[1].(int)
						return true
					}
					return false
				})
				Convey("the handler should fail too", func() {
					server.containerStats(mockWriter, r)
					So(statusCode, ShouldEqual, http.StatusInternalServerError)
				})
			})
			Reset(func() {
				serverJSONCodec = oldJSONCodec
			})
		})
		Reset(func() {
			newHTTPDriver = oldHTTPDriverCtor
		})

	})
}

func TestDefaultContext_serveStatusWrapper(t *testing.T) {
	config := exchange.NewSystemConfig()
	memory := exchange.NewMetricMemory()
	Convey("While using the server subsystem", t, func() {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "./stats/container", strings.NewReader(`{"num_stats": 1}`))
		mockDriver := &mockHTTPDriver{}
		oldHTTPDriverCtor := newHTTPDriver
		newHTTPDriver = func() HTTPDriver {
			return mockDriver
		}
		tmp := newDefaultContext(config, memory)
		server := &tmp
		server.setup()
		Convey("and issuing request for status publisher", func() {
			Convey("the handler should not fail", func() {
				server.serveStatusWrapper("foo", func() interface{} {
					return map[string]string{"bar": "bonk"}
				}, w, r)
				w.Flush()
				So(w.Code, ShouldEqual, http.StatusOK)
				Convey("the response should represent valid JSON", func() {
					var output map[string]interface{}
					err := json.Unmarshal(w.Body.Bytes(), &output)
					So(err, ShouldBeNil)
					Convey("and the output should represent the published status", func() {
						So(output, ShouldContainKey, "bar")
					})
				})
			})
			Convey("while published object is not valid for JSON", func() {
				server.serveStatusWrapper("foo", func() interface{} {
					return map[float64]int{3.5: 2}
				}, w, r)
				w.Flush()
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
			})

		})
		Reset(func() {
			newHTTPDriver = oldHTTPDriverCtor
		})

	})
}

func TestDefaultContext_buildStatsResponse(t *testing.T) {
	parseDate := func(dateStr string) time.Time {
		res, _ := time.Parse("02.01.06", dateStr)
		return res
	}
	extractDays := func(c *cadv.ContainerInfo) (res []int) {
		for _, s := range c.Stats {
			res = append(res, s.Timestamp.Day())
		}
		return res
	}
	config := exchange.NewSystemConfig()
	memory := exchange.NewMetricMemory()
	Convey("While using the server subsystem", t, func() {
		mockDriver := &mockHTTPDriver{}
		oldHTTPDriverCtor := newHTTPDriver
		newHTTPDriver = func() HTTPDriver {
			return mockDriver
		}
		tmp := newDefaultContext(config, memory)
		server := &tmp
		server.setup()
		Convey("with some stats in memory between 1st and 5th July", func() {
			container := makeDummyContainerInfo("/")
			for d := 1; d <= 5; d++ {
				stats := cadv.ContainerStats{
					Timestamp: parseDate(fmt.Sprintf("%02d.%02d.%02d", d, 7, 9))}
				container.Stats = append(container.Stats, &stats)
			}
			server.memory.ContainerMap["/"] = container
			Convey("when stats are requested >= 2nd July", func() {
				request := &exchange.StatsRequest{
					Start: parseDate("02.07.09"),
					End:   parseDate("11.12.13"),
				}
				response := server.buildStatsResponse(request).(map[string]*cadv.ContainerInfo)
				Convey("correct list of dates should be present in response", func() {
					days := extractDays(response["/"])
					So(days, ShouldResemble, []int{5, 4, 3, 2})
				})
			})
			Convey("when stats are requested >= 2 and <= 4th July", func() {
				request := &exchange.StatsRequest{
					Start: parseDate("02.07.09"),
					End:   parseDate("04.07.09"),
				}
				response := server.buildStatsResponse(request).(map[string]*cadv.ContainerInfo)
				Convey("correct list of dates should be present in response", func() {
					days := extractDays(response["/"])
					So(days, ShouldResemble, []int{4, 3, 2})
				})
			})
			Convey("when 3 stats elements are requested", func() {
				request := &exchange.StatsRequest{
					Start:    parseDate("01.01.01"),
					End:      parseDate("11.12.13"),
					NumStats: 3,
				}
				response := server.buildStatsResponse(request).(map[string]*cadv.ContainerInfo)
				Convey("correct number and list of dates should be present in response", func() {
					days := extractDays(response["/"])
					So(days, ShouldResemble, []int{5, 4, 3})
				})
			})
		})
		Reset(func() {
			newHTTPDriver = oldHTTPDriverCtor
		})
	})
}

func makeDummyContainerInfo(id string) *cadv.ContainerInfo {
	res := cadv.ContainerInfo{
		ContainerReference: cadv.ContainerReference{
			Id:   id,
			Name: id,
		}}
	return &res
}
