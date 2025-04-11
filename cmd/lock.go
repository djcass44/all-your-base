package cmd

import (
	"chainguard.dev/apko/pkg/apk/fs"
	"encoding/json"
	"fmt"
	"github.com/Snakdy/container-build-engine/pkg/containers"
	"github.com/Snakdy/container-build-engine/pkg/fetch"
	"github.com/Snakdy/container-build-engine/pkg/oci/auth"
	"github.com/djcass44/all-your-base/pkg/packages/rpm"
	"github.com/gosimple/hashdir"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/djcass44/all-your-base/pkg/airutil"
	aybv1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/lockfile"
	"github.com/djcass44/all-your-base/pkg/packages"
	"github.com/djcass44/all-your-base/pkg/packages/alpine"
	"github.com/djcass44/all-your-base/pkg/packages/debian"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/crane"
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
	log.Info("generating parent image checksum")
	if cfg.Spec.From != containers.MagicImageScratch {
		baseDigest, err := crane.Digest(airutil.ExpandEnv(cfg.Spec.From), crane.WithAuthFromKeychain(auth.KeyChain(auth.Auth{})))
		if err != nil {
			return err
		}

		resolved := cfg.Spec.From
		if !skipImageLocking {
			resolved = cfg.Spec.From + "@" + baseDigest
		} else {
			log.Info("warning: this build may not be reproducible - image locking is disabled")
		}

		lockFile.Packages[""] = lockfile.Package{
			Name:      cfg.Spec.From,
			Resolved:  resolved,
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

	alpineKeeper, err := alpine.NewPackageKeeper(cmd.Context(), repoURLs(cfg.Spec.Repositories[strings.ToLower(string(aybv1.PackageAlpine))]), fs.NewMemFS())
	if err != nil {
		return err
	}
	debianKeeper, err := debian.NewPackageKeeper(cmd.Context(), repoURLs(cfg.Spec.Repositories[strings.ToLower(string(aybv1.PackageDebian))]))
	if err != nil {
		return err
	}
	yumKeeper, err := rpm.NewPackageKeeper(cmd.Context(), repoURLs(cfg.Spec.Repositories[strings.ToLower(string(aybv1.PackageRPM))]))
	if err != nil {
		return err
	}

	// get package integrity
	log.Info("generating package checksums")
	for _, pkg := range cfg.Spec.Packages {
		var keeper packages.PackageManager
		switch pkg.Type {
		case aybv1.PackageAlpine:
			keeper = alpineKeeper
		case aybv1.PackageDebian:
			keeper = debianKeeper
		case aybv1.PackageRPM:
			keeper = yumKeeper
		default:
			return fmt.Errorf("unknown package type: %s", pkg.Type)
		}

		for _, name := range pkg.Names {
			packageList, err := keeper.Resolve(cmd.Context(), name)
			if err != nil {
				return err
			}

			for _, p := range packageList {
				log.V(1).Info("downloading package", "name", p.Name)

				packageUrl := p.Resolved
				for _, r := range repoList {
					// we need to chop the repo if it has a space as everything
					// after that is not useful (e.g. debian repo data)
					repoName, _, _ := strings.Cut(r.URL, " ")
					originalRepoName, _, _ := strings.Cut(r.Original, " ")
					if strings.HasPrefix(p.Resolved, repoName) {
						packageUrl = strings.ReplaceAll(p.Resolved, repoName, originalRepoName)
					}
				}

				p.Resolved = packageUrl
				lockFile.Packages[p.Name] = p
			}
		}

	}

	// get file integrity
	log.Info("generating file checksums")
	for _, file := range cfg.Spec.Files {
		// if the file source has a '/' suffix, then we should
		// treat it as a directory
		if strings.HasSuffix(file.URI, "/") {
			src := airutil.ExpandEnv(file.URI)
			log.V(1).Info("hashing directory", "dir", src)
			digest, err := hashdir.Make(src, "sha256")
			if err != nil {
				log.Error(err, "failed to generate directory digest", "alg", "sha256", "path", src)
				return err
			}
			lockFile.Packages[file.URI] = lockfile.Package{
				Name:      file.URI,
				Resolved:  src,
				Integrity: "sha256:" + digest,
				Type:      aybv1.PackageDir,
			}
			continue
		}
		dst, err := os.MkdirTemp("", "file-download-*")
		if err != nil {
			log.Error(err, "failed to prepare download directory")
			return err
		}
		srcUri, err := url.Parse(airutil.ExpandEnv(file.URI))
		if err != nil {
			return err
		}
		// disable archive handling
		q := srcUri.Query()
		q.Set("archive", "false")
		srcUri.RawQuery = q.Encode()

		log.V(1).Info("downloading file", "file", srcUri, "path", dst)
		out, err := fetch.Fetch(cmd.Context(), srcUri.String(), dst, "")
		if err != nil {
			log.Error(err, "failed to download file", "src", srcUri.String())
			return err
		}
		integrity, err := lockfile.Sha256(out)
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

	log.Info("exporting lockfile")
	f, err := os.Create(lockfile.Name(configPath))
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")
	return enc.Encode(lockFile)
}
