package requestutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsGzipped(t *testing.T) {
	var cases = []struct {
		s  string
		ok bool
	}{
		{
			"application/gzip",
			true,
		},
		{
			"application/x-gzip",
			true,
		},
		{
			"application/javascript",
			false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.s, func(t *testing.T) {
			ok := isGzipped(tt.s)
			assert.EqualValues(t, tt.ok, ok)
		})
	}
}
