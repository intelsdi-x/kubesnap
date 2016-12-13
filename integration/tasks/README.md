# Running Snap in various environments
Snap can be deployed to collect metrics in various environments including Docker containers and Kubernetes. It can be run in a Docker container to gather metrics i.e. from host (monitoring resources usage, applications, other containers).
Deployment of Snap along with Kubernetes cluster gives a possibility to monitor all of the pods in the cluster.

## Getting started
First step is to download kubesnap repo. All of the needed files are in the `kubesnap/integration` directory.
```sh
$ git clone https://github.com/intelsdi-x/kubesnap
$ cd kubesnap/integration
```

### Running Snap in Docker container
In order to run Snap in a single docker containter you can pull Docker image from official Snap repo on DockerHub or build it by yourself. Dockerfile is in the directory `kubesnap/integration/docker`. You simply run the command:
```sh
$ docker run -d --name snap -p 8181:8181 -e PROCFS_MOUNT=/proc_host -v /var/run/docker.sock:/var/run/docker.sock -v /proc:/proc_host -v /usr/bin/docker:/usr/local/bin/docker -v /var/lib/docker:/var/lib/docker mkuculyma/test:1
```
Mounting of the directories `-v /var/run/docker.sock:/var/run/docker.sock -v /proc:/proc_host -v /usr/bin/docker:/usr/local/bin/docker -v /var/lib/docker:/var/lib/docker` and PROCFS_MOUNT environment variable `-e PROCFS_MOUNT=/proc_host` is needed for docker collector to gather metrics from the host.
```sh
$ docker exec -ti snap bash
```
Now that you are inside the container with running Snap daemon, you may run command to list the plugins
```sh
$ snaptel plugin list
``` 
The output should be `No plugins found. Have you loaded a plugin?` as there are no plugins loaded yet. Let's download some...

```sh
$ curl -fsL "https://github.com/intelsdi-x/snap-plugin-collector-docker/releases/download/5/snap-plugin-collector-docker_linux_x86_64" -o snap-plugin-collector-docker
$ curl -fsL "https://github.com/intelsdi-x/snap-plugin-publisher-file/releases/download/2/snap-plugin-publisher-file_linux_x86_64" -o snap-plugin-publisher-file
```
...and load them.
```sh
$ snaptel plugin load snap-plugin-collector-docker
$ snaptel plugin load snap-plugin-publisher-file
```
Now, running the command
```sh
$ snaptel plugin list
``` 
will print information about the two loaded plugins. To start the collection of metrics you can create task.
```sh
$ curl -sO https://raw.githubusercontent.com/intelsdi-x/snap-plugin-collector-docker/master/examples/tasks/docker-file.json
$ snaptel task create -t ./docker-file.json
``` 
Command
```sh
$ snaptel task list
``` 
will provide information whether the metrics collection was successfull or not.

### Running Snap on Kubernetes cluster

To run Snap in Kubernetes pods create daemonset from manifest file `snap.yaml`.
```sh
$ kubectl create -f kubesnap/integrations/k8s/snap.yaml
```
Verify that pods have been created.
```sh
$ kubectl get pods --namespace=kube-system
```
Log into the one of pods using the pod name returned by `kubectl get pods` command.
```sh
$ kubectl exec -ti snap-xxxxx bash --namespace=kube-system
```
Now that you are inside the pod with running Snap daemon, you may run command to list the plugins
```sh
$ snaptel plugin list
``` 
The output should be `No plugins found. Have you loaded a plugin?` as there are no plugins loaded yet. Let's download some...

```sh
$ curl -fsL "https://github.com/intelsdi-x/snap-plugin-collector-docker/releases/download/5/snap-plugin-collector-docker_linux_x86_64" -o snap-plugin-collector-docker
$ curl -fsL "https://github.com/intelsdi-x/snap-plugin-publisher-file/releases/download/2/snap-plugin-publisher-file_linux_x86_64" -o snap-plugin-publisher-file
$ curl -fsL "https://github.com/intelsdi-x/snap-plugin-publisher-heapster/releases/download/1/snap-plugin-publisher-heapster_linux_x86_64" -o snap-plugin-publisher-heapster
```
...and load them.
```sh
$ snaptel plugin load snap-plugin-collector-docker
$ snaptel plugin load snap-plugin-publisher-file
$ snaptel plugin load snap-plugin-publisher-heapster
```
Now, running the command
```sh
$ snaptel plugin list
``` 
will print information about three loaded plugins. To start the collection of metrics you can create task.
```sh
$ curl -sO https://raw.githubusercontent.com/intelsdi-x/kubesnap/integration/integration/tasks/docker-file.json
$ snaptel task create -t ./tasks/docker-file.json
``` 
Command
```sh
$ snaptel task list
``` 
will provide information whether the metrics collection was successfull or not.
```sh
$ curl -fsL "https://github.com/intelsdi-x/snap-plugin-collector-docker/releases/download/5/snap-plugin-collector-docker_linux_x86_64" -o snap-plugin-collector-docker
$ curl -fsL "https://github.com/intelsdi-x/snap-plugin-publisher-file/releases/download/2/snap-plugin-publisher-file_linux_x86_64" -o snap-plugin-publisher-file
$ curl -fsL "https://github.com/intelsdi-x/snap-plugin-publisher-heapster/releases/download/1/snap-plugin-publisher-heapster_linux_x86_64" -o snap-plugin-publisher-heapster
```
and load them:
```sh
$ snaptel plugin load snap-plugin-collector-docker
$ snaptel plugin load snap-plugin-publisher-file
$ snaptel plugin load snap-plugin-publisher-heapster
```
The last step is to download task manifests and create them in order to start the collection of metrics.

```sh
$ curl -sO https://raw.githubusercontent.com/intelsdi-x/kubesnap/integration/integration/tasks/docker-file.json
$ curl -sO https://raw.githubusercontent.com/intelsdi-x/kubesnap/integration/integration/tasks/docker-heapster.json
$ snaptel task create -t ./tasks/docker-file.json
$ snaptel task create -t ./tasks/docker-heapster.json
```
Use task watch to verify that tasks have been created correctly.
```sh
$ snaptel task watch <task_id>
```
