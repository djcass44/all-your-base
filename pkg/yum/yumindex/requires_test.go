package yumindex

import (
	"context"
	_ "embed"
	"encoding/xml"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

//go:embed testdata/aa4c51a3ffefb98ce7e412a98df164f40bfc07c4bc659aa855eb10f791d03c16-primary.xml
var primaryDB string

func TestMetadata_GetProviders(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	var metadata Metadata
	require.NoError(t, xml.Unmarshal([]byte(primaryDB), &metadata))

	matches := metadata.GetProviders(ctx, []string{"libacl.so.1()(64bit)", "libacl.so.1(ACL_1.0)(64bit)", "libc.so.6()(64bit)", "libc.so.6(GLIBC_2.11)(64bit)", "libc.so.6(GLIBC_2.14)(64bit)", "libc.so.6(GLIBC_2.15)(64bit)", "libc.so.6(GLIBC_2.2.5)(64bit)", "libc.so.6(GLIBC_2.28)(64bit)", "libc.so.6(GLIBC_2.3)(64bit)", "libc.so.6(GLIBC_2.3.4)(64bit)", "libc.so.6(GLIBC_2.4)(64bit)", "libselinux.so.1()(64bit)", "libtinfo.so.6()(64bit)", "rtld(GNU_HASH)"}, nil)
	for _, m := range matches {
		t.Logf("match: %s=%s", m.Name, m.Version.Ver)
	}
	assert.NotEmpty(t, matches)
}
