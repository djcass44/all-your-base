package rpm

import (
	"context"
	"github.com/chainguard-dev/go-apk/pkg/fs"
	"github.com/djcass44/all-your-base/pkg/packages"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

// interface guard
var _ packages.PackageManager = &PackageKeeper{}

func TestPackageKeeper_Unpack(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	tempDir := t.TempDir()
	out := fs.DirFS(tempDir)

	pkg := &PackageKeeper{}
	err := pkg.Unpack(ctx, "./testdata/git-lfs-3.2.0-2.el8.x86_64.rpm", out)
	assert.NoError(t, err)

	assert.DirExists(t, filepath.Join(tempDir, "usr", "bin"))
	assert.FileExists(t, filepath.Join(tempDir, "usr", "bin", "git-lfs"))
}

func TestPackageKeeper_Resolve(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))
	pkg, err := NewPackageKeeper(ctx, []string{"https://cdn-ubi.redhat.com/content/public/ubi/dist/ubi8/8/x86_64/appstream/os"})
	require.NoError(t, err)

	packageNames, err := pkg.Resolve(ctx, "git-lfs")
	assert.NoError(t, err)
	t.Logf("%+v", packageNames)
}
