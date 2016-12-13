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
ENV DOCKER_VERSION 1.10.3
ENV DOCKER_SHA256 d0df512afa109006a450f41873634951e19ddabf8c7bd419caeb5a526032d86d

RUN curl -fSL "https://${DOCKER_BUCKET}/builds/Linux/x86_64/docker-$DOCKER_VERSION" -o /usr/local/bin/docker \
	&& echo "${DOCKER_SHA256}  /usr/local/bin/docker" | sha256sum -c - \
	&& chmod +x /usr/local/bin/docker

EXPOSE 8181

ENV SNAP_LOG_LEVEL=1
ENV SNAP_TRUST_LEVEL=0

COPY snapteld.conf /etc/snap/snapteld.conf
CMD /usr/local/sbin/snapteld -l ${SNAP_LOG_LEVEL} -t ${SNAP_TRUST_LEVEL} -o ''
