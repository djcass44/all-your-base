package requestutil

import (
	"fmt"
	"io"
	"net/http"

	"github.com/carlmjohnson/requests"
	"github.com/gabriel-vasile/mimetype"
	"github.com/go-logr/logr"
	"github.com/mholt/archives"
)

const ContentTypeGzip = "application/gzip"

func WithGzip(out io.Writer) requests.ResponseHandler {
	return func(response *http.Response) error {
		log := logr.FromContextOrDiscard(response.Request.Context())
		var stream io.ReadCloser

		// if it's a gzip response, decompress it
		if mimetype.EqualsAny(response.Header.Get("Content-Type"), ContentTypeGzip) {
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
