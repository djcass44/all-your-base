# Operating systems

Ayb supports the big 3 Linux variants; Alpine, Debian and Fedora.
In order to support package installation without privilege it can't use the native package managers directly, requiring Ayb to reimplement the package managers.
This can cause some issues and quirks.

## Known issues

### General

* Post-install scripting is not executed.

### Yum/RPM

* Installing an RPM package does not also install its dependencies (_this is being worked on_).
* Only the primary XML package list is searched when locating packages. The SQLite database format is not used.

### Debian

* Only XZ (`data.tar.xz`) and Zstd (`data.tar.zst`) data archives are supported.

### Alpine

* Installing an Alpine package with Ayb is not acknowledged by the native `apk` tool, which will believe the package has not been installed. We recommend avoiding mixing the two.
