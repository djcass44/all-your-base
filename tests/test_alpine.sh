#!/usr/bin/env bash

set -eux

go run main.go lock -c tests/fixtures/alpine_318_full.yaml
go run main.go build -c tests/fixtures/alpine_318_full.yaml --save /tmp/test.tar
docker load < /tmp/test.tar
docker run alpine-318-full