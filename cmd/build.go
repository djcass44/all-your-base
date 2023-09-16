package cmd

import (
	"context"
	"fmt"
	aybv1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/containerutil"
	"github.com/djcass44/all-your-base/pkg/packages"
	"github.com/djcass44/all-your-base/pkg/packages/alpine"
	"github.com/go-logr/logr"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/hashicorp/go-getter"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"path/filepath"
	"strings"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build an image",
	RunE:  build,
}

const (
	flagConfig = "config"
	flagSave   = "save"
)

func init() {
	buildCmd.Flags().StringP(flagConfig, "c", "", "path to an image configuration file")
	buildCmd.Flags().String(flagSave, "", "path to save the image as a tar archive")

	_ = buildCmd.MarkFlagRequired(flagConfig)
	_ = buildCmd.MarkFlagFilename(flagConfig, ".yaml", ".yml")
}

func build(cmd *cobra.Command, _ []string) error {
	log := logr.FromContextOrDiscard(cmd.Context())

	configPath, _ := cmd.Flags().GetString(flagConfig)
	localPath, _ := cmd.Flags().GetString(flagSave)

	// read the config file
	cfg, err := readConfig(configPath)
	if err != nil {
		return err
	}

	rootfs, err := os.MkdirTemp("", fmt.Sprintf("%s-*", cfg.Name))
	if err != nil {
		return err
	}
	log.V(3).Info("prepared root filesystem", "path", rootfs)

	alpineKeeper, err := alpine.NewPackageKeeper(cmd.Context(), repoURLs(cfg.Spec.Repositories[strings.ToLower(string(aybv1.PackageAlpine))]))

	// install packages
	for _, pkg := range cfg.Spec.Packages {
		var keeper packages.Package
		ext := filepath.Ext(pkg.URL)
		switch pkg.Type {
		case aybv1.PackageAlpine:
			keeper = alpineKeeper
		default:
			return fmt.Errorf("unknown package extension: %s", ext)
		}

		packageList, err := keeper.Resolve(cmd.Context(), pkg.Name)
		if err != nil {
			return err
		}

		for _, p := range packageList {
			// download the package
			log.Info("installing package", "name", p)
			pkgPath, err := downloadFile(cmd.Context(), os.ExpandEnv(p))
			if err != nil {
				return err
			}

			// unpack the package into the root
			// filesystem
			if err := keeper.Unpack(cmd.Context(), pkgPath, rootfs); err != nil {
				return err
			}
		}
	}

	// download files
	for _, file := range cfg.Spec.Files {
		path := filepath.Clean(file.Path)
		dst := filepath.Join(rootfs, path)
		//if strings.HasSuffix(file.Path, "/") {
		//	dst = filepath.Join(rootfs, path, filepath.Base(file.URI))
		//}
		log.Info("downloading file", "file", file.URI, "path", dst)
		client := &getter.Client{
			Ctx:             cmd.Context(),
			Src:             file.URI,
			Dst:             dst,
			Mode:            getter.ClientModeFile,
			DisableSymlinks: true,
		}
		if err := client.Get(); err != nil {
			return err
		}
	}

	// package everything up as our final container image

	baseImage := os.ExpandEnv(cfg.Spec.From)
	if baseImage == "" {
		log.Info("using scratch base as nothing was provided")
		baseImage = "scratch"
	}

	platform, _ := v1.ParsePlatform("linux/amd64")
	img, err := containerutil.Append(cmd.Context(), rootfs, baseImage, platform)
	if err != nil {
		return err
	}

	if localPath != "" {
		return containerutil.Save(cmd.Context(), img, cfg.Name, localPath)
	}

	return nil
}

func repoURLs(p []aybv1.Repository) []string {
	s := make([]string, len(p))
	for i := range p {
		s[i] = p[i].URL
	}
	return s
}

func downloadFile(ctx context.Context, url string) (string, error) {
	log := logr.FromContextOrDiscard(ctx)
	log.V(1).Info("downloading file", "url", url)
	f, err := os.CreateTemp("", "*.apk")
	if err != nil {
		return "", err
	}
	defer f.Close()

	client := &getter.Client{
		Ctx:             ctx,
		Src:             url,
		Dst:             f.Name(),
		Mode:            getter.ClientModeFile,
		DisableSymlinks: true,
	}
	if err := client.Get(); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func readConfig(s string) (aybv1.Build, error) {
	f, err := os.Open(s)
	if err != nil {
		return aybv1.Build{}, err
	}

	var config aybv1.Build
	if err := yaml.NewYAMLOrJSONDecoder(f, 4).Decode(&config); err != nil {
		return aybv1.Build{}, err
	}
	return config, nil
}
