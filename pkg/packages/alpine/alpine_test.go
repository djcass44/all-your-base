package alpine

import (
	"context"
	"github.com/djcass44/all-your-base/pkg/packages"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

// interface guard
var _ packages.Package = &PackageKeeper{}

func TestPackageKeeper_Unpack(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	out := t.TempDir()

	pkg := &PackageKeeper{}
	err := pkg.Unpack(ctx, "./testdata/git-2.40.1-r0.apk", out)
	assert.NoError(t, err)

	assert.DirExists(t, filepath.Join(out, "var", "git"))
	assert.FileExists(t, filepath.Join(out, "usr", "bin", "git"))
}
