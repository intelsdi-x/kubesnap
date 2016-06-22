#!/bin/bash

tag_and_push ()
{
	if allow_action $1; then
		echo " - pushing docker image: $1"
		docker tag $1 $DOCKER_REG_ADDR/$1:1
		docker push $DOCKER_REG_ADDR/$1:1
	fi
}

build_docker_in ()
{
	if allow_action $2; then
		pushd "$1"
		echo " - building docker image: $2 in directory: $1"
		docker build -t $2 .
		popd
	fi
}

error_step0_needs_var ()
{
	echo -e "\nRequired variable: $1 must be set in ENV prior to running this script.\nCorrect your environment and re-run step 0."
	exit -1
}

error_step0_needs_cmdfile ()
{
	echo -e "\nRequired file: $1 wasn't found in cmddir=$cmddir.\nCorrect your environment and re-run step 0."
	exit -1
}

setup_cluster_roles ()
{
	for ctrl in `"$cmddir/kubectl" get nodes --no-headers |grep -ie SchedulingDisabled |awk '{print $1}'`
	do
		"$cmddir/kubectl" label nodes $ctrl tribe-role=chief
	done
	for node in `"$cmddir/kubectl" get nodes --no-headers |grep -vie SchedulingDisabled |awk '{print $1}'`
	do
		"$cmddir/kubectl" label nodes $node tribe-role=member
	done
}

git_clone ()
{
	echo "- cloning git repo: $1 [`basename $1`]"
	git clone $1
}

allow_action ()
{
	action=$1
	in_excludes=0
	if [[ ":$exclude_actions:" == *":$action:"* ]]; then
		in_excludes=1
	fi
	if [[ $include_actions == "all" ]]; then
		if [ $in_excludes -eq 1 ]; then
			return 1
		fi
		return 0
	fi
	if [[ $exclude_actions == "all" ]]; then
		if [[ ":$include_actions:" == *":$action:"* ]]; then
			return 0
		fi
		return 1
	fi
	if [ $in_excludes -eq 1 ]; then
		return 1
	fi
	return 0
}

## NOT USED YET (this is just a braindump of tricks used on GCE)
## ensure_docker_up
##
## - ensure that docker service is up or try to bring it up
ensure_docker_up ()
{
	## if docker isn't ready, try restarting
	#sudo systemctl status docker >/dev/null && echo "oK"
	#sudo systemctl restart docker
	#sudo systemctl status docker >/dev/null && echo "oK"
	## if docker still isn't running, try a fix
	#sudo apt-get update 
	#sudo apt-get install linux-image-extra-$(uname -r) 
	#sudo systemctl restart docker 
	## if docker isn't running at this point, despair and cry
	exit -1
}

## NOT USED YET (this is just a braindump of tricks used on GCE)
## fix_refs_to_gce_project target_proj
##	target_proj - name of the target project to refer to in kubernetes deployment
## 	
## - fix references to gce project in kubernetes deployment files
## usage:
## 
fix_refs_to_gce_project ()
{
	target_proj="$1"
	pushd kubernetes/cluster/addons
	grep -rile snap4kube-1 --include=*.yaml |xargs sed -i 's/snap4kube-1/$target_proj/g'
	popd
}

workdir="${workdir:-`pwd`}"
cmddir0=`dirname "$0"`
cmddir="${cmddir:-$cmddir0}"
voldir="${voldir:-$cmddir/dockstore}"
include_actions=${include_actions:-}
exclude_actions=${exclude_actions:-}
if [[ ( -z "$include_actions" ) && ( -z "$exclude_actions" ) ]]; then
	include_actions=all
elif [[ -z "$exclude_actions" ]]; then
	exclude_actions=all
fi

echo -e "\nKubesnap dev setup started with with:\n  workdir=$workdir\n  cmddir=$cmddir\n  voldir=$voldir\n  step=$1\n  include_actions=$include_actions\n  exclude_actions=$exclude_actions"

if [[ ( -z "$1" ) || ( "$1" == "0" ) || ( "$1" == "setenv" ) ]]
then
	echo "-- step 0: set up environment (setenv)"
	if [[ -z "$DOCKER_REG_ADDR" ]];	then
		error_step0_needs_var "DOCKER_REG_ADDR"
	fi
	if [[ -z "$OAUTH_TOKEN" ]]; then
		error_step0_needs_var "OAUTH_TOKEN"
	fi
	if [[ ! -e "$cmddir/kubectl" ]]; then
		error_step0_needs_cmdfile "kubectl"
	fi

	if allow_action "configure_roles" ; then
		echo " - configure cluster roles [configure_roles]"
		setup_cluster_roles
	fi

	if allow_action "start_registry" ; then
		echo " - start docker registry [start_registry]"
		mkdir -p "$voldir"
		"$cmddir/start-reg.sh" "$voldir"
	fi
	if allow_action "use_registry" ; then
		echo -e" - use docker registry: reconfigure daemon to use provided registry [use_registry]"
		"$cmddir/use-reg.sh"
	fi
	if allow_action "configure_git" ; then
		echo " - configure github access in git [configure_git]"
		git config --global url."https://$OAUTH_TOKEN:x-oauth-basic@github.com/".insteadOf "https://github.com/"
	fi

	if allow_action "configure_deploy" ; then
		echo " - configure scripts for deploy [configure_deploy]"
		grep "$workdir/kubesnap/deploy/" -rile "10\.1\.23\.1:5000" |xargs -l sed -i "s/10\.1\.23\.1:5000/$DOCKER_REG_ADDR/g"
	fi
fi

if [[ ( -z "$1" ) || ( "$1" == "1" ) || ( "$1" == "getsrc" ) ]]
then
	echo "-- step 1: getting sources (getsrc)"
	pushd $workdir
	if allow_action "heapster.git"; then
		git_clone "https://github.com/kubernetes/heapster.git"
		pushd heapster
		git reset --hard de510e4bdcdea96722b5bde19ff0b7a142939485
		popd
	fi
	if allow_action "kubesnap.git"; then
		git_clone "https://github.com/intelsdi-x/kubesnap.git"
	fi
	mkdir -p kubesnap/src/snap
	pushd kubesnap/src/snap
	if allow_action "kubesnap-plugin-publisher-heapster.git"; then
		git_clone "https://github.com/intelsdi-x/kubesnap-plugin-publisher-heapster.git"
	fi
	if allow_action "kubesnap-plugin-collector-docker.git"; then
		git_clone "https://github.com/intelsdi-x/kubesnap-plugin-collector-docker.git"
	fi
	popd
	popd
fi

if [[ ( -z "$1" ) || ( "$1" == "2" ) || ( "$1" == "build" ) ]]
then
	pushd $workdir
	echo "-- step 2: building containers (build)"
	echo " - building containers"
	build_docker_in "$workdir/kubesnap/src/heapster" heapster-snap
	build_docker_in "$workdir/heapster/influxdb" influxdb-snap
	build_docker_in "$workdir/heapster/grafana" grafana-snap
	build_docker_in "$workdir/kubesnap/src/snap" snap
	build_docker_in "$workdir/kubesnap/src/workload" workload
	build_docker_in "$workdir/kubesnap/src/snap_stub" snap_stub
	popd
fi

if [[ ( -z "$1" ) || ( "$1" == "3" ) || ( "$1" == "push" )  ]]
then
	echo "-- step 3: pushing containers (push) "
	tag_and_push "heapster-snap"
	tag_and_push "influxdb-snap"
	tag_and_push "grafana-snap"
	tag_and_push "snap"
	tag_and_push "workload"
	tag_and_push "snap_stub"
fi




