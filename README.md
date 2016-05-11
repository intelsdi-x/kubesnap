# kubesnap

## How to run

### Prerequisites

- **Kubernetes** cluster - tested with Kubernetes on CoreOS
- One minion node labeled as tribe chief:
```
kubectl label nodes 10.91.97.195 tribe-role=chief
```
- Other minion nodes labeled as tribe members:
```
kubectl label nodes 10.91.97.194 tribe-role=member
```

### Steps

#### Building

- Clone **Heapster**
```
git clone https://github.com/kubernetes/heapster.git
cd heapster
git reset --hard de510e4bdcdea96722b5bde19ff0b7a142939485
```

- Clone **kubesnap**
```
git clone https://github.com/intelsdi-x/kubesnap.git
```

- Build **Heapster** container
```
cd kubesnap/src/heapster
docker build -t heapster-snap .
docker tag heapster-snap <docker_registry>/heapster-snap:1
docker push <docker_registry>/heapster-snap:1
```

- Build **Influxdb** container
```
cd heapster/influxdb
docker build -t influxdb-snap .
docker tag influxdb-snap <docker_registry>/influxdb-snap:1
docker push <docker_registry>/influxdb-snap:1
```

- Build **Grafana** container
```
cd heapster/grafana
docker build -t grafana-snap .
docker tag grafana-snap <docker_registry>/grafana-snap:1
docker push <docker_registry>/grafana-snap:1
```

- Build **snap** container
```
cd kubesnap/src/snap
docker build -t snap .
docker tag snap <docker_registry>/snap:1
docker push <docker_registry>/snap:1
```

- (Optional) Build **workload** container
```
cd kubesnap/src/workload
docker build -t workload .
docker tag workload <docker_registry>/workload:1
docker push <docker_registry>/workload:1
```

- (Optional) Build **snap_stub** container
```
cd kubesnap/src/snap_stub
docker build -t snap_stub .
docker tag snap_stub 10.1.23.1:5000/snap_stub:1
docker push 10.1.23.1:5000/snap_stub:1
```

#### Running

- Deploy snap (or snap_stub)
```
cd kubesnap/deploy
kubectl create -f snap/ OR kubectl create -f snap_stub/
```

- Deploy Heapster, InfluxDB and Grafana
```
cd kubesnap/deploy
kubectl create -f heapster/multi-node/
```

- (Optional) Deploy workload
```
cd kubesnap/deploy
kubectl create -f workload/
```

#### Demo with Horizontal Pod Autoscaler 

- Autoscale the workload, an example:
```
kubectl autoscale rc workload-controller --min=1 --max=5 --cpu-percent=5 --namespace=kube-system
```

- Increase CPU utilization in workload-controller (in this example workload will use 2 CPUs):
```
curl -H "Content-Type: application/json" -X POST -d '{"load":2}' http://10.2.50.52:7777/set_load
curl http://10.2.50.52:7777/load (to check current load)
```
