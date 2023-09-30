package debian

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseVersion(t *testing.T) {
	var cases = []struct {
		in  string
		out *PackageVersion
		ok  bool
	}{
		{
			"0ad-data (<= 0.0.23.1-5)",
			&PackageVersion{
				Names:      []string{"0ad-data"},
				Version:    "0.0.23.1-5",
				Constraint: "<=",
			},
			true,
		},
		{
			"foo [i386]",
			&PackageVersion{
				Names:      []string{"foo"},
				Version:    "",
				Constraint: "",
			},
			true,
		},
		{
			"foo | bar",
			&PackageVersion{
				Names:      []string{"foo", "bar"},
				Version:    "",
				Constraint: "",
			},
			true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.in, func(t *testing.T) {
			out, err := ParseVersion(tt.in)
			if tt.ok {
				assert.NoError(t, err)
				assert.EqualValues(t, tt.out, out)
				return
			}
			assert.Error(t, err)
		})
	}
}
