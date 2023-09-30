package archiveutil

import (
	"context"
	"github.com/chainguard-dev/go-apk/pkg/fs"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestUnar(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	rootfs := fs.NewMemFS()

	f, err := os.Open("./testdata/test.ar")
	require.NoError(t, err)
	defer f.Close()

	err = Unar(ctx, f, rootfs)
	assert.NoError(t, err)

	_, err = rootfs.Stat("test.txt")
	assert.NotErrorIs(t, err, os.ErrNotExist)
}
