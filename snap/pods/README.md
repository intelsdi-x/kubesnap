# snap pods

There are two pod manifest templates for snap:

- snap-daemonset.yml for deploying set of snap daemons on several nodes, the number of daemons is maintained by Kubernetes's replication controller
- snap-pod.yml for deploying one snap daemon on single node

## Configure snap arguments

To run snap pod with extra command line arguments for snap daemon, you must create new file based on template and modify **args: []** field. For example:

	args: ["--tribe"]

## Create pod or daemonset

To create pod or daemonset, use kubectl tool:
	
	kubectl create -f <pod_manifest_file>

## Remove pod or daemonset

To stop and remove pod from cluster:

	kubectl delete pod <pod_name>

To stop and remove daemonset from cluster:

	kubectl delete ds <daemonset_name>
