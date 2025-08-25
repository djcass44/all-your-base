package containerutil

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/Snakdy/container-build-engine/pkg/oci/auth"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// Push is a minor modification of the container-build-engine containers.Push function
// that allows for custom certificates.
func Push(ctx context.Context, img v1.Image, dst string, certPool *x509.CertPool) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("ref", dst)
	log.Info("pushing image")

	// tweak the default transport so that we
	// can provide a custom certPool
	transport := remote.DefaultTransport.(*http.Transport).Clone()
	transport.ForceAttemptHTTP2 = true
	transport.TLSClientConfig.ClientCAs = certPool

	// push the image
	if err := crane.Push(img, dst, crane.WithContext(ctx), crane.WithAuthFromKeychain(auth.KeyChain(auth.Auth{})), crane.WithTransport(transport)); err != nil {
		log.Error(err, "failed to push image")
		return err
	}
	// parse what we just pushed
	ref, err := name.ParseReference(dst)
	if err != nil {
		log.Error(err, "failed to parse reference")
		return err
	}
	d, err := img.Digest()
	if err != nil {
		log.Error(err, "failed to read digest")
		return err
	}
	fmt.Println(ref.String() + "@" + d.String())
	return nil
}
