package statements

import (
	"chainguard.dev/apko/pkg/apk/fs"
	"context"
	cbev1 "github.com/Snakdy/container-build-engine/pkg/api/v1"
	"github.com/Snakdy/container-build-engine/pkg/pipelines"
	"github.com/djcass44/all-your-base/pkg/downloader"
	"github.com/djcass44/all-your-base/pkg/packages/debian"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPackageStatement_Run(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))
	pkg, err := debian.NewPackageKeeper(ctx, []string{"https://mirror.aarnet.edu.au/pub/debian bullseye main"})
	require.NoError(t, err)

	packageNames, err := pkg.Resolve(ctx, "openjdk-17-jdk")
	require.NoError(t, err)

	dl, err := downloader.NewDownloader(t.TempDir())
	require.NoError(t, err)

	rootfs := fs.NewMemFS()

	bctx := &pipelines.BuildContext{
		Context:          ctx,
		WorkingDirectory: t.TempDir(),
		FS:               rootfs,
		ConfigFile:       nil,
	}

	for _, pkgName := range packageNames {
		s := NewPackageStatement(nil, pkg, nil, dl)
		s.SetOptions(cbev1.Options{
			"type":     string(pkgName.Type),
			"name":     pkgName.Name,
			"version":  pkgName.Version,
			"resolved": pkgName.Resolved,
		})
		_, err := s.Run(bctx)
		assert.NoError(t, err)
	}
}
