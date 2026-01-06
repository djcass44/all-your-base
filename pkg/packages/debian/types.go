package debian

import (
	"chainguard.dev/apko/pkg/apk/fs"
	"github.com/djcass44/all-your-base/pkg/debian"
	ociv1 "github.com/google/go-containerregistry/pkg/v1"
)

type PackageKeeper struct {
	rootfs  fs.FullFS
	indices []*debian.Index
	base    ociv1.Image
}
