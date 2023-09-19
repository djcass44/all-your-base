package containerutil

import (
	"context"
	"fmt"
	"github.com/chainguard-dev/go-apk/pkg/fs"
	"github.com/djcass44/ci-tools/pkg/ociutil"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"path/filepath"
	"strings"
)

const MagicImageScratch = "scratch"
const DefaultPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/home/somebody/.local/bin"

func Append(ctx context.Context, fs fs.FullFS, baseRef string, platform *v1.Platform, username string) (v1.Image, error) {
	log := logr.FromContextOrDiscard(ctx)
	// pull the base image
	log.Info("pulling base image", "base", baseRef)
	var base v1.Image
	var err error

	if baseRef == MagicImageScratch {
		base = empty.Image
	} else {
		base, err = crane.Pull(baseRef, crane.WithContext(ctx), crane.WithAuthFromKeychain(ociutil.KeyChain(ociutil.Auth{})))
		if err != nil {
			return nil, fmt.Errorf("pulling %s: %w", baseRef, err)
		}
	}

	// create our new layer
	log.Info("containerising filesystem")
	layer, err := NewLayer(fs, platform)
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
	cfg.Config.WorkingDir = filepath.Join("/home", username)
	cfg.Config.User = username

	var found bool
	for i, e := range cfg.Config.Env {
		if strings.HasPrefix(e, "PATH=") {
			cfg.Config.Env[i] = cfg.Config.Env[i] + ":/home/somebody/.local/bin"
			found = true
		}
	}
	if !found {
		cfg.Config.Env = append(cfg.Config.Env, "PATH="+DefaultPath)
	}
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
