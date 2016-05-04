FROM golang:1.6.0

RUN apt-get install -y git
RUN git clone https://github.com/kubernetes/heapster.git /go/src/k8s.io/heapster
RUN cd /go/src/k8s.io/heapster && git reset --hard de510e4bdcdea96722b5bde19ff0b7a142939485

COPY metrics/sources /go/src/k8s.io/heapster/metrics/sources

ENV GOMAXPROCS=1
RUN cd /go/src/k8s.io/heapster && make && mv heapster /heapster && mv eventer /eventer

ENTRYPOINT ["/heapster"]
