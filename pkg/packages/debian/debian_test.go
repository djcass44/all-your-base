package debian

import (
	"context"
	"path/filepath"
	"testing"

	"chainguard.dev/apko/pkg/apk/fs"
	"github.com/Snakdy/container-build-engine/pkg/containers"
	"github.com/djcass44/all-your-base/pkg/packages"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// interface guard
var _ packages.PackageManager = &PackageKeeper{}

func TestPackageKeeper_Unpack(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	tempDir := t.TempDir()
	out := fs.DirFS(tempDir)

	pkg := &PackageKeeper{}
	err := pkg.Unpack(ctx, "./testdata/git-lfs_3.4.0-1+b1_amd64.deb", out)
	assert.NoError(t, err)

	assert.DirExists(t, filepath.Join(tempDir, "usr", "bin"))
	assert.FileExists(t, filepath.Join(tempDir, "usr", "bin", "git-lfs"))
}

func TestPackageKeeper_Resolve(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))
	testfs := fs.NewMemFS()

	baseImage, err := containers.Get(ctx, "harbor.dcas.dev/docker.io/library/debian:bullseye")
	require.NoError(t, err)

	pkg, err := NewPackageKeeper(ctx, []string{"https://mirror.aarnet.edu.au/pub/debian bullseye main"}, testfs, baseImage)
	require.NoError(t, err)

	packageNames, err := pkg.Resolve(ctx, "git")
	assert.NoError(t, err)
	t.Logf("%+v", packageNames)

	for _, i := range installedFiles {
		out, err := testfs.ReadFile(i)
		require.NoError(t, err)
		t.Logf("%+v", string(out))
	}
}
