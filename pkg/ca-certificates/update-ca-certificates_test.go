package ca_certificates

import (
	"context"
	"github.com/Snakdy/container-build-engine/pkg/files"
	"github.com/chainguard-dev/go-apk/pkg/fs"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUpdateCertificates(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	rootfs := fs.NewMemFS()
	require.NoError(t, files.CopyDirectory("./testdata", "/", rootfs))

	err := UpdateCertificates(ctx, rootfs)
	assert.NoError(t, err)
}
