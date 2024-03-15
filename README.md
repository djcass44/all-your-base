# all-your-base: OCI image builder

*All your base* images are secure and verifiable.

ayb has the following key features:
* **Fully reproducible by default.** Run ayb twice and you will get the exact same image.
* **Small.** ayb generates images only contain what you tell it to contain.
* **Secure by default.** ayb configures images to run as a non-root user. ayb also requires no privileges to run.
* **Portable**. ayb has been designed to work in multiple environments, even those without internet.

## Quickstart

An ayb file for building a base image looks like this:

```yaml
apiVersion: ayb.dcas.dev/v1
kind: Build
metadata:
  name: my-base-image
spec:
  from: alpine:3.18
  repositories:
    alpine:
      - url: https://mirror.aarnet.edu.au/pub/alpine/v3.18/main
  packages:
    - type: Alpine
      names:
        - git
```

Have a look through the [`examples`](examples) directory for more.

We can build this with ayb from any environment:

```shell
# generate or update the lockfile
ayb lock  --config tests/fixtures/alpine_318_full.yaml
# build the image
ayb build --config tests/fixtures/alpine_318_full.yaml --save ayb-alpine.tar
```
You can then load the generated tar image into an OCI environment:

```shell
docker load < ayb-alpine.tar
```

You can also publish the image directly to a registry:

```shell
ayb build --config tests/fixtures/alpine_318_full.yaml --image myrepo/alpine318 --tag test --tag latest
```

## Documentation

Documentation can be found in the [`docs`](./docs) directory.
