# all-your-base: OCI image builder

*All your base* images builds secure and verifiable base images.

## Quickstart

An ayb file for building a base image looks like this:

```yaml
apiVersion: ayb.dcas.dev/v1
kind: Build
metadata:
  name: my-base-image
spec:
  from: alpine:3.18
  packages:
    - git
```

We can build this with ayb from any environment:

```shell
ayb build tests/fixtures/alpine_318_full.yaml --save ayb-alpine.tar
```
You can then load the generated tar image into an OCI environment:

```shell
docker load < ayb-alpine.tar
```

You can also publish the image directly to a registry:

```shell
ayb build tests/fixtures/alpine_318_full.yaml --image myrepo/alpine318 -t test
```