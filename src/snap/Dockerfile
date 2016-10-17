FROM ubuntu
MAINTAINER mateusz.kleina@intel.com

RUN apt-get update
RUN apt-get -y install wget curl make git vim netcat gcc mysql-client

ENV DOCKER_BUCKET get.docker.com
ENV DOCKER_VERSION 1.10.3
ENV DOCKER_SHA256 d0df512afa109006a450f41873634951e19ddabf8c7bd419caeb5a526032d86d

RUN curl -fSL "https://${DOCKER_BUCKET}/builds/Linux/x86_64/docker-$DOCKER_VERSION" -o /usr/local/bin/docker \
	&& echo "${DOCKER_SHA256}  /usr/local/bin/docker" | sha256sum -c - \
	&& chmod +x /usr/local/bin/docker

# get golang for building snap and plugins
ENV GOLANG_VERSION 1.6.2
ENV GOLANG_DOWNLOAD_URL https://golang.org/dl/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_DOWNLOAD_SHA256 e40c36ae71756198478624ed1bb4ce17597b3c19d243f3f0899bb5740d56212a

RUN curl -fsSL "$GOLANG_DOWNLOAD_URL" -o golang.tar.gz \
	&& echo "$GOLANG_DOWNLOAD_SHA256  golang.tar.gz" | sha256sum -c - \
	&& tar -C /usr/local -xzf golang.tar.gz \
	&& rm golang.tar.gz

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
ENV GOMAXPROCS=1

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# build snap
RUN git clone https://github.com/intelsdi-x/snap.git /go/src/github.com/intelsdi-x/snap
WORKDIR /go/src/github.com/intelsdi-x/snap
RUN git reset --hard 902b9aec57f72943a244a744da39ef3e145df104
RUN make && make install

# build heapster publisher
COPY snap-plugin-publisher-heapster /go/src/github.com/intelsdi-x/snap-plugin-publisher-heapster
WORKDIR /go/src/github.com/intelsdi-x/snap-plugin-publisher-heapster
RUN make

# build docker collector
COPY snap-plugin-collector-docker /go/src/github.com/intelsdi-x/snap-plugin-collector-docker
WORKDIR /go/src/github.com/intelsdi-x/snap-plugin-collector-docker
RUN make

# build meminfo collector
COPY snap-plugin-collector-meminfo /go/src/github.com/intelsdi-x/snap-plugin-collector-meminfo
WORKDIR /go/src/github.com/intelsdi-x/snap-plugin-collector-meminfo
RUN make

# build processes collector
COPY snap-plugin-collector-processes /go/src/github.com/intelsdi-x/snap-plugin-collector-processes
WORKDIR /go/src/github.com/intelsdi-x/snap-plugin-collector-processes
RUN make

# build nova collector
COPY snap-plugin-collector-nova /go/src/github.com/intelsdi-x/snap-plugin-collector-nova
WORKDIR /go/src/github.com/intelsdi-x/snap-plugin-collector-nova
RUN make

# build keystone collector
COPY snap-plugin-collector-keystone /go/src/github.com/intelsdi-x/snap-plugin-collector-keystone
WORKDIR /go/src/github.com/intelsdi-x/snap-plugin-collector-keystone
RUN make

# build neutron collector
COPY snap-plugin-collector-neutron /go/src/github.com/intelsdi-x/snap-plugin-collector-neutron
WORKDIR /go/src/github.com/intelsdi-x/snap-plugin-collector-neutron
RUN make

# build glance collector
COPY snap-plugin-collector-glance /go/src/github.com/intelsdi-x/snap-plugin-collector-glance
WORKDIR /go/src/github.com/intelsdi-x/snap-plugin-collector-glance
RUN make

# create directory for snap plugins and task autoload
RUN mkdir -p /opt/snap/plugins
RUN mkdir -p /opt/snap/tasks
RUN mkdir -p /proc_host

# build start_snap binary
COPY start_snap /go/src/start_snap
WORKDIR /go/src/start_snap
RUN go build -o start_snap
RUN cp -a /go/src/start_snap/start_snap /usr/local/bin/start_snap 

# copy task
COPY tasks /opt/snap/tasks

# copy config
COPY config.json /opt/snap/config.json

# Install plugins in autoload directory
RUN cp -a /go/src/github.com/intelsdi-x/snap-plugin-publisher-heapster/build/rootfs/snap-plugin-publisher-heapster /opt/snap/plugins
RUN cp -a /go/src/github.com/intelsdi-x/snap-plugin-collector-docker/build/linux/x86_64/snap-plugin-collector-docker /opt/snap/plugins
RUN cp -a /go/src/github.com/intelsdi-x/snap-plugin-collector-keystone/build/linux/x86_64/snap-plugin-collector-keystone /opt/snap/plugins
RUN cp -a /go/src/github.com/intelsdi-x/snap-plugin-collector-glance/build/linux/x86_64/snap-plugin-collector-glance /opt/snap/plugins
RUN cp -a /go/src/github.com/intelsdi-x/snap-plugin-collector-neutron/build/linux/x86_64/snap-plugin-collector-neutron /opt/snap/plugins
RUN cp -a /go/src/github.com/intelsdi-x/snap-plugin-collector-nova/build/linux/x86_64/snap-plugin-collector-nova /opt/snap/plugins
RUN cp -a /go/src/github.com/intelsdi-x/snap-plugin-collector-meminfo/build/linux/x86_64/snap-plugin-collector-meminfo /opt/snap/plugins
RUN cp -a /go/src/github.com/intelsdi-x/snap-plugin-collector-processes/build/linux/x86_64/snap-plugin-collector-processes /opt/snap/plugins
RUN cp -a /go/src/github.com/intelsdi-x/snap/build/plugin/snap-publisher-file /opt/snap/plugins

# start snap daemon
ENV SNAPD_BIN /usr/local/bin/snapd
ENV SNAPCTL_BIN /usr/local/bin/snapctl
ENV PLUGINS_AUTOLOAD_DIR /opt/snap/plugins 
ENV PLUGINS_TO_LOAD 3
ENV TASK_AUTOLOAD_FILE /opt/snap/tasks/heapster-docker.json
ENV PROCFS_MOUNT /proc_host
EXPOSE 8181 8777
COPY ./start.sh /
ENTRYPOINT ["/start.sh"]
