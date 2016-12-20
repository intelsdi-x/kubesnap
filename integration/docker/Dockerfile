FROM ubuntu:16.04
MAINTAINER Maria Rolbiecka maria.a.rolbiecka@intel.com

RUN apt-get update && apt-get install -y curl

ENV SNAP_VERSION 1.0.0
ENV SNAP_DOWNLOAD_URL https://github.com/intelsdi-x/snap/releases/download/${SNAP_VERSION}/snap-${SNAP_VERSION}-linux-amd64.tar.gz
ENV SNAP_DIR /snap

WORKDIR ${SNAP_DIR}

RUN curl -sfL "$SNAP_DOWNLOAD_URL" -o snap-telemetry.tar.gz \
    && tar xf snap-telemetry.tar.gz \
    && rm snap-telemetry.tar.gz \
    && cp snapteld /usr/local/sbin \
    && cp snaptel /usr/local/bin

ENV DOCKER_BUCKET get.docker.com
ENV DOCKER_VERSION 1.12.4
ENV DOCKER_SHA256 f7cb7bb55d6ceba3ba3d24d62027e84799763b4c41b0bda5d8d5b9ba31ed0f2f 

RUN curl -fSL "https://${DOCKER_BUCKET}/builds/Linux/x86_64/docker-$DOCKER_VERSION.tgz" -o /usr/local/bin/docker \
	&& echo "${DOCKER_SHA256}  /usr/local/bin/docker" | sha256sum -c - \
	&& chmod +x /usr/local/bin/docker

RUN mkdir /var/log/snap

EXPOSE 8181 8777

COPY snapteld.conf /etc/snap/snapteld.conf
CMD /usr/local/sbin/snapteld 
