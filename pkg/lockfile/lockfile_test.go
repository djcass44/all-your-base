package lockfile

import (
	v1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLock_Validate(t *testing.T) {
	var cases = []struct {
		name string
		cfg  v1.BuildSpec
		ok   bool
	}{
		{
			name: "matching spec",
			cfg: v1.BuildSpec{
				Packages: []v1.Package{
					{
						Names: []string{"test-package"},
					},
				},
				Files: []v1.File{
					{
						URI: "test-file",
					},
				},
			},
			ok: true,
		},
		{
			name: "extra package",
			cfg: v1.BuildSpec{
				Packages: []v1.Package{
					{
						Names: []string{"test-package", "fake-package"},
					},
				},
				Files: []v1.File{
					{
						URI: "test-file",
					},
				},
			},
			ok: false,
		},
		{
			name: "extra file",
			cfg: v1.BuildSpec{
				Packages: []v1.Package{
					{
						Names: []string{"test-package"},
					},
				},
				Files: []v1.File{
					{
						URI: "test-file",
					},
					{
						URI: "https://example.com/file.tgz",
					},
				},
			},
			ok: false,
		},
		{
			name: "extra file in lock",
			cfg: v1.BuildSpec{
				Packages: []v1.Package{
					{
						Names: []string{"test-package"},
					},
				},
				Files: []v1.File{},
			},
			ok: false,
		},
		{
			name: "extra package in lock",
			cfg: v1.BuildSpec{
				Packages: []v1.Package{},
				Files: []v1.File{
					{
						URI: "test-file",
					},
				},
			},
			ok: true,
		},
	}

	lock := &Lock{
		Packages: map[string]Package{
			"test-package": {},
			"test-file": {
				Type: v1.PackageFile,
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := lock.Validate(tt.cfg)
			if !tt.ok {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

}

func TestLock_SortedKeys(t *testing.T) {
	l := &Lock{
		Packages: map[string]Package{
			"packageA": {},
			"packageC": {},
			"packageB": {},
		},
	}

	outOne := l.SortedKeys()
	outTwo := l.SortedKeys()
	assert.ElementsMatch(t, outOne, outTwo)
}
