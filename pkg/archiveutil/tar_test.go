package archiveutil

import (
	"chainguard.dev/apko/pkg/apk/fs"
	"context"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestUntar(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	rootfs := fs.NewMemFS()

	f, err := os.Open("./testdata/test.tar")
	require.NoError(t, err)
	defer f.Close()

	err = Untar(ctx, f, rootfs)
	assert.NoError(t, err)

	_, err = rootfs.Stat("test.txt")
	assert.NotErrorIs(t, err, os.ErrNotExist)

	_, err = rootfs.Stat("test-symbolic.txt")
	assert.NotErrorIs(t, err, os.ErrNotExist)
}
