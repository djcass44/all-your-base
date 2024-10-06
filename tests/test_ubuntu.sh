#!/usr/bin/env bash

set -eux

go run main.go lock --v=1 -c tests/fixtures/ubuntu_vfs_full.yaml
go run main.go build --v=1 -c tests/fixtures/ubuntu_vfs_full.yaml --save /tmp/test.tar
docker load < /tmp/test.tar
docker run ubuntu-vfs-full