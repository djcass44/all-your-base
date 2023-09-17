package cmd

import (
	"fmt"
	aybv1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/containerutil"
	"github.com/djcass44/all-your-base/pkg/downloader"
	"github.com/djcass44/all-your-base/pkg/linuxutil"
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

	flagUsername = "username"
	flagUid      = "uid"

	flagCacheDir = "cache-dir"
)

func init() {
	buildCmd.Flags().StringP(flagConfig, "c", "", "path to an image configuration file")
	buildCmd.Flags().String(flagSave, "", "path to save the image as a tar archive")

	buildCmd.Flags().String(flagUsername, "somebody", "name of the non-root user to create")
	buildCmd.Flags().Int(flagUid, 1001, "uid of the non-root user to create")

	buildCmd.Flags().String(flagCacheDir, "", "cache directory (defaults to user cache dir)")

	_ = buildCmd.MarkFlagRequired(flagConfig)
	_ = buildCmd.MarkFlagFilename(flagConfig, ".yaml", ".yml")
	_ = buildCmd.MarkFlagDirname(flagCacheDir)
}

func build(cmd *cobra.Command, _ []string) error {
	log := logr.FromContextOrDiscard(cmd.Context())

	configPath, _ := cmd.Flags().GetString(flagConfig)
	localPath, _ := cmd.Flags().GetString(flagSave)

	username, _ := cmd.Flags().GetString(flagUsername)
	uid, _ := cmd.Flags().GetInt(flagUid)

	cacheDir, _ := cmd.Flags().GetString(flagCacheDir)
	if cacheDir == "" {
		cacheDir, _ = os.UserCacheDir()
		cacheDir = filepath.Join(cacheDir, "ayb")
	}

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

	dl, err := downloader.NewDownloader(cacheDir)
	if err != nil {
		return err
	}

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
			pkgPath, err := dl.Download(cmd.Context(), os.ExpandEnv(p))
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

	// create the non-root user
	if err := linuxutil.NewUser(cmd.Context(), rootfs, username, uid); err != nil {
		return err
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
		if err := os.Chmod(dst, 0664); err != nil {
			log.Error(err, "failed to update file permissions", "file", dst)
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
	img, err := containerutil.Append(cmd.Context(), rootfs, baseImage, platform, username)
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
