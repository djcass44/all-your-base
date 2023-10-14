package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/djcass44/all-your-base/pkg/airutil"
	aybv1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/containerutil"
	"github.com/djcass44/all-your-base/pkg/lockfile"
	"github.com/djcass44/all-your-base/pkg/packages"
	"github.com/djcass44/all-your-base/pkg/packages/alpine"
	"github.com/djcass44/all-your-base/pkg/packages/debian"
	"github.com/djcass44/ci-tools/pkg/ociutil"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/hashicorp/go-getter"
	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "generate a lockfile",
	RunE:  lock,
}

const flagSkipImageLocking = "skip-image-locking"

func init() {
	lockCmd.Flags().StringP(flagConfig, "c", "", "path to an image configuration file")

	lockCmd.Flags().Bool(flagSkipImageLocking, false, "skip locking of the base image")

	_ = lockCmd.MarkFlagRequired(flagConfig)
	_ = lockCmd.MarkFlagFilename(flagConfig, ".yaml", ".yml")
	_ = lockCmd.MarkFlagDirname(flagCacheDir)
}

func lock(cmd *cobra.Command, _ []string) error {
	log := logr.FromContextOrDiscard(cmd.Context())

	configPath, _ := cmd.Flags().GetString(flagConfig)
	skipImageLocking, _ := cmd.Flags().GetBool(flagSkipImageLocking)

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
	if !skipImageLocking && cfg.Spec.From != containerutil.MagicImageScratch {
		baseDigest, err := crane.Digest(airutil.ExpandEnv(cfg.Spec.From), crane.WithAuthFromKeychain(ociutil.KeyChain(ociutil.Auth{})))
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

	type expandedRepo struct {
		URL      string
		Original string
	}

	var repoList []expandedRepo
	for _, v := range cfg.Spec.Repositories {
		for _, vv := range v {
			repoList = append(repoList, expandedRepo{
				URL:      airutil.ExpandEnv(vv.URL),
				Original: vv.URL,
			})
		}
	}

	alpineKeeper, err := alpine.NewPackageKeeper(cmd.Context(), repoURLs(cfg.Spec.Repositories[strings.ToLower(string(aybv1.PackageAlpine))]))
	if err != nil {
		return err
	}
	debianKeeper, err := debian.NewPackageKeeper(cmd.Context(), repoURLs(cfg.Spec.Repositories[strings.ToLower(string(aybv1.PackageDebian))]))
	if err != nil {
		return err
	}

	// get package integrity
	for _, pkg := range cfg.Spec.Packages {
		var keeper packages.PackageManager
		switch pkg.Type {
		case aybv1.PackageAlpine:
			keeper = alpineKeeper
		case aybv1.PackageDebian:
			keeper = debianKeeper
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

				packageUrl := p.Resolved
				for _, r := range repoList {
					if strings.HasPrefix(p.Resolved, r.URL) {
						packageUrl = strings.ReplaceAll(p.Resolved, r.URL, r.Original)
					}
				}

				p.Resolved = packageUrl
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
		srcUri, err := url.Parse(airutil.ExpandEnv(file.URI))
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
			log.Error(err, "failed to download file", "src", srcUri.String())
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
