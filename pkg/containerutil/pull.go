package containerutil

import (
	"context"
	"fmt"
	"github.com/djcass44/all-your-base/pkg/empty"
	"github.com/djcass44/ci-tools/pkg/ociutil"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func Pull(ctx context.Context, ref string) (v1.Image, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("ref", ref)
	log.Info("pulling image")

	if ref == MagicImageScratch {
		return empty.Image, nil
	}
	// pull the image
	img, err := crane.Pull(ref, crane.WithContext(ctx), crane.WithAuthFromKeychain(ociutil.KeyChain(ociutil.Auth{})))
	if err != nil {
		return nil, fmt.Errorf("pulling %s: %w", ref, err)
	}
	return img, nil
}
