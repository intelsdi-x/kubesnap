FROM ubuntu
MAINTAINER mateusz.kleina@intel.com

RUN apt-get update
RUN apt-get -y install wget curl make git


ENV GOLANG_VERSION 1.6.1
ENV GOLANG_DOWNLOAD_URL https://golang.org/dl/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_DOWNLOAD_SHA256 6d894da8b4ad3f7f6c295db0d73ccc3646bce630e1c43e662a0120681d47e988

RUN curl -fsSL "$GOLANG_DOWNLOAD_URL" -o golang.tar.gz \
	&& echo "$GOLANG_DOWNLOAD_SHA256  golang.tar.gz" | sha256sum -c - \
	&& tar -C /usr/local -xzf golang.tar.gz \
	&& rm golang.tar.gz

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

COPY publisher/snap-plugin-collector-users /go/src/github.com/intelsdi-x/snap-plugin-collector-users
WORKDIR /go/src/github.com/intelsdi-x/snap-plugin-collector-users

ENV GOMAXPROCS=1
RUN go get github.com/tools/godep
RUN cd /go/src/github.com/intelsdi-x/snap-plugin-collector-users && make

# Download and install snap release
RUN wget https://github.com/intelsdi-x/snap/releases/download/v0.13.0-beta/snap-v0.13.0-beta-linux-amd64.tar.gz
RUN tar -zxf snap-v0.13.0-beta-linux-amd64.tar.gz
RUN cp -a snap-v0.13.0-beta/bin/. /usr/local/bin

# Create directory for snap plugins
RUN mkdir -p /opt/snap/plugins

# Download snap plugin pack release
RUN wget https://github.com/intelsdi-x/snap/releases/download/v0.13.0-beta/snap-plugins-v0.13.0-beta-linux-amd64.tar.gz
RUN tar -zxf snap-plugins-v0.13.0-beta-linux-amd64.tar.gz

# Install specific plugins
RUN cp -a snap-v0.13.0-beta/plugin/snap-plugin-collector-docker /opt/snap/plugins
RUN cp -a /go/src/github.com/intelsdi-x/snap-plugin-collector-users/build/rootfs/snap-plugin-collector-users /opt/snap/plugins

# Start snap daemon with auto discover option
EXPOSE 8181
ENTRYPOINT ["snapd", "-t", "0", "-a", "/opt/snap/plugins"]
