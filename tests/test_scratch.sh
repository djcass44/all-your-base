#!/usr/bin/env bash

set -eux

BUILD_CONFIG="scratch_airgap_test.yaml"

go run main.go lock -c "tests/fixtures/$BUILD_CONFIG" --v=2
go run main.go build -c "tests/fixtures/$BUILD_CONFIG" --save /tmp/test.tar --v=2
docker load < /tmp/test.tar
docker run -it scratch-airgap-test