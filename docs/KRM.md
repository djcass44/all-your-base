# Resource model

`all-your-base` uses the [Kubernetes Resource Model](https://github.com/kubernetes/design-proposals-archive/blob/main/architecture/resource-management.md) (KRM) to describe a build.

## Metadata

Since we're using KRM, we need the basics:

```yaml
apiVerison: ayb.dcas.dev/v1
kind: Build
metadata:
  name: my-image
```

## Base image

Rather than starting from *scratch*, we can build off of an existing image.

```yaml
apiVerison: ayb.dcas.dev/v1
kind: Build
metadata:
  name: my-image
spec:
  from: alpine:3.18
```

Or, we can start from scratch:

```yaml
apiVerison: ayb.dcas.dev/v1
kind: Build
metadata:
  name: my-image
spec:
  from: scratch
```

The `scratch` image is a magic image that doesn't actually exist.
When we use it, it means we are starting from nothing.

For more information about how it works, read the [Docker](https://hub.docker.com/_/scratch/) documentation.

## Repositories

**Alpine**

When referencing an Alpine repository, include everything before the `APKINDEX.tar.gz` file until the architecture.

```yaml
apiVerison: ayb.dcas.dev/v1
kind: Build
metadata:
  name: my-image
spec:
  from: alpine:3.18
  repositories:
    alpine:
      - uri: https://mirror.aarnet.edu.au/pub/alpine/v3.18/main
      - uri: https://mirror.aarnet.edu.au/pub/alpine/v3.18/community
```

**Debian**

Debian repositories follow the normal Debian repository format that you would find in a `sources.list` file.

```yaml
apiVerison: ayb.dcas.dev/v1
kind: Build
metadata:
  name: my-image
spec:
  from: debian:bullseye
  repositories:
    debian:
      - uri: "https://mirror.aarnet.edu.au/pub/debian bullseye main"
      - uri: "https://mirror.aarnet.edu.au/pub/debian-security bullseye-security main"
```

## Packages

TODO

## Files

TODO

## Links

Links allow you to create arbitrary symbolic links.

```yaml
apiVerison: ayb.dcas.dev/v1
kind: Build
metadata:
  name: my-image
spec:
  links:
    - source: /usr/bin/sh
      target: /bin/sh
```

The above example fixes compatability issues where the `sh` binary is placed in `/usr/bin/sh` when some applications expect it to be in `/bin/sh`.

## Environment variables

Environment variables are fairly standard and has been designed to look pretty much the same as what you would expect in a Kubernetes resource definition.

```yaml
apiVerison: ayb.dcas.dev/v1
kind: Build
metadata:
  name: my-image
spec:
  env:
    - name: FOO
      value: "BAR"
```

## Command/Entrypoint

By default, `all-your-base` will rewrite the entrypoint to `/bin/sh` and nullify the command.

Both the entrypoint and command can be set to custom values:

```yaml
apiVerison: ayb.dcas.dev/v1
kind: Build
metadata:
  name: my-image
spec:
  entrypoint:
    - /bin/bash
    - -c
  command:
    - /some/magic/application.sh
```
