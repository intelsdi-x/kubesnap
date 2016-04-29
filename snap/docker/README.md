# snap in Docker container

This is complete working Dockerfile for snap telemetry framework.

## Requirements

- Linux system
- Docker Engine installed. You can read [Install Docker Engine](https://docs.docker.com/engine/installation) page for instructions.

## Setup
To simplify Docker setup, there is Makefile included. You don't need to type Docker commands each time to get it work.

### Build
To download and build working snap Docker image, launch:
	
	$ git clone https://github.com/intelsdi-x/kubesnap
	$ cd kubesnap/snap/docker
	$ make PROXY_HTTP=<http_proxy> PROXY_HTTPS=<https_proxy>

You can pass your system wide $HTTP_PROXY and $HTTPS_PROXY environment variables to proxy arguments or skip it at all.

### Run

To start prepared image, simply type:

	$ make run

by default, snap daemon starts with:
	
	$ snapd -t 0 -a /opt/snap/plugins

if you want to override these arguments, run:

	$ make run ARGS=<snapd_args>

for example:

	$ make run ARGS="-t 1 --tribe"

### Manage snap plugins remotely

snap has REST API, which allows to manage plugins and tasks inside snap Docker container remotely. For more information, read [snap API](https://github.com/mkleina/snap/blob/master/docs/REST_API.md#plugin-api).

### Built-in plugins
This Docker image includes all plugins from snap plugins pack release. By default, there is only snap-plugin-collector-docker installed. If you want more, just modify Dockerfile and rebuild image:

	# Install specific plugins
	RUN cp -a snap-v0.13.0-beta/plugin/snap-plugin-collector-docker /opt/snap/plugins
	RUN cp -a snap-v0.13.0-beta/plugin/<plugin_to_install> /opt/snap/plugins

## Author

[Mateusz Kleina](https://github.com/mkleina)
