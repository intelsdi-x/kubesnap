#!/bin/bash

## snap_repo, if not local should end with slash '/'
SNAP_REPO="${SNAP_REPO:-}"
SNAP_IMAGE="${SNAP_IMAGE:-snap:1}"
SNAP_PORT="${SNAP_PORT:-38777}"
SNAP_PORT2="${SNAP_PORT:-38181}"
K8S_VERSION="${K8S_VERSION:-v1.3.0-alpha.5}"
CADVISOR_REPO="${CADVISOR_REPO:-google/}"
CADVISOR_IMAGE="${CADVISOR_IMAGE:-cadvisor:v0.23.2}"
CADVISOR_PORT="${CADVISOR_PORT:-34194}"
HOST_IP="${HOST_IP:-localhost}"
quiet=""

####
##  Launch resource comparison between cadvisor and kubesnap
##
##  Usage:
##  res-compare.sh COMMAND
##
##  Supported COMMANDs:
##  - init: launch snap image locally
##  - run_k8: download and launch kubernetes environment locally in docker,
##    waiting for K8S to come up
##  - killall: destroy ALL currently running docker containers
##  - wait: block until kubesnap-publisher starts responding
##  - stats_snap: grab 1 sample of container statistics from snap, as json
##  - stats_cadvisor: grab 1 sample of container statistics from cadvisor, as json


run_kubernetes ()
{
	export K8S_VERSION="$K8S_VERSION"
	export ARCH=amd64
	$quiet docker run -d \
	    --volume=/:/rootfs:ro \
	    --volume=/sys:/sys:rw \
	    --volume=/var/lib/docker/:/var/lib/docker:rw \
	    --volume=/var/lib/kubelet/:/var/lib/kubelet:rw \
	    --volume=/var/run:/var/run:rw \
	    --net=host \
	    --pid=host \
	    --privileged \
	    gcr.io/google_containers/hyperkube-${ARCH}:${K8S_VERSION} \
		/hyperkube kubelet \
		--containerized \
		--hostname-override=127.0.0.1 \
		--api-servers=http://localhost:8080 \
		--config=/etc/kubernetes/manifests \
		--cluster-dns=10.0.0.10 \
		--cluster-domain=cluster.local \
		--allow-privileged --v=2
}

run_snap ()
{
	$quiet docker run -dt --name snap --privileged \
		-v /vagrant/:/vagrant/ -v /run/systemd/:/run/systemd/ -v /var/lib/docker/:/var/lib/docker/ \
		-v /sys/:/sys/ -v /var/run/docker.sock:/var/run/docker.sock \
		-v /proc/:/host_proc \
		-p ${SNAP_PORT2}:8181 -p ${SNAP_PORT}:8777 \
		-e PROCFS_MOUNT=/host_proc \
		${SNAP_REPO}${SNAP_IMAGE}
}

run_cadvisor ()
{
	$quiet docker run \
		-v /:/rootfs:ro   -v /var/run:/var/run:rw \
		-v /sys:/sys:ro   -v /var/lib/docker/:/var/lib/docker:ro \
		-p ${CADVISOR_PORT}:8080    \
		--detach=true   \
		--name=cadvisor \
		${CADVISOR_REPO}${CADVISOR_IMAGE}
}

run_init ()
{
#	run_kubernetes
	run_snap
	run_cadvisor
}

kill_all ()
{
	sudo pkill kubelet; docker pause $(docker ps -aq)
	for id in $(docker ps -aq); do docker unpause $id; docker rm -fv $id; done
}

run_k8 ()
{
	echo "  "
	while true; do
		run_kubernetes		
		echo -n "waiting for k8s to come up "
		for i in `seq 1 11`; do echo -n "."; sleep 1; done
		if [ $( docker ps --format '{{.ID}}' |wc -l ) == "0" ]; then
			echo -e " [FAIL]\n  rinse and repeat- "
			kill_all
			kill_all
			sleep 3
		else
			echo " [uP]"
			return 0
		fi
	done
}

kill_stats ()
{
	docker rm -fv snap
	docker rm -fv cadvisor
}

wait_for_publisher ()
{
	echo -n "  "
	echo -n "waiting for publisher to come up "
	until [ $(curl -sL -w "%{http_code}\\n" -H 'Content-Type: application/json' -X POST -d '{"num_stats": 1}' "${HOST_IP}:38777/stats/container/" -o /dev/null) == "200" ]; do echo -n "."; sleep 1; done; echo  " [uP]"
}

whdocker ()
{
	ptrn="$1"
	found=`docker ps --format "{{.ID}} {{.Image}}" |grep -ie "$ptrn" |awk '{print $1}'`
	printf "Found id: %s\n" "$found"
}


if [[ "$1" == "init" ]]; then
	run_init
fi

if [[ "$1" == "run_k8" ]]; then
	run_k8
fi

if [[ "$1" == "killall" ]]; then
	kill_all
fi

if [[ "$1" == "kill" ]]; then
	kill_stats
fi

if [[ "$1" == "boot" ]]; then
	run_init
	wait_for_publisher
fi

if [[ "$1" == "wait" ]]; then
	wait_for_publisher
fi

if [[ "$1" == "stats_snap" ]]; then
	curl -H 'Content-Type: application/json' -X POST -d '{"num_stats": 1}' ${HOST_IP}:${SNAP_PORT}/stats/container/ 2>/dev/null |python -m json.tool
fi

if [[ "$1" == "whdocker" ]]; then
	whdocker "snap"
	whdocker "cadvisor"
fi

if [[ "$1" == "stats_cadvisor" ]]; then
	curl -H 'Content-Type: application/json' -d '{"num_stats": 1}' http://${HOST_IP}:${CADVISOR_PORT}/api/v1.2/docker/ 2>/dev/null | python -m json.tool
fi

