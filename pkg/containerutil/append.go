package containerutil

import (
	"context"
	"fmt"
	"github.com/djcass44/ci-tools/pkg/ociutil"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
)

func Append(ctx context.Context, appPath, baseRef string, platform *v1.Platform) (v1.Image, error) {
	log := logr.FromContextOrDiscard(ctx)
	// pull the base image
	log.Info("pulling base image", "base", baseRef)
	var base v1.Image
	var err error

	if baseRef == "scratch" {
		base = empty.Image
	} else {
		base, err = crane.Pull(baseRef, crane.WithContext(ctx), crane.WithAuthFromKeychain(ociutil.KeyChain(ociutil.Auth{})))
		if err != nil {
			return nil, fmt.Errorf("pulling %s: %w", baseRef, err)
		}
	}

	// create our new layer
	log.Info("containerising directory", "path", appPath)
	layer, err := NewLayer(appPath, platform)
	if err != nil {
		return nil, err
	}

	// append our layer
	layers := []mutate.Addendum{
		{
			Layer: layer,
			History: v1.History{
				Author:    "all-your-base",
				CreatedBy: "all-your-base build",
				Created:   v1.Time{},
			},
		},
	}
	withData, err := mutate.Append(base, layers...)
	if err != nil {
		return nil, fmt.Errorf("appending layers: %w", err)
	}
	// grab a copy of the base image's config file, and set
	// our entrypoint and env vars
	cfg, err := withData.ConfigFile()
	if err != nil {
		return nil, err
	}
	cfg = cfg.DeepCopy()
	cfg.Author = "github.com/djcass44/all-your-base"
	cfg.Config.WorkingDir = "/"
	if cfg.Config.Labels == nil {
		cfg.Config.Labels = map[string]string{}
	}

	// package everything up
	img, err := mutate.ConfigFile(withData, cfg)
	if err != nil {
		return nil, err
	}
	return img, nil
}
