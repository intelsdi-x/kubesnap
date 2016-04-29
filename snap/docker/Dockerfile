FROM ubuntu
MAINTAINER mateusz.kleina@intel.com

# Install tools
RUN apt-get update
RUN apt-get -y install wget

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

# Start snap daemon with auto discover option
EXPOSE 8181
ENTRYPOINT ["snapd", "-t", "0", "-a", "/opt/snap/plugins"]
