build:
	docker build --no-cache=true --build-arg http_proxy=$(PROXY_HTTP) --build-arg=https_proxy=$(PROXY_HTTPS) -t snap-docker .

run:
	docker run --rm --pid=host --net=host -v /sys/fs/cgroup:/sys/fs/cgroup -v /var/run/docker.sock:/var/run/docker.sock --name snap -p 8181:8181 snap-docker $(ARGS)
