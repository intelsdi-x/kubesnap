FROM ubuntu
MAINTAINER mateusz.kleina@intel.com

# Install stress tool
RUN apt-get update
RUN apt-get -y install stress

# Start stress
ENTRYPOINT ["stress"]
