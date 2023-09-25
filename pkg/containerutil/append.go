package containerutil

import (
	"context"
	"fmt"
	"github.com/chainguard-dev/go-apk/pkg/fs"
	"github.com/go-logr/logr"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"path/filepath"
	"strings"
)

const MagicImageScratch = "scratch"
const DefaultPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/home/somebody/.local/bin"
const DefaultUsername = "somebody"

type Image struct {
	author     string
	username   string
	env        []string
	baseImage  v1.Image
	entrypoint []string
	cmd        []string
}

func NewImage(opts ...ImageOption) *Image {
	img := &Image{
		author:    "github.com/djcass44/all-your-base",
		env:       nil,
		baseImage: empty.Image,
		username:  DefaultUsername,
	}

	for _, opt := range opts {
		opt(img)
	}

	return img
}

type ImageOption func(image *Image)

func WithBaseImage(img v1.Image) ImageOption {
	return func(image *Image) {
		image.baseImage = img
	}
}

func WithEnv(env ...string) ImageOption {
	return func(image *Image) {
		image.env = env
	}
}

func WithUsername(s string) ImageOption {
	return func(image *Image) {
		image.username = s
	}
}

func WithEntrypoint(ep, cmd []string) ImageOption {
	return func(image *Image) {
		image.entrypoint = ep
		image.cmd = cmd
	}
}

func (ib *Image) Append(ctx context.Context, fs fs.FullFS, platform *v1.Platform) (v1.Image, error) {
	log := logr.FromContextOrDiscard(ctx)

	// create our new layer
	log.Info("containerising filesystem")
	layer, err := NewLayer(ctx, fs, platform)
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
	withData, err := mutate.Append(ib.baseImage, layers...)
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
	cfg.Author = ib.author
	cfg.Config.WorkingDir = filepath.Join("/home", ib.username)
	cfg.Config.User = ib.username

	if ib.entrypoint != nil {
		log.Info("overriding entrypoint", "before", cfg.Config.Entrypoint, "after", ib.entrypoint)
		cfg.Config.Entrypoint = ib.entrypoint
	}
	if ib.cmd != nil {
		log.Info("overriding command", "before", cfg.Config.Cmd, "after", ib.cmd)
		cfg.Config.Cmd = ib.cmd
	}

	cfg.Config.Env = ib.env

	var found bool
	for i, e := range cfg.Config.Env {
		if strings.HasPrefix(e, "PATH=") {
			cfg.Config.Env[i] = cfg.Config.Env[i] + fmt.Sprintf(":/home/%s/.local/bin", ib.username)
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
