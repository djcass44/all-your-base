package ca_certificates

import (
	"context"
	"github.com/Snakdy/container-build-engine/pkg/files"
	"github.com/Snakdy/container-build-engine/pkg/vfs"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUpdateCertificates(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	rootfs := vfs.NewVFS(t.TempDir())
	require.NoError(t, files.CopyDirectory(ctx, "./testdata", "/", rootfs))

	err := UpdateCertificates(ctx, rootfs)
	assert.NoError(t, err)
}
