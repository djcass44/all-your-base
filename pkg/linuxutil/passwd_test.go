package linuxutil

import (
	"context"
	_ "embed"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"testing"
)

//go:embed testdata/existing
var expected string

//go:embed testdata/single
var expectedEmpty string

func TestNewUser(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	setup := func(s string) string {
		rootfs := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(rootfs, "etc"), 0755))

		f, err := os.Open(s)
		require.NoError(t, err)

		path := filepath.Join(rootfs, "etc", "passwd")
		out, err := os.Create(path)
		require.NoError(t, err)

		_, _ = io.Copy(out, f)
		return rootfs
	}

	t.Run("empty", func(t *testing.T) {
		rootfs := t.TempDir()

		err := NewUser(ctx, rootfs, "somebody", 1001)
		assert.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(rootfs, "etc", "passwd"))
		require.NoError(t, err)
		assert.EqualValues(t, expectedEmpty, string(data))
	})

	t.Run("normal", func(t *testing.T) {
		path := setup("./testdata/normal")

		err := NewUser(ctx, path, "somebody", 1001)
		assert.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(path, "etc", "passwd"))
		require.NoError(t, err)
		assert.EqualValues(t, expected, string(data))
	})
	t.Run("existing", func(t *testing.T) {
		path := setup("./testdata/existing")

		err := NewUser(ctx, path, "somebody", 1001)
		assert.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(path, "etc", "passwd"))
		require.NoError(t, err)
		assert.EqualValues(t, expected, string(data))
	})
}
