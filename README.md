# kubesnap

## How to run (single host example)

### Prerequisites

1. Running **Kubernetes** cluster - tested with Kubernetes 31de62216dea226f0553e698da38000b73684d49
2. Cloned **Heapster** repo - tested with Heapster de510e4bdcdea96722b5bde19ff0b7a142939485

### Steps

1. Copy files from heapster directory to **Heapster** repo

2. In **Heapster** repo build influxdb container:
```
docker build --no-cache=true --build-arg http_proxy=http://<proxy_ip>:<proxy_port> --build-arg https_proxy=https://<proxy_ip>:<proxy_port> -t influx_for_snap/ver:1 .
```

3. In **Heapster** repo build grafana container:
```
docker build --no-cache=true --build-arg http_proxy=http://<proxy_ip>:<proxy_port> -t grafana_for_snap/ver:1 .
```

4. Start influxdb-grafana-controller from **Heapster** repo
```
kubectl.sh create -f deploy/kube-config/influxdb/influxdb-grafana-controller.yaml
```

5. Start grafan Pod
```
kubectl.sh create -f deploy/kube-config/influxdb/grafana-service.yaml
```

6. Start influxdb Pod
```
kubectl.sh create -f deploy/kube-config/influxdb/influxdb-service.yaml
```

7. Export env. variables for Heapster
```
export KUBERNETES_SERVICE_HOST=127.0.0.1
export KUBERNETES_SERVICE_PORT=8080
```

8. Run stub server
Go to stubs directory and follow the instructions to start snap stub server

9. Build heapster from **Heapster** repo
```
make all
```

10. Run heapster from **Heapster** repo
```
./heapster --source=kubernetes.snap:http://127.0.0.1:8080 --sink=influxdb:http://10.0.0.15:8086
```

## APIs

API | Endpoint
----|-----
cadvisor | localhost:4194/api/v1.3
kubernetes | localhost:8080
heapster | localhost:8082/api/v1/model/metrics/
kubelet | localhost:10255/stats/container/
summary | localhost:10255/stats/summary
snap mock | localhost:8777/stats/container/

## GUIs

GUI | URL
----|-----
cadvisor | localhost:4194
influxdb | 10.0.0.15:8083
grafana | 10.0.0.16
