#### Resource consumption comparison


##### Introduction

This location contains scripts, tools and results related to running
a resource comparison between Kubesnap and Cadvisor telemetry tools.


##### Strategy

Due to the nature of Cadvisor and Kubesnap tools the comparison scenario
may be realized simply by launching the tools.

The following steps outline the strategy for resource comparison:
* Launch deployment environment
* Launch telemetry tools: Cadvisor and Kubesnap
* Wait until tools become responsive
* (optional) Launch any workload to cause desired impact on monitoring
  tools
* Continuously issue requests against monitoring tools' APIs, collecting
  results
* Gather and process the results

 
##### Supported setups

For now the following setups are supported:
* Deployment along local cluster of Kubernetes 


##### Tooling
   
The resource comparison test is equipped with following tool:
* `res-compare.sh`: utility for deploying environment, and gathering 
  response from monitoring-tools' APIs
  
###### Tool `res-compare.sh`

```
  Launch resource comparison between cadvisor and kubesnap

  Usage:
  res-compare.sh COMMAND

  Supported COMMANDs:
  - init: launch snap image locally
  - run_k8: download and launch kubernetes environment locally in docker,
    waiting for K8S to come up
  - killall: destroy ALL currently running docker containers
  - wait: block until kubesnap-publisher starts responding
  - stats_snap: grab 1 sample of container statistics from snap, as json
  - stats_cadvisor: grab 1 sample of container statistics from cadvisor, as json
  
  Supported variables:
  SNAP_REPO="${SNAP_REPO:-}"
  SNAP_IMAGE="${SNAP_IMAGE:-snap:1}"
  SNAP_PORT="${SNAP_PORT:-38777}"
  SNAP_PORT2="${SNAP_PORT:-38181}"
  K8S_VERSION="${K8S_VERSION:-v1.3.0-alpha.5}"
  CADVISOR_REPO="${CADVISOR_REPO:-google/}"
  CADVISOR_IMAGE="${CADVISOR_IMAGE:-cadvisor:v0.23.2}"
  CADVISOR_PORT="${CADVISOR_PORT:-34194}"
  HOST_IP="${HOST_IP:-localhost}"
```

`res-compare` deploys Kubesnap and Cadvisor from prebuilt docker images.
Snap image must contain required plugins (kubesnap-plugin-collector-docker
and kubesnap-plugin-publisher-heapster) and auto-launch a monitoring task.
