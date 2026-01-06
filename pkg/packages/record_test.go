package packages

import (
	"context"
	"testing"

	"chainguard.dev/apko/pkg/apk/fs"
	"github.com/Snakdy/container-build-engine/pkg/oci/empty"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecord(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	type Package struct {
		Name string
	}
	var RecordFile = "packages.txt"

	pkg := []Package{
		{
			Name: "foo",
		},
		{
			Name: "bar",
		},
	}

	t.Run("packages are recorded when there is no existing record", func(t *testing.T) {
		testfs := fs.NewMemFS()
		err := Record(ctx, empty.Image, RecordFile, pkg, testfs, func(t Package) string {
			return t.Name
		})
		assert.NoError(t, err)

		out, err := testfs.ReadFile(RecordFile)
		assert.NoError(t, err)

		t.Logf("Record contains:\n%+v", string(out))

		assert.Contains(t, string(out), pkg[0].Name)
		assert.Contains(t, string(out), pkg[1].Name)
	})
	t.Run("packages are recorded when there is an existing record", func(t *testing.T) {
		testfs := fs.NewMemFS()
		require.NoError(t, testfs.WriteFile(RecordFile, []byte("foo\nzoo\n"), 0644))
		err := Record(ctx, empty.Image, RecordFile, pkg, testfs, func(t Package) string {
			return t.Name
		})
		assert.NoError(t, err)

		out, err := testfs.ReadFile(RecordFile)
		assert.NoError(t, err)

		t.Logf("Record contains:\n%+v", string(out))

		assert.Contains(t, string(out), pkg[0].Name)
		assert.Contains(t, string(out), pkg[1].Name)
		assert.Contains(t, string(out), "zoo")
	})
}
