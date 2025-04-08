package requestutil

import (
	"fmt"
	"github.com/carlmjohnson/requests"
	"github.com/go-logr/logr"
	"github.com/mholt/archives"
	"io"
	"net/http"
)

func WithGzip(out io.Writer) requests.ResponseHandler {
	return func(response *http.Response) error {
		log := logr.FromContextOrDiscard(response.Request.Context())
		var stream io.ReadCloser

		// if it's a gzip response, decompress it
		if response.Header.Get("Content-Type") == "application/gzip" {
			log.V(8).Info("decompressing gzip response")
			dec, err := archives.Gz{}.OpenReader(response.Body)
			if err != nil {
				return fmt.Errorf("decompressing: %w", err)
			}
			stream = dec
		} else {
			stream = response.Body
		}

		_, err := io.Copy(out, stream)
		if err != nil {
			return fmt.Errorf("writing uncompressed output: %w", err)
		}
		return nil
	}
}
