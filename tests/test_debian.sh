#!/usr/bin/env bash

set -eux

go run main.go lock -c tests/fixtures/debian_bookworm_full.yaml
go run main.go build -c tests/fixtures/debian_bookworm_full.yaml --save /tmp/test.tar
docker load < /tmp/test.tar
docker run debian-bookworm-full