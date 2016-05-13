FROM ubuntu
MAINTAINER mateusz.kleina@intel.com

RUN apt-get update
RUN apt-get -y install wget curl make git

# get golang for building snap and plugins
ENV GOLANG_VERSION 1.6.1
ENV GOLANG_DOWNLOAD_URL https://golang.org/dl/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_DOWNLOAD_SHA256 6d894da8b4ad3f7f6c295db0d73ccc3646bce630e1c43e662a0120681d47e988

RUN curl -fsSL "$GOLANG_DOWNLOAD_URL" -o golang.tar.gz \
	&& echo "$GOLANG_DOWNLOAD_SHA256  golang.tar.gz" | sha256sum -c - \
	&& tar -C /usr/local -xzf golang.tar.gz \
	&& rm golang.tar.gz

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
ENV GOMAXPROCS=2

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# build snap (with support of dynamic query mechanism)
RUN git clone https://github.com/IzabellaRaulin/snap.git /go/src/github.com/intelsdi-x/snap
WORKDIR /go/src/github.com/intelsdi-x/snap
RUN git checkout remotes/origin/specific_dynamic_metric_rebased
RUN git reset --hard 6e9c54b3c9cd14f1844ccc78b04ca4665810abce
RUN make && make install

# build heapster publisher
COPY kubesnap-plugin-publisher-heapster /go/src/github.com/intelsdi-x/kubesnap-plugin-publisher-heapster
WORKDIR /go/src/github.com/intelsdi-x/kubesnap-plugin-publisher-heapster
RUN make

# build docker collector
COPY kubesnap-plugin-collector-docker /go/src/github.com/intelsdi-x/kubesnap-plugin-collector-docker
WORKDIR /go/src/github.com/intelsdi-x/kubesnap-plugin-collector-docker
RUN make

# Create directory for snap plugins autoload
RUN mkdir -p /opt/snap/plugins

# Install plugins in autoload directory
RUN cp -a /go/src/github.com/intelsdi-x/kubesnap-plugin-publisher-heapster/build/rootfs/snap-plugin-publisher-heapster /opt/snap/plugins
RUN cp -a /go/src/github.com/intelsdi-x/kubesnap-plugin-collector-docker/build/rootfs/snap-plugin-collector-docker /opt/snap/plugins

# Start snap daemon
EXPOSE 8181 8777
ENTRYPOINT ["/usr/local/bin/snapd", "-t", "0", "-a", "/opt/snap/plugins"]
