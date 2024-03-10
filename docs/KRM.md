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

Rather than starting from *scratch*, we can build on an existing image.

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


**Fedora/UBI**

Repositories are referenced by the full URL of the directory containing the `Packages` and `repodata` directories.

```yaml
apiVersion: ayb.dcas.dev/v1
kind: Build
metadata:
  name: my-image
spec:
  from: registry.access.redhat.com/ubi9/ubi-minimal
  repositories:
    rpm:
      - url: https://cdn-ubi.redhat.com/content/public/ubi/dist/ubi8/8/x86_64/appstream/os
      - url: https://cdn-ubi.redhat.com/content/public/ubi/dist/ubi8/8/x86_64/baseos/os
```

## Packages

The packages property is a list of type and name groups.

Alpine example:

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
  packages:
    - type: Alpine
      names:
        - git
```

Debian example:

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
  packages:
    - type: Debian
      names:
        - git
```

UBI example:

```yaml
apiVersion: ayb.dcas.dev/v1
kind: Build
metadata:
  name: my-image
spec:
  from: registry.access.redhat.com/ubi9/ubi-minimal
  repositories:
    rpm:
      - url: https://cdn-ubi.redhat.com/content/public/ubi/dist/ubi8/8/x86_64/appstream/os
      - url: https://cdn-ubi.redhat.com/content/public/ubi/dist/ubi8/8/x86_64/baseos/os
  packages:
    - type: RPM
      names:
        - python3
```

While you could install packages that are provided by multiple package manager types (e.g. Alpine and Debian) in the same image, we don't recommend it.

## Files

Files can be included in the image.

```yaml
apiVersion: ayb.dcas.dev/v1
kind: Build
metadata:
  name: example
spec:
  from: scratch
  files:
    - uri: "" # any URI supported by the https://github.com/hashicorp/go-getter project. Could be a local file, HTTPS URL, S3 bucket, etc
      path: "" # folder to place downloaded file (or files if it's an archive). If downloading a raw file, you can also specify the file name
      executable: false # whether to make the file executable. Only works on single files
      subPath: "" # if unpacking an archive, extracts an individual file
```

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
