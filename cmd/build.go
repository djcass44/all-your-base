package cmd

import (
	"fmt"
	"github.com/chainguard-dev/go-apk/pkg/fs"
	aybv1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/containerutil"
	"github.com/djcass44/all-your-base/pkg/downloader"
	"github.com/djcass44/all-your-base/pkg/fileutil"
	"github.com/djcass44/all-your-base/pkg/linuxutil"
	"github.com/djcass44/all-your-base/pkg/lockfile"
	"github.com/djcass44/all-your-base/pkg/packages"
	"github.com/djcass44/all-your-base/pkg/packages/alpine"
	"github.com/go-logr/logr"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/hashicorp/go-getter"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/yaml"
	"net/url"
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

	flagUid = "uid"

	flagCacheDir = "cache-dir"
	flagPlatform = "platform"
)

const (
	defaultUsername = "somebody"
	defaultUid      = 1001
)

func init() {
	buildCmd.Flags().StringP(flagConfig, "c", "", "path to an image configuration file")
	buildCmd.Flags().String(flagSave, "", "path to save the image as a tar archive")

	buildCmd.Flags().Int(flagUid, 1001, "uid of the non-root user to create")

	buildCmd.Flags().String(flagCacheDir, "", "cache directory (defaults to user cache dir)")
	buildCmd.Flags().String(flagPlatform, "linux/amd64", "build platform")

	_ = buildCmd.MarkFlagRequired(flagConfig)
	_ = buildCmd.MarkFlagFilename(flagConfig, ".yaml", ".yml")
	_ = buildCmd.MarkFlagDirname(flagCacheDir)
}

func build(cmd *cobra.Command, _ []string) error {
	log := logr.FromContextOrDiscard(cmd.Context())

	configPath, _ := cmd.Flags().GetString(flagConfig)
	localPath, _ := cmd.Flags().GetString(flagSave)

	cacheDir, _ := cmd.Flags().GetString(flagCacheDir)
	cacheDir = getCacheDir(cacheDir)

	platform, _ := cmd.Flags().GetString(flagPlatform)

	// read the config file
	cfg, err := readConfig(configPath)
	if err != nil {
		return err
	}

	lockFile, err := lockfile.Read(cmd.Context(), configPath)
	if err != nil {
		return err
	}

	configPath, err = filepath.Abs(configPath)
	if err != nil {
		return err
	}

	// set our working directory to the directory containing the
	// configuration file
	wd := filepath.Dir(configPath)
	_ = os.Chdir(wd)
	log.Info("updating working directory", "dir", wd)

	rootfs := fs.NewMemFS()
	log.V(3).Info("prepared root filesystem")

	dl, err := downloader.NewDownloader(cacheDir)
	if err != nil {
		return err
	}

	alpineKeeper, err := alpine.NewPackageKeeper(cmd.Context(), repoURLs(cfg.Spec.Repositories[strings.ToLower(string(aybv1.PackageAlpine))]))
	if err != nil {
		return err
	}

	// install packages
	for _, pkg := range cfg.Spec.Packages {
		var keeper packages.PackageManager
		switch pkg.Type {
		case aybv1.PackageAlpine:
			keeper = alpineKeeper
		default:
			return fmt.Errorf("unknown package type: %s", pkg.Type)
		}

		for _, name := range pkg.Names {
			packageList, err := keeper.Resolve(cmd.Context(), name)
			if err != nil {
				return err
			}

			for _, p := range packageList {
				log.Info("installing package", "name", p.Name)

				// check that the package is in the lockfile
				locked, ok := lockFile.Packages[p.Name]
				if !ok {
					return fmt.Errorf("package could not be located in lockfile: %s", p.Name)
				}

				log.V(4).Info("comparing package integrity against lockfile", "expected", locked.Integrity, "actual", p.Integrity)
				if locked.Integrity != p.Integrity {
					return fmt.Errorf("package integrity check failed (expected: '%s', got: '%s')", locked.Integrity, p.Integrity)
				}

				// download the package
				pkgPath, err := dl.Download(cmd.Context(), p.Resolved)
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
	}

	// create the non-root user
	if err := linuxutil.NewUser(cmd.Context(), rootfs, defaultUsername, defaultUid); err != nil {
		return err
	}

	// download files
	for _, file := range cfg.Spec.Files {
		path := filepath.Clean(file.Path)
		dst, err := os.MkdirTemp("", "file-download-*")
		if err != nil {
			log.Error(err, "failed to prepare download directory")
			return err
		}
		srcUri, err := url.Parse(os.ExpandEnv(file.URI))
		if err != nil {
			return err
		}
		checksum, ok := lockFile.Packages[file.URI]
		if !ok {
			return fmt.Errorf("failed to locate lock statement for package: %s\nYou may need to update the lock file with the 'lock' command", file.URI)
		}
		q := srcUri.Query()
		q.Set("checksum", checksum.Integrity)
		srcUri.RawQuery = q.Encode()

		log.Info("downloading file", "file", srcUri.String(), "path", dst)
		client := &getter.Client{
			Ctx:             cmd.Context(),
			Pwd:             wd,
			Src:             srcUri.String(),
			Dst:             dst,
			DisableSymlinks: true,
			Mode:            getter.ClientModeAny,
			Getters:         getters,
		}
		if err := client.Get(); err != nil {
			log.Error(err, "failed to download file")
			return err
		}
		var permissions os.FileMode = 0644
		if file.Executable {
			permissions = 0755
		}
		copySrc := dst
		if file.SubPath != "" || filepath.Ext(file.URI) == "" {
			if file.SubPath != "" {
				copySrc = filepath.Join(dst, file.SubPath)
			}
			if filepath.Ext(file.URI) == "" {
				copySrc = filepath.Join(dst, filepath.Base(file.URI))
			}
			log.V(1).Info("updating file permissions", "file", copySrc, "permissions", permissions)
			if err := os.Chmod(copySrc, permissions); err != nil {
				log.Error(err, "failed to update file permissions", "file", copySrc)
				return err
			}
		}
		log.V(2).Info("copying file or directory", "src", copySrc, "dst", path)
		if err := fileutil.CopyDirectory(copySrc, path, rootfs); err != nil {
			log.Error(err, "failed to copy directory")
			return err
		}
	}

	// create links
	for _, link := range cfg.Spec.Links {
		srcPath := filepath.Clean(link.Source)
		dstPath := filepath.Clean(link.Target)

		log.Info("creating link", "src", srcPath, "dst", dstPath)
		if err := rootfs.Symlink(srcPath, dstPath); err != nil {
			log.Error(err, "failed to create link")
			return err
		}
	}

	// package everything up as our final container image

	baseImage := os.ExpandEnv(cfg.Spec.From)
	switch baseImage {
	case containerutil.MagicImageScratch:
	case "":
		log.Info("using scratch base as nothing was provided")
		baseImage = containerutil.MagicImageScratch
	default:
		baseImage = baseImage + "@" + lockFile.Packages[""].Integrity
	}

	imgPlatform, err := v1.ParsePlatform(platform)
	if err != nil {
		log.Error(err, "failed to parse platform")
		return err
	}
	img, err := containerutil.Append(cmd.Context(), rootfs, baseImage, imgPlatform, defaultUsername)
	if err != nil {
		return err
	}

	if localPath != "" {
		return containerutil.Save(cmd.Context(), img, cfg.Name, localPath)
	}

	return nil
}

var getters = map[string]getter.Getter{
	"file":  &getter.FileGetter{Copy: true},
	"https": &getter.HttpGetter{XTerraformGetDisabled: true, Netrc: true},
	"s3":    &getter.S3Getter{},
	"git":   &getter.GitGetter{},
	"gcs":   &getter.GCSGetter{},
	"hg":    &getter.HgGetter{},
}

func getCacheDir(d string) string {
	if d == "" {
		d, _ = os.UserCacheDir()
		d = filepath.Join(d, "ayb")
	}
	return d
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
