#!/bin/bash
set -e 
function clone_repos {
        # Cloning required repos
        echo "===> CLONING docker collector" && git clone https://github.com/intelsdi-x/kubesnap-plugin-collector-docker /home/marcin_krolik/kubesnap/src/snap/kubesnap-plugin-collector-docker
        echo "===> CLONING heapster publisher" && git clone https://github.com/intelsdi-x/kubesnap-plugin-publisher-heapster /home/marcin_krolik/kubesnap/src/snap/kubesnap-plugin-publisher-heapster
        echo "===> CLONING grafana" && git clone https://github.com/andrzej-k/heapster $HOME/heapster
        echo "===> CLONING kubernetes" && git clone https://github.com/andrzej-k/kubernetes $HOME/kubernetes
}
function install_gcloud {
        # Installation of gcloud 114 version
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
function install_docker {
        # Docker installation
        apt-get install -y apt-transport-https ca-certificates
        apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
        echo "deb https://apt.dockerproject.org/repo ubuntu-xenial main" | tee /etc/apt/sources.list.d/docker.list
        apt-get -y update
                echo "===> INSTALLING docker"
        apt-get -y install docker-engine
}
function build_snap_image {
        #Build snap image
        cd $HOME/kubesnap/src/snap
        echo "===> BUILD snap image"
        docker build --no-cache  -t snap .
        echo "===> TAG snap image"
        docker tag snap gcr.io/$PROJECT/snap 
        echo "===> PUSH snap image"
        gcloud docker push gcr.io/$PROJECT/snap
}
function build_heapster_image {
        # Build heapster image
        cd $HOME/kubesnap/src/heapster
        echo "===> BUILD heapster image"
        docker build --no-cache  -t heapster-snap .
        echo "===> TAG heapster image"
        docker tag heapster-snap gcr.io/$PROJECT/heapster-snap
        echo "===> PUSH heapster image"
        gcloud docker push gcr.io/$PROJECT/heapster-snap
}

function build_grafana_image {
        # Build grafana image
        cd $HOME/heapster/grafana
        git checkout snap
        echo "===> BUILD grafana image"
        docker build --no-cache -t grafana-snap .
        echo "===> TAG grafana image"
        docker tag grafana-snap gcr.io/$PROJECT/grafana-snap
        echo "===> PUSH grafana image"
        gcloud docker push gcr.io/$PROJECT/grafana-snap
}

function build_kubernetes {
        echo "===> BUILDING Kubernetes"
	cd $HOME/kubernetes
        git checkout snap
	echo "===> FIXING refs to project" 
        sed -i "s/snap4kube-1/$PROJECT/g" cluster/addons/snap/snap.yaml
        sed -i "s/snap4kube-1/$PROJECT/g" cluster/addons/cluster-monitoring/influxdb/heapster-controller.yaml 
        sed -i "s/snap4kube-1/$PROJECT/g" cluster/addons/cluster-monitoring/influxdb/influxdb-grafana-controller.yaml 

        make -j release-skip-tests 
}

function install_make {
        #install make
        apt-get install -y make
}

function usage {
        echo "Usage: "
        echo "  $0 project_name"
}

function main {
        
        PROJECT=$1
        if [ "$PROJECT" == "" ]; then
                usage
                exit
        else
                echo "===> START building $PROJECT"
        fi      
        clone_repos
        install_gcloud
        install_docker
        install_make
                
        build_snap_image
        build_heapster_image
        build_grafana_image
        build_kubernetes
        
        echo "===> FINISHED"
}

main $1
