package rpm

import (
	"context"
	"path/filepath"
	"testing"

	"chainguard.dev/apko/pkg/apk/fs"
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
	out := fs.DirFS(ctx, tempDir)

	pkg := &PackageKeeper{}

	t.Run("xz rpm", func(t *testing.T) {
		err := pkg.Unpack(ctx, "./testdata/git-core-2.39.3-1.el8_8.x86_64.rpm", out)
		assert.NoError(t, err)

		assert.DirExists(t, filepath.Join(tempDir, "usr", "bin"))
		assert.FileExists(t, filepath.Join(tempDir, "usr", "bin", "git"))
	})
	t.Run("gzip rpm", func(t *testing.T) {
		err := pkg.Unpack(ctx, "./testdata/cuda-cublas-10-0-10.0.130-1.x86_64.rpm", out)
		assert.NoError(t, err)

		assert.DirExists(t, filepath.Join(tempDir, "usr", "local", "cuda-10.0"))
		assert.FileExists(t, filepath.Join(tempDir, "usr", "local", "cuda-10.0", "targets", "x86_64-linux", "lib", "libcublas.so.10.0.130"))
	})
	t.Run("zstandard rpm", func(t *testing.T) {
		err := pkg.Unpack(ctx, "./testdata/perl-Net-SSLeay-1.94-3.el9.x86_64.rpm", out)
		assert.NoError(t, err)

		assert.DirExists(t, filepath.Join(tempDir, "usr", "lib64", "perl5"))
		assert.FileExists(t, filepath.Join(tempDir, "usr", "bin", "git"))
	})
}

func TestPackageKeeper_Resolve(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))
	pkg, err := NewPackageKeeper(ctx, []string{"https://cdn-ubi.redhat.com/content/public/ubi/dist/ubi8/8/x86_64/appstream/os"})
	require.NoError(t, err)

	packageNames, err := pkg.Resolve(ctx, "git")
	assert.NoError(t, err)
	t.Logf("%+v", packageNames)
}
