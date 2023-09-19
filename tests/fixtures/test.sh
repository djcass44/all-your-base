#!/usr/bin/env bash

set -eux

for f in *.yaml; do
  echo "Testing fixture: $f"
  go run ../../main.go lock -c "$f"
  go run ../../main.go build --save /tmp/test.tar -c "$f"
done