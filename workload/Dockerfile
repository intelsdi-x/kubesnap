FROM ubuntu

RUN apt-get update
RUN apt-get install -y git stress curl

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

COPY workload /go/src/workload
WORKDIR /go/src/workload

ENV GOMAXPROCS=1
RUN go get github.com/gorilla/mux
RUN go build

EXPOSE 7777
ENTRYPOINT ["/go/src/workload/workload"]
