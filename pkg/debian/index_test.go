package debian

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewIndex(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))
	index, err := NewIndex(ctx, "https://mirror.aarnet.edu.au/pub/debian", "bullseye", "main", "amd64")
	assert.NoError(t, err)
	assert.NotZero(t, index.Count())
}

func TestIndex_GetPackageWithDependencies(t *testing.T) {
	ctx := logr.NewContext(context.TODO(), testr.NewWithOptions(t, testr.Options{Verbosity: 10}))

	idx, err := newIndex(ctx, "", "./testdata/Packages.gz")
	assert.NoError(t, err)

	t.Run("dependencies are found", func(t *testing.T) {
		out, err := idx.GetPackageWithDependencies(ctx, map[string]Package{}, &PackageVersion{
			Names:   []string{"0ad"},
			Version: "0.0.23.1-5+b1",
		})
		assert.NoError(t, err)
		assert.Len(t, out, 2)
	})
	t.Run("no dependents returns package", func(t *testing.T) {
		out, err := idx.GetPackageWithDependencies(ctx, map[string]Package{}, &PackageVersion{
			Names: []string{"0ad-data"},
		})
		assert.NoError(t, err)
		assert.Len(t, out, 1)
	})
}

func TestPackageMatchesConstraints(t *testing.T) {
	var cases = []struct {
		s1 string
		pv *PackageVersion
		ok bool
	}{
		{
			"0.0.23.1-1.1",
			&PackageVersion{
				Names:      []string{"0ad-data"},
				Version:    "0.0.23.1-5",
				Constraint: "<=",
			},
			true,
		},
		{
			"0.0.23.1-1.1",
			&PackageVersion{
				Names:      []string{"0ad-data"},
				Version:    "0.0.23.1",
				Constraint: ">=",
			},
			true,
		},
		{
			"",
			&PackageVersion{
				Names:   []string{"foo"},
				Version: "1.2.3",
			},
			true,
		},
		{
			"1.2.3",
			&PackageVersion{
				Names: []string{"foo"},
			},
			true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.s1, func(t *testing.T) {
			ok := tt.pv.Matches(tt.s1)
			assert.EqualValues(t, tt.ok, ok)
		})
	}
}
