package alpine

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
	err := pkg.Unpack(ctx, "./testdata/git-2.40.1-r0.apk", out)
	assert.NoError(t, err)

	assert.DirExists(t, filepath.Join(tempDir, "var", "git"))
	assert.FileExists(t, filepath.Join(tempDir, "usr", "bin", "git"))
}

func TestPackageKeeper_Resolve(t *testing.T) {
	testfs := fs.NewMemFS()

	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))
	pkg, err := NewPackageKeeper(ctx, []string{"https://mirror.aarnet.edu.au/pub/alpine/v3.18/main"}, testfs)
	require.NoError(t, err)

	packageNames, err := pkg.Resolve(ctx, "git")
	assert.NoError(t, err)
	t.Logf("%+v", packageNames)

	out, err := testfs.ReadFile(installedFile)
	require.NoError(t, err)

	t.Logf("%+v", string(out))
}
