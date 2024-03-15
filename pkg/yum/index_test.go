package yum

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetMetadata(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	repoData, err := getMetadata(ctx, "https://cdn-ubi.redhat.com/content/public/ubi/dist/ubi8/8/x86_64/baseos/os")
	assert.NoError(t, err)
	assert.NotEmpty(t, repoData.Data)
}

func TestNewIndex(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	index, err := NewIndex(ctx, "https://cdn-ubi.redhat.com/content/public/ubi/dist/ubi8/8/x86_64/baseos/os")
	assert.NoError(t, err)
	assert.NotZero(t, index.Packages)
	assert.NotEmpty(t, index.Package)

	t.Logf("packages: %s", index.Packages)
}
