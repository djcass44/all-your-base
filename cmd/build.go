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
	"github.com/spf13/cobra"
	"io"
	"k8s.io/apimachinery/pkg/util/yaml"
	"net/http"
	"os"
	"path/filepath"
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

	// install packages
	for _, pkg := range cfg.Spec.Packages {
		var keeper packages.Package
		ext := filepath.Ext(pkg.URL)
		switch ext {
		case ".apk":
			keeper = &alpine.PackageKeeper{}
		default:
			return fmt.Errorf("unknown package extension: %s", ext)
		}
		// download the package
		pkgPath, err := downloadFile(cmd.Context(), pkg.URL)
		if err != nil {
			return err
		}

		// unpack the package into the root
		// filesystem
		if err := keeper.Unpack(cmd.Context(), pkgPath, rootfs); err != nil {
			return err
		}
	}

	baseImage := cfg.Spec.From
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
		return containerutil.Save(cmd.Context(), img, "latest", localPath)
	}

	return nil
}

func downloadFile(ctx context.Context, url string) (string, error) {
	log := logr.FromContextOrDiscard(ctx)
	log.V(1).Info("downloading file", "url", url)
	f, err := os.CreateTemp("", "*.apk")
	if err != nil {
		return "", err
	}
	defer f.Close()

	// get the data
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	log.V(2).Info("http request completed", "code", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected response code: %d", resp.StatusCode)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
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
