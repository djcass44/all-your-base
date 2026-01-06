package alpine

import (
	"chainguard.dev/apko/pkg/apk/apk"
	"chainguard.dev/apko/pkg/apk/fs"
	ociv1 "github.com/google/go-containerregistry/pkg/v1"
)

type PackageKeeper struct {
	rootfs  fs.FullFS
	indices []apk.NamedIndex
	base    ociv1.Image
}
