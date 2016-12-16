# Running Snap in various environments
Snap can be deployed to collect metrics in various environments including Docker containers and Kubernetes. It can be run in a Docker container to gather metrics i.e. from host and other containers. Deployment of Snap along with Kubernetes cluster gives a possibility to monitor pods in the cluster.

1. [Getting started](#1-getting-started)  
2. [Snap in Docker container](#2-running-snap-in-docker-container)  
3. [Snap on Kubernetes cluster](#3-running-snap-on-kubernetes-cluster)
4. [Snap on Kubernetes with Heapster publisher](#4-running-snap-with-heapster-publisher)
5. [Customizing Snap Dockerfile and Kubernetes manifest for specific plugins](#5-cutomizing-snap-dockerfile-and-kubernetes-manifests-for-specific-plugins)
6. [Running Snap example on GCE](#8-running-snap-example-on-gce)

### 1. Getting started
To start examples below you need to have Kubernetes cluster set up.
First step is to download Snap repo. All of the needed files are in the `snap/integration` directory.
```sh
$ git clone https://github.com/intelsdi-x/snap
$ cd snap/integration
```

### 2. Running Snap in Docker container
In order to run Snap in a single docker containter you can pull Snap Docker image `intelsdi/snap` from official Snap repo on DockerHub (https://hub.docker.com/r/intelsdi/snap/) or build it by yourself. 

#### a) Running Snap in a container using DockerHub image
To run Snap with official image `intelsdi/snap` from DockerHub repo you simply run the command:
```sh
$ docker run -d --name snap -p 8181:8181 -e PROCFS_MOUNT=/proc_host -v /var/run/docker.sock:/var/run/docker.sock -v /proc:/proc_host -v /usr/bin/docker:/usr/local/bin/docker -v /var/lib/docker:/var/lib/docker intelsdi/snap
```
Mounting of the directories `-v /var/run/docker.sock:/var/run/docker.sock -v /proc:/proc_host -v /usr/bin/docker:/usr/local/bin/docker -v /var/lib/docker:/var/lib/docker` and `PROCFS_MOUNT` environment variable (`-e PROCFS_MOUNT=/proc_host`) is needed for Docker collector. Snap Docker collector allows to collect runtime metrics from Docker containers and its host machine. It gathers information about resource usage and performance characteristics. More information about docker collector can be found here: https://github.com/intelsdi-x/snap-plugin-collector-docker.

#### b) Running Snap in a container using your own image
However, if you prefer building Snap image on your own, you can use Dockerfile located in the directory `snap/integration/docker`. To build Snap image from Dockerfile, you run the command:

```sh
$ docker build -t <snap-image-name> snap/integration/docker
```
and when the image is ready, you may start Snap with your own image:
```sh
$ docker run -d --name snap -p 8181:8181 -e PROCFS_MOUNT=/proc_host -v /var/run/docker.sock:/var/run/docker.sock -v /proc:/proc_host -v /usr/bin/docker:/usr/local/bin/docker -v /var/lib/docker:/var/lib/docker <snap-image-name>
```
#### c) Loading Snap plugins inside running container
To verify that Snap container has started correctly and perform some actions to start collection of metrics we need to log into the Snap container. Getting into the container is quite simple:
```sh
$ docker exec -ti snap bash
```
Now that you are inside the container with running Snap daemon, you may run command to list the plugins:
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
$ curl -sO https://raw.githubusercontent.com/intelsdi-x/snap-plugin-collector-docker/master/examples/docker-file.json
$ snaptel task create -t ./docker-file.json
``` 
Command:
```sh
$ snaptel task list
``` 
will provide information whether the metrics collection was successfull or not. If you want to know how to load other plugins read the section [Customizing Snap Dockerfile and Kubernetes manifest for specific plugins](#5-cutomizing-snap-dockerfile-and-kubernetes-manifests-for-specific-plugins).

### 3. Running Snap on Kubernetes cluster
To run Snap in Kubernetes pods create daemonset from manifest file `snap/integration/kubernetes/snap.yaml`.
```sh
$ kubectl create -f snap/integration/kubernetes/snap.yaml
```
Verify that pods have been created. 
```sh
$ kubectl get pods --namespace=kube-system
```
Log into the one of pods using the pod name returned by `kubectl get pods` command.
```sh
$ kubectl exec -ti <snap-pod-name> bash --namespace=kube-system
```
Now that you are inside the pod with running Snap daemon, you may run command to list the plugins
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
will print information about two loaded plugins. To start the collection of metrics you can create task.
```sh
$ curl -sO https://raw.githubusercontent.com/intelsdi-x/snap/integration/integration/docker-file.json
$ snaptel task create -t ./docker-file.json
``` 
Command:
```sh
$ snaptel task list
``` 
will provide information whether the metrics collection was successfull or not. Use task watch to verify that tasks have been created correctly.
```sh
$ snaptel task watch <task-id>
```
If you want to know how to load other plugins read the section [Customizing Snap Dockerfile and Kubernetes manifest for specific plugins](#5-cutomizing-snap-dockerfile-and-kubernetes-manifests-for-specific-plugins).

### 4. Running Snap on Kubernetes with Heapster publisher
There is also a possibility to publish metrics gathered by Snap collector to Heapster (https://github.com/kubernetes/heapster). This solution requires Snap Heapster publisher (https://github.com/intelsdi-x/snap-plugin-publisher-heapster) and running customized Heapster Docker image with Snap source added so that Heapster can scratch metrics directly from running Snap instances.

#### a) Running Snap Heapster publisher
To start you need to follow the steps described in section [Snap on Kubernetes cluster](#3-running-snap-on-kubernetes-cluster). After that you download Snap Heapster publisher plugin and load it:
```sh
$ curl -fsL "https://github.com/intelsdi-x/snap-plugin-publisher-heapster/releases/download/1/snap-plugin-publisher-heapster_linux_x86_64" -o snap-plugin-publisher-heapster
$ snaptel plugin load ./snap-plugin-publisher-heapster
```
Then you can download an exemplary task manifest and create it:
```sh
$ curl -sO https://raw.githubusercontent.com/intelsdi-x/snap/integration/integration/docker-heapster.json
$ snaptel task create -t ./docker-heapster.json
``` 

#### b) Running customized Heapster with Snap source
To run the customized Heapster image you need to create Heapster service and deployment from manifest files `snap/integration/kubernetes/heapster-service.yaml` and `snap/integration/kubernetes/heapster-controller.yaml`.
```sh
$ kubectl create -f snap/integration/kubernetes/heapster/heapster-service.yaml
$ kubectl create -f snap/integration/kubernetes/heapster/heapster-controller.yaml
```
#### c) Verification
To verify that metric are being collected in Heapster you may check Heapster pod logs:
```
$ kubectl logs -f <heapster-pod-name> --namespace=kube-system
```

### 5. Customizing Snap Dockerfile and Kubernetes manifests for specific plugins
Inside a container you may use many different Snap plugins. Examples of loading plugins is described in sections [Snap in Docker container](#2-running-snap-in-docker-container) and [Snap on Kubernetes cluster](#3-running-snap-on-kubernetes-cluster). This way you may download and load almost any Snap plugin inside of the container. In order to get plugin binary URL you simply choose the plugin from plugin catalog  https://github.com/intelsdi-x/snap/blob/master/docs/PLUGIN_CATALOG.md. You click the plugin name. This redirects you to the plugin repository. Next you go to the `release` section... 


<img src="https://cloud.githubusercontent.com/assets/6523391/21221560/1c428440-c2be-11e6-9d73-6c565b88aa6e.png" width="70%">


...and copy the link for the latest plugin release.


<img src="https://cloud.githubusercontent.com/assets/6523391/21221622/69a08e6c-c2be-11e6-916f-f7179332b435.png" width="70%">


Some of the plugins require configuration. An example of such a plugin is CPU collector. 

#### a) Plugin configuration requirements
All of the plugins requirements can be found in their documentation. The documentation of the Snap plugin collector can be found here: https://github.com/intelsdi-x/snap-plugin-collector-cpu/blob/master/README.md. As it is stated in the documentation, the CPU plugin collector gathers information from the file `/proc/stat` residing in the host machine. Running this plugin inside the container requires mapping of this file inside of the container. The original host file `/proc/stat` has to be available inside of the container. This has to be done in both cases: Docker container and Kubernetes pod.

#### b) Customizing Snap for specific plugin in Docker conatainer
In a Docker container mapping of the files is done with the addition of `-v` flag when running the container. 
```sh
$ docker run -d --name snap -v <path-to-file-on-host>:<path-to-file-in-container> intelsdi/snap
```
What is more, CPU plugin requires enviroment variable `PROCFS_MOUNT` to be set. In Docker this is done with the use of `-e` flag when running the container.
```sh
$ docker run -d --name snap -v <path-to-file-on-host>:<path-to-file-in-container> -e PROCFS_MOUNT=<path-to-file-in-container> intelsdi/snap
```
So, to run Snap with CPU plugin reading resource usage from host you need to use command as below.
```sh
$ docker run -d --name snap -v /proc/host:/proc_host -e PROCFS_MOUNT=/proc_host intelsdi/snap
```

#### c) Customizing Snap for specific plugin in Kubernetes pod
To run CPU collector in Kubernetes pod we need to fullfill the same requirements. We have to mount the `/proc/stat` file inside of the pod and export `PROCFS_MOUNT` variable. In Kubernetes this adjustment needs to be added in the manifest file `snap/integration/kubernetes/snap.yaml`. 
Volumes are added with `volumeMounts` and `volume` parameters as shown below in the exemplary manifest. More information about mounting of volumes can be found in Kubernetes documentation (http://kubernetes.io/docs/user-guide/volumes/). And environment variable is added with `env` parameter (http://kubernetes.io/docs/tasks/configure-pod-container/define-environment-variable-container/).  
In order to run Snap with CPU collector you have to create `snap-example.yaml` file manifest shown below.
```
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: snap
  namespace: kube-system
  labels:
    kubernetes.io/cluster-service: "true"
spec:
  template:
    metadata:
      name: snap-example
      labels:
        daemon: snapteld
    spec:
      hostNetwork: true
      containers:
      - name: snap
        image: intelsdi/snap
        env:
        - name: PROCFS_MOUNT
          value: "/proc_host"  
        volumeMounts:
          - mountPath: /proc_host
            name: proc
        ports:
        - containerPort: 8181
          hostPort: 8181
          name: snap-api
        imagePullPolicy: Always
        securityContext:
          privileged: true
      volumes:
        - name: proc
          hostPath:
            path: /proc
```

And create the daemonset:
```sh
$ kubectl create -f snap-example.yaml
```

#### d) Loading specific plugin (CPU collector)
Next steps to have CPU collector plugin running are very similar to those described in section [Customizing Snap Dockerfile and Kubernetes manifest for specific plugins](#5-cutomizing-snap-dockerfile-and-kubernetes-manifests-for-specific-plugins). Inside the container or pod we run following commands:
```sh
$ curl -fsL "https://github.com/intelsdi-x/snap-plugin-collector-cpu/releases/download/6/snap-plugin-collector-cpu_linux_x86_64" -o snap-plugin-collector-cpu
$ curl -fsL "https://github.com/intelsdi-x/snap-plugin-publisher-file/releases/download/2/snap-plugin-publisher-file_linux_x86_64" -o snap-plugin-publisher-file
$ snaptel plugin load snap-plugin-collector-cpu
$ snaptel plugin load snap-plugin-publisher-file
$ curl -sO https://raw.githubusercontent.com/intelsdi-x/snap/integration/integration/cpu-file.json
$ snaptel task create -t ./cpu-file.json
```

### 6. Running Snap example on GCE
If you do not have Kubernetes cluster running, you can run kubesnap example on GCE (https://github.com/intelsdi-x/kubesnap).
#### 6.1. Start work with Google Cloud Platform
To start work with Google Cloud Platform you have to follow steps defined below.
#### a) Open Google Cloud Platform Console
 - go to https://console.cloud.google.com/  
 - log in using your e-mail address
 - follow the instruction [how to create a Cloud Platform Console project](https://cloud.google.com/storage/docs/quickstart-console)


#### b) Select your project  
- select your project from the drop-down menu in the top right corner
  <img src="https://raw.githubusercontent.com/intelsdi-x/kubesnap/master/docs/images/image_01.png">

#### c) Switch to _**Compute Engine**_ screen

- select _Products & Services_ from GC Menu in the top left corner  
  <img src="https://raw.githubusercontent.com/intelsdi-x/kubesnap/master/docs/images/image_02.png"> 

- and then select _Compute Engine_ from the drop-down list
  <img src="https://raw.githubusercontent.com/intelsdi-x/kubesnap/master/docs/images/image_03.png">

#### d) Create a new VM instance  
- create a new VM instance
  <img src="https://raw.githubusercontent.com/intelsdi-x/kubesnap/master/docs/images/image_04.png">

- set the instance name
- choose a machine with at least 4 vCPUs and at least 15GB RAM
- select Ubuntu 16.04 with standard persistent disk with at least 100GB
  <img src="https://raw.githubusercontent.com/intelsdi-x/kubesnap/master/docs/images/image_05.png">

#### e) Open the VM terminal by click on SSH  
 -  click on SSH to open the VM terminal (it will open as a new window)
  <img src="https://raw.githubusercontent.com/intelsdi-x/kubesnap/master/docs/images/image_07.png">

#### f) Authorize access to Google Cloud Platform  
- manage credentials for the Google Cloud SD. To do that, run the following command:
  ```sh
  $ gcloud auth login
  ```
  Answer `Y` to the question (see below) and follow the instructions:
  -	copy the link in your browser and 
  -	authenticate with a service account which you use in Google Cloud Environment,
  - copy the verification code from browser window and enter it
  <img src="https://raw.githubusercontent.com/intelsdi-x/kubesnap/master/docs/images/image_08.png">

- check if you are on credentialed accounts:  
 ```sh
 $ gcloud auth list
 ```

#### 6.2. Install kubesnap
Clone kubesnap into your home directory:

 ```sh
 $ git clone https://github.com/intelsdi-x/kubesnap
 ```

 Go to kubesnap/tools:
 ```sh
 $ cd kubesnap/tools
 ```

 Provision kubesnap (it takes approximately 35 minutes on a VM with 4 vCPUs and 15 GB of RAM in us-central1-b zone):
 ```sh
 $ ./provision-kubesnap.sh
 ```



