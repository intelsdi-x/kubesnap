#!/bin/bash
set -e
function usage {
        echo "Usage: "
        echo "$0 [OPTIONS]"
	echo "  Each option can be either 0 or 1"
        echo "  --tribe		- 0 starts snapd without tribe support, 1 starts snapd with tribe support"
	echo "  --docker	- 0 skips installation of docker engine, 1 install docker engine"
	echo "  --golang	- 0 skips installation of golang, 1 install golang"
	echo "  --gcloud	- 0 skips installation of gcloud sdk 114.0.0, 1 install gcloud sdk"
	echo "  --snap		- 0 skips building and pushing snap image, 1 builds and pushes snap image to gcr.io"
	echo "  --heapster	- 0 skips building and pushing heapster image, 1 build and pushes heapster image to gcr.io"
	echo "  --grafana	- 0 skips building and pushing grafana image, 1 build and pushes grafana image to gcr.io"
	echo "  --kubernetes	- 0 skips building of kubernetes, 1 builds kubernetes with release-skip-tests"
	echo "  --clone_repos	- 0 skips cloning of required github repos, 1 clones all required github repos"
	echo "  --help 		- displays this help"
        echo ""
        echo "After successfull installation of all components, start Kubernetes cluster with:"
        echo "$ cd kubernetes/; go run hack/e2e.go -v --up"
}

function parse_args {
	while [[ $# > 0 ]]; do
		case "$1" in
		--docker)	DOCKER="$2"; shift ;;
		--golang)	GOLANG="$2"; shift ;;
		--snap)		SNAP="$2"; shift ;;
		--heapster)	HEAPSTER="$2"; shift  ;;
		--grafana)	GRAFANA="$2"; shift  ;;
		--kubernetes)	KUBERNETES="$2"; shift  ;;
		--tribe) 	TRIBE="$2"; shift ;;
		--clone_repos)	REPOS="$2"; shift ;;
		--gcloud)	GCLOUD="$2"; shift;;
		--help)		usage ; exit ;;
		*)		echo "Error: invalid option '$1'" ; usage; exit 1 ;;
		esac
		shift
	done
}
 
function clone_repos {
	if [ -d "$HOME/kubesnap/src/snap/kubesnap-plugin-collector-docker" ]; then
		rm -rf $HOME/kubesnap/src/snap/kubesnap-plugin-collector-docker
	fi
	if [ -d "$HOME/kubesnap/src/snap/kubesnap-plugin-publisher-heapster" ]; then
		rm -rf $HOME/kubesnap/src/snap/kubesnap-plugin-publisher-heapster
	fi
	if [ -d "$HOME/heapster" ];then
		rm -rf $HOME/heapster
	fi
	if [ -d "$HOME/kubernetes" ];then
		rm -rf $HOME/kubernetes
	fi
        # Cloning required repos
        echo "===> CLONING docker collector" && git clone https://github.com/intelsdi-x/kubesnap-plugin-collector-docker $HOME/kubesnap/src/snap/kubesnap-plugin-collector-docker
        echo "===> CLONING heapster publisher" && git clone https://github.com/intelsdi-x/kubesnap-plugin-publisher-heapster $HOME/kubesnap/src/snap/kubesnap-plugin-publisher-heapster
        echo "===> CLONING heapster" && git clone https://github.com/andrzej-k/heapster $HOME/heapster
        echo "===> CLONING kubernetes" && git clone https://github.com/andrzej-k/kubernetes $HOME/kubernetes
}
function install_gcloud {
        # Installation of gcloud 114 version
	if [ -f "$HOME/google-cloud-sdk/path.bash.inc" ];then
		source $HOME/google-cloud-sdk/path.bash.inc
	fi
	ver=`gcloud --version | grep "SDK" | awk '{print $4}'`
	if [ "$ver" == "114.0.0" ]; then
		echo "===> GCLOUD version OK, skipping installation"
		return
	fi

       	echo "===> DOWNLOADING gcloud tar ball"
	wget -q https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-114.0.0-linux-x86_64.tar.gz -O gcloud-sdk-114.tar.gz
       	echo "===> UNPACKING tar ball"
	tar -xf gcloud-sdk-114.tar.gz -C $HOME && rm gcloud-sdk-114.tar.gz
	echo "===> INSTALLING gcloud"
       	yes | $HOME/google-cloud-sdk/install.sh
        
       	# The next line updates PATH for the Google Cloud SDK.
       	source $HOME/google-cloud-sdk/path.bash.inc
       	# The next line enables shell command completion for gcloud.
       	source $HOME/google-cloud-sdk/completion.bash.inc
        
       	expected=`echo $HOME/google-cloud-sdk/bin/gcloud`
       	current=`which gcloud`
       	if [ "$current" != "$expected" ];then 
               	echo "===> ERROR gcloud version different then expected!"
               	exit
       	fi
}
function install_golang {
	 sudo apt-get -y install golang-go
}
function install_docker {
        # Docker installation
        sudo apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
        echo "deb https://apt.dockerproject.org/repo ubuntu-xenial main" | sudo tee /etc/apt/sources.list.d/docker.list
        sudo apt-get -y update
        echo "===> INSTALLING docker"
        sudo apt-get -y install apt-transport-https ca-certificates linux-image-extra-$(uname -r) docker-engine
	sudo usermod -aG docker $USER
}
function build_snap_image {
        #Build snap image
        cd $HOME/kubesnap/src/snap
        echo "===> BUILD snap image for project $1"
        sg docker -c "docker build --no-cache  -t snap ."
        echo "===> TAG snap image"
        sg docker -c "docker tag snap gcr.io/$1/snap "
        echo "===> PUSH snap image"
        sg docker -c "gcloud docker push gcr.io/$1/snap"
}
function build_heapster_image {
        # Build heapster image
        cd $HOME/kubesnap/src/heapster
        echo "===> BUILD heapster image for project $1"
        sg docker -c "docker build --no-cache  -t heapster-snap ."
        echo "===> TAG heapster image"
        sg docker -c "docker tag heapster-snap gcr.io/$1/heapster-snap"
        echo "===> PUSH heapster image"
        sg docker -c "gcloud docker push gcr.io/$1/heapster-snap"
}

function build_grafana_image {
        # Build grafana image
        cd $HOME/heapster/grafana
        git checkout snap
        echo "===> BUILD grafana image for project $1"
        sg docker -c "docker build --no-cache -t grafana-snap ."
        echo "===> TAG grafana image"
        sg docker -c "docker tag grafana-snap gcr.io/$1/grafana-snap"
        echo "===> PUSH grafana image"
        sg docker -c "gcloud docker push gcr.io/$1/grafana-snap"
}

function build_kubernetes {
        echo "===> BUILDING Kubernetes for project $1 with tribe $tribe"
	sudo apt-get install -y make
	cd $HOME/kubernetes
	if $tribe; then
	        git checkout snap_tribe
	else
		git checkout snap
	fi
	echo "===> FIXING refs to project" 
        sed -i "s/snap4kube-1/$1/g" cluster/addons/snap/snap.yaml
        sed -i "s/snap4kube-1/$1/g" cluster/addons/cluster-monitoring/influxdb/heapster-controller.yaml 
        sed -i "s/snap4kube-1/$1/g" cluster/addons/cluster-monitoring/influxdb/influxdb-grafana-controller.yaml 
	
        sg docker -c "make release-skip-tests"
}

function get_project {
	out=`gcloud config list | grep project | awk -F "=" '{print $2}'`
	project="$(echo -e "${out}" | tr -d '[[:space:]]')"
}

function cleanup {
	rm -rf $HOME/heapster
	rm -rf $HOME/kubernetes
}

function set_defaults {
	REPOS="1"
	GCLOUD="1"
	DOCKER="1"
	GOLANG="1"
	SNAP="1"
	HEAPSTER="1"
	GRAFANA="1"
	KUBERNETES="0"
}

function success {
	if [ "$KUBERNETES" == "0" ];then
		echo "Build and start Kubernetes cluster with"
		echo "cd kubernetes/; KUBE_RELEASE_RUN_TESTS=n KUBE_FASTBUILD=true go run hack/e2e.go -v --build --up"
	else
		echo "Start Kubernetes cluster with"
		echo "$ cd kubernetes/; go run hack/e2e.go -v --up"
	fi
}

function main {
	#if [[ $(id -u) -ne 0 ]]; then
    	#	echo "Please re-run this script as root."
	#	usage
    	#	exit 1
	#fi
	
	set_defaults	
       	parse_args "$@"	
	
	if [ "$TRIBE" == "1" ];then
		tribe=true
	else
		tribe=false
	fi
	
	get_project
	
	if [ "$REPOS" == "1" ];then
        	clone_repos
	fi
	
	if [ "$GCLOUD" == "1" ];then
        	install_gcloud
	fi

	if [ "$GOLANG" == "1" ];then
		install_golang
	fi

	if [ "$DOCKER" == "1" ];then
        	install_docker
	fi 
	
	if [ "$SNAP" == "1" ];then        
        	build_snap_image $project
	fi
	
	if [ "$HEAPSTER" == "1" ];then
        	build_heapster_image $project
	fi

	if [ "$GRAFANA" == "1" ];then
        	build_grafana_image $project
	fi

	if [ "$KUBERNETES" == "1" ];then
        	build_kubernetes $project $tribe
	fi

	success
        
}

main $@
