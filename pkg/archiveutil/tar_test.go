package archiveutil

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestUntar(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	out := t.TempDir()

	f, err := os.Open("./testdata/test.tar")
	require.NoError(t, err)
	defer f.Close()

	err = Untar(ctx, f, out)
	assert.NoError(t, err)

	assert.FileExists(t, filepath.Join(out, "test.txt"))
}
