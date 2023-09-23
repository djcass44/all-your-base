package cmd

import (
	"encoding/json"
	"fmt"
	aybv1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/containerutil"
	"github.com/djcass44/all-your-base/pkg/lockfile"
	"github.com/djcass44/all-your-base/pkg/packages"
	"github.com/djcass44/all-your-base/pkg/packages/alpine"
	"github.com/djcass44/ci-tools/pkg/ociutil"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/hashicorp/go-getter"
	"github.com/spf13/cobra"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "generate a lockfile",
	RunE:  lock,
}

func init() {
	lockCmd.Flags().StringP(flagConfig, "c", "", "path to an image configuration file")

	lockCmd.Flags().String(flagCacheDir, "", "cache directory (defaults to user cache dir)")

	_ = lockCmd.MarkFlagRequired(flagConfig)
	_ = lockCmd.MarkFlagFilename(flagConfig, ".yaml", ".yml")
	_ = lockCmd.MarkFlagDirname(flagCacheDir)
}

func lock(cmd *cobra.Command, _ []string) error {
	log := logr.FromContextOrDiscard(cmd.Context())

	configPath, _ := cmd.Flags().GetString(flagConfig)

	cacheDir, _ := cmd.Flags().GetString(flagCacheDir)
	cacheDir = getCacheDir(cacheDir)

	// read the config file
	cfg, err := readConfig(configPath)
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

	lockFile := lockfile.Lock{
		Name:            cfg.Name,
		LockfileVersion: 1,
		Packages:        map[string]lockfile.Package{},
	}

	// get the digest of the base image
	if cfg.Spec.From != containerutil.MagicImageScratch {
		baseDigest, err := crane.Digest(os.ExpandEnv(cfg.Spec.From), crane.WithAuthFromKeychain(ociutil.KeyChain(ociutil.Auth{})))
		if err != nil {
			return err
		}

		lockFile.Packages[""] = lockfile.Package{
			Name:      cfg.Spec.From,
			Resolved:  cfg.Spec.From + "@" + baseDigest,
			Integrity: baseDigest,
			Type:      aybv1.PackageOCI,
		}
	}

	alpineKeeper, err := alpine.NewPackageKeeper(cmd.Context(), repoURLs(cfg.Spec.Repositories[strings.ToLower(string(aybv1.PackageAlpine))]))
	if err != nil {
		return err
	}

	// get package integrity
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
				log.Info("downloading package", "name", p.Name)

				lockFile.Packages[p.Name] = p
			}
		}

	}

	// get file integrity
	for _, file := range cfg.Spec.Files {
		dst, err := os.CreateTemp("", "file-download-*")
		if err != nil {
			log.Error(err, "failed to prepare download directory")
			return err
		}
		_ = dst.Close()
		srcUri, err := url.Parse(os.ExpandEnv(file.URI))
		if err != nil {
			return err
		}
		// disable archive handling
		q := srcUri.Query()
		q.Set("archive", "false")
		srcUri.RawQuery = q.Encode()

		log.Info("downloading file", "file", srcUri, "path", dst.Name())
		client := &getter.Client{
			Ctx:             cmd.Context(),
			Pwd:             wd,
			Src:             srcUri.String(),
			Dst:             dst.Name(),
			DisableSymlinks: true,
			Mode:            getter.ClientModeFile,
			Getters:         getters,
		}
		if err := client.Get(); err != nil {
			log.Error(err, "failed to download file")
			return err
		}
		integrity, err := lockfile.Sha256(dst.Name())
		if err != nil {
			return err
		}
		lockFile.Packages[file.URI] = lockfile.Package{
			Name:      file.URI,
			Resolved:  srcUri.String(),
			Integrity: "sha256:" + integrity,
			Type:      aybv1.PackageFile,
		}
	}

	f, err := os.Create(lockfile.Name(configPath))
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")
	return enc.Encode(lockFile)
}
