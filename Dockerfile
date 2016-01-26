FROM alpine:3.2
MAINTAINER Rohith <gambol99@gmail.com>

RUN apk update && \
    apk add ca-certificates

ADD bin/etcd-discovery /etcd-discovery

ENTRYPOINT [ "/etcd-discovery" ]