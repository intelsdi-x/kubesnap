build:
	docker build --no-cache=true --build-arg http_proxy=$(PROXY_HTTP) --build-arg=https_proxy=$(PROXY_HTTPS) -t stress-docker .

run:
	docker run --rm --name stress stress-docker $(ARGS)
