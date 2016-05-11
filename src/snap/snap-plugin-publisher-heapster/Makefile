default:
	$(MAKE) deps
	$(MAKE) all
deps:
	bash -c "godep restore"
all:
	bash -c "./scripts/build.sh $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))"

