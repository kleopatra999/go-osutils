FROM golang:1.4.2
MAINTAINER peter.edge@gmail.com

RUN mkdir -p /go/src/github.com/peter-edge/go-osutils
ADD . /go/src/github.com/peter-edge/go-osutils/
WORKDIR /go/src/github.com/peter-edge/go-osutils
