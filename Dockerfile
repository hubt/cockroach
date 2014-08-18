# This builds a docker image with source cloned from github
# This is appropriate as a build server
# Do NOT use this docker image in a production setting.

FROM ubuntu

FROM ubuntu:14.04
MAINTAINER Shawn Morel <shawn@strangemonad.com>

# Setup the toolchain
RUN apt-get update
RUN apt-get install -y build-essential
RUN apt-get install -y curl git bzr mercurial
RUN curl -L -s http://golang.org/dl/go1.3.linux-amd64.tar.gz | tar -v -C /usr/local/ -xz

ENV PATH  /usr/local/go/bin:/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin
ENV GOPATH  /go
ENV GOROOT  /usr/local/go
ENV COCKROACH /go/src/github.com/cockroachdb
RUN mkdir -p $COCKROACH

RUN apt-get install -y php5 php5-curl
# install arcanist and libphutil(version from phacility not facebook)
RUN cd $COCKROACH && git clone https://github.com/phacility/libphutil.git
RUN cd $COCKROACH && git clone https://github.com/phacility/arcanist.git && ln -s $COCKROACH/arcanist/bin/arc /usr/bin

RUN apt-get install -y libsnappy-dev zlib1g-dev libbz2-dev libgflags-dev

VOLUME [ "/go/src/github.com/cockroachdb/cockroach" ]
