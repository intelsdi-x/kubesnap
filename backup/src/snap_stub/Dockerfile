FROM golang:1.6.0

RUN apt-get install -y git

RUN mkdir /main
COPY main /main

ENV GOMAXPROCS=2
WORKDIR /main
RUN go get github.com/gorilla/mux
RUN go build

EXPOSE 8777
ENTRYPOINT ["/main/main"]
