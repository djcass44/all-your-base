# Overrides

To enable multi-environment builds, Ayb supports overriding certain fields in the `build.yaml` file.
This can be useful when an image is built in a development environment and a corporate environment without direct internet access (e.g., behind a corporate firewall, air-gap).

## Parent image

The parent image can be overridden by including an environment variable:

```yaml
apiVersion: ayb.dcas.dev/v1
kind: Build
metadata:
  name: debian-bullseye-full
spec:
  from: ${MY_REGISTRY:docker.io}/library/debian:bullseye
```

By default, Ayb will resolve the `from` field to `docker.io/library/debian:bullseye`.
If you set the `MY_REGISTRY` environment variable to `registry.corporate.internal/docker.io`, Ayb will resolve it to `registry.corporate.internal/docker.io/library/debian:bullseye`.

Keep in mind that Ayb will still validate the digest of the image. If you are unable to retain digests you can use the `--skip-image-locking` flag when locking.

## Repositories & files

Repositories and file URLs can be overridden in the same fashion as the parent image:

```yaml
apiVersion: ayb.dcas.dev/v1
kind: Build
metadata:
  name: debian-bullseye-full
spec:
  from: ${MY_REGISTRY:docker.io}/library/debian:bullseye
  repositories:
    debian:
      - url: ${FOO_BAR_ZOO:-https://mirror.aarnet.edu.au/pub/debian} bullseye main
  packages:
    - type: Debian
      names:
        - git
        - g++
  files:
    - uri: ${MY_GNU_MIRROR:https://ftp.gnu.org}/gnu/hello/hello-2.12.tar.gz?checksum=cf04af86dc085268c5f4470fbae49b18afbc221b78096aab842d934a76bad0ab&archive=false
      path: /hello-2.12.tar.gz
  entrypoint:
    - /bin/bash
```