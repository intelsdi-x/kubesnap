<!--
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
-->

[![Build Status](https://travis-ci.org/intelsdi-x/snap-plugin-publisher-heapster.svg?branch=master)](https://travis-ci.com/intelsdi-x/snap-plugin-publisher-heapster)

# snap publisher plugin - Heapster

This plugin provides a HTTP server, exposing metrics via REST API.
Conforming to API of Kubelet this publisher can be used as an alternate backend for [Heapster](https://github.com/kubernetes/heapster) monitoring tool.

1. [Getting Started](#getting-started)
  * [Operating systems](#operating-systems)
  * [Installation](#installation)
  * [To build the plugin binary](#to-build-the-plugin-binary)
  * [Configuration and usage](#configuration-and-usage)
2. [Documentation](#documentation)
  * [Supported REST server endpoints](#supported-rest-server-endpoints)
  * [Configuration parameters](#configuration-parameters)
  * [Custom metrics](#custom-metrics)
  * [Examples](#examples)
  * [Roadmap](#roadmap)
3. [Community Support](#community-support)
4. [Contributing](#contributing)
5. [License](#license-and-authors)
6. [Acknowledgements](#acknowledgements)

## Getting Started

This plugin needs to be used with [snap-plugin-collector-docker](https://github.com/intelsdi-x/snap-plugin-collector-docker).

Docker collector is necessary, because it delivers containers' metrics.
Its metrics define the structure for published data.
In particular docker collector delivers metrics for root container ('/')
which receives custom metrics by default.

### Operating systems
* Linux/amd64
* Darwin/amd64

### Installation

You can get the pre-built binaries for your OS and architecture at snap's [GitHub Releases](https://github.com/intelsdi-x/snap/releases) page.

### To build the plugin binary:
Fork https://github.com/intelsdi-x/snap-plugin-publisher-heapster
Clone repo into `$GOPATH/src/github.com/intelsdi-x/`:

```bash
$ git clone https://github.com/<yourGithubID>/snap-plugin-publisher-heapster.git
```

Build the plugin by running make within the cloned repo:
```bash
$ make
```
It may take a while to pull dependencies if you haven't had them already.

This builds the plugin in `/build/rootfs/`

### Configuration and Usage
* Set up the [snap framework](https://github.com/intelsdi-x/snap/blob/master/README.md#getting-started)
* Ensure `$SNAP_PATH` is exported
`export SNAP_PATH=$GOPATH/src/github.com/intelsdi-x/snap/build`

## Documentation

### Supported REST server endpoints

* **POST** `/stats/container/` - compatible with Kubelet, endpoint supports JSON requests.
Example request:
```json
{
    "containerName":"/",
    "num_stats":10,
    "start": "2016-09-09T12:57:05+02:00",
    "end": "2016-09-12T12:57:05+02:00",
	"subcontainers": true
}
```
As a matter of fact all parameters are optional, but not all should be left out (ie.: it would be wise to limit nubmer of stats per container).

Here is the copy of documentation for request copied from source code [exchange/exchange.go](https://github.com/intelsdi-x/snap-plugin-publisher-heapster/blob/45f8aad513347e59e6104d1cee1405e75854a8fc/exchange/exchange.go#L32-L53):
```go
// StatsRequest is a struct representing HTTP request for stats.
type StatsRequest struct {
	// ContainerName is name of the container to retrieve stats for
	// (defaults to root or '/').
	ContainerName string `json:"containerName,omitempty"`

	// NumStats defines amount of stats to return per container, maximum.
	// Ignored if start and end time is given.
	NumStats int `json:"num_stats,omitempty"`

	// Start defines time window limiting the stats to fetch per container.
	// Defaults to beginning of time.
	Start time.Time `json:"start,omitempty"`

	// End defines time window limiting the stats to fetch per container.
	// Defaults to current time.
	End time.Time `json:"end,omitempty"`

	// Subcontainers controls depth of query, allowing to fetch stats for
	// subcontainers.
	Subcontainers bool `json:"subcontainers,omitempty"`
}
```


### Configuration parameters

Plugin supports several parameters in task manifest (defaults given below):
```yaml
  plugin_name: "heapster"
  config:
    stats_depth: 0
    stats_span: "10m"
    server_addr: "127.0.0.1"
    server_port: 8777
    verbose_at: ""
    silent_at: ""
    mute_at: ""
```

All of them are optional. Their meaning is as follows:
* `stats_depth` - max. number of stats elements stored per each container
* `stats_span` - max. time difference between oldest and most recent stats elements per each container, e.g.:
  `"10m"` (10 minutes at most)
* `server_addr` - address the server should bind to
* `server_port` - port number the server should listen at
* `verbose_at`- space delimited list of modules which should be logging at verbose level, e.g.:
  `"/heapster /server"`
* `silent_at` - space delimited list of modules which should be logging at silent level (severity at least warning)
* `mute_at` - list of modules which should have logging muted (only errors logged)

List of modules in the publisher:
* `/heapster` - main module of the plugin,
* `/server` - module implementing REST server,
* `/processor/main` - module handling incoming docker metrics,
* `/processor/custom` - module handling incoming custom metrics.

### Custom metrics

Heapster publisher can expose custom metrics in addition to those reported
for containers. It can work with any number of additional collectors and 
integrate their metrics into the published stats. This can be 
accomplished purely by configuration.

Heapster recognizes specific tags on collected metrics and can expose
them via REST API. Tags may be set via task manifest with a per-metric
granularity. You can read more about tags in snap [documentation for tasks](https://github.com/intelsdi-x/snap/blob/d32bca6e6d6c00e8883af9df8bcd5a077ed7b744/docs/TASKS.md).

Here's the list of tags supported by heapster publisher with a description:
* `custom_metric_name` - name for the custom metric to appear under custom metrics list in the stats
  defaults to original namespace of the incoming metric
* `custom_metric_type` - type of the custom metric, as understood by Cadvisor, see [cadvisor/info/v1/metric.go](https://github.com/google/cadvisor/blob/1c8d7896a5225400df51b47330b50368e548eb94/info/v1/metric.go#L21-L33);
  defaults to `gauge`
* `custom_metric_format` - format of the metric data type, as understood by Cadvisor, see [cadvisor/info/v1/metric.go](https://github.com/google/cadvisor/blob/1c8d7896a5225400df51b47330b50368e548eb94/info/v1/metric.go#L35-L41);
  defaults to `int`
* `custom_metric_units` - purely informative purpose, describes the units of the metric data;
  defaults to `none`
* `custom_metric_container_path` - name of the container that should have those custom metrics assigned;
  defaults to `/`, or root container

A metric must be tagged with **at least one** of the tags above to be recognized
as custom metric.

An example manifest for a task exercising custom metrics may be found in
file [examples/tasks/heapster-custom-metrics.yaml](https://github.com/intelsdi-x/snap-plugin-publisher-heapster/blob/master/example/tasks/heapster-custom-metrics.yaml).

### Examples

You can find example task manifests for using this plugin in folder [examples/tasks/](https://github.com/intelsdi-x/snap-plugin-publisher-heapster/tree/master/exchange/examples/tasks)
Start snap in one terminal window:

```bash
$SNAP_PATH/bin/snapd -l 1 -t 0
```

In another terminal window load required plugins - docker and heapster
(replace fully-qualified paths with actual plugin binaries if you downloaded them):
```bash
cd $GOPATH/src/github.com/intelsdi-x/
$SNAP_PATH/bin/snapctl plugin load ./snap-plugin-collector-docker/build/rootfs/snap-plugin-collector-docker
$SNAP_PATH/bin/snapctl plugin load ./snap-plugin-collector-docker/build/rootfs/snap-plugin-publisher-heapster
```

Create task manifest referencing docker- and heapster- plugins:
```yaml
  version: 1
  schedule:
    type: "simple"
    interval: "10s"
  workflow:
    collect:
      metrics:
        /intel/docker/*: {}
      publish:
        -
          plugin_name: "heapster"
          config:
            stats_span: "10m"
            server_addr: "127.0.0.1"
            server_port: 8777
```

Create a task by the following command:
```bash
cd $GOPATH/src/github.com/intelsdi-x/
$ $SNAP_PATH/bin/snapctl task create -t ./snap-plugin-publisher-heapster/examples/tasks/heapster-with-docker.yaml

Using task manifest to create task
Task created
ID: 9b26dc87-a3f8-4e8a-9303-ef88cb12e3a2
Name: Task-9b26dc87-a3f8-4e8a-9303-ef8cb12e3a2
State: Running
```

Verify that heapster publisher correctly exposes metrics:
```bash
$ curl -H 'Content-Type: application/json' -X POST -d '{"num_stats": 1}' 127.0.0.1:8777/stats/container/ |python -m json.tool
```

You should see output similar to this one:
```json
{
    "/": {
        "id": "/",
        "name": "/",
        "spec": {
            "cpu": {
                "limit": 0,
                "max_limit": 0
            },
            "creation_time": "2016-06-16T13:05:16.10346462+02:00",
            "custom_metrics": [
            ],
            "has_cpu": false,
            "has_custom_metrics": true,
            "has_diskio": false,
            "has_filesystem": true,
            "has_memory": false,
            "has_network": true,
            "memory": {}
        },
        "stats": [
            {
                "cpu": {
                    "load_average": 0,
                    "usage": {
                        "per_cpu_usage": [
                            9409993187764,
                            5362499917666,
                            9168442203791,
                            6403390387149,
                            9431651754669,
                            9164956796928,
                            4759647789316,
                            5411972414128
                        ],
                        "system": 10640710000000,
                        "total": 59112554451411,
                        "user": 50338490000000
                    }
                },
[...] Rest ommitted
```

### Roadmap

There isn't a current roadmap for this plugin, but it is in active development.
As we launch this plugin, we do not have any outstanding requirements for the next release. If you have a feature request, please add it as an [issue](https://github.com/intelsdi-x/snap-plugin-collector-docker/issues/new) and/or submit a [pull request](https://github.com/intelsdi-x/snap-plugin-publisher-heapster/pulls).

## Community Support
This repository is one of **many** plugins in the **snap framework**: a powerful telemetry framework. See the full project at http://github.com/intelsdi-x/snap To reach out to other users, head to the [main framework](https://github.com/intelsdi-x/snap#community-support)

## Contributing
We love contributions!

There's more than one way to give back, from examples to blogs to code updates. See our recommended process in [CONTRIBUTING.md](CONTRIBUTING.md).

## License
[snap](http://github.com/intelsdi-x/snap), along with this plugin, is an Open Source software released under the Apache 2.0 [License](LICENSE).

## Acknowledgements

* Author:       [Marcin Olszewski](https://github.com/marcintao)

**Thank you!** Your contribution is incredibly important to us.
