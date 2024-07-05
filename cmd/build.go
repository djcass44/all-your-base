package cmd

import (
	"fmt"
	"github.com/Snakdy/container-build-engine/pkg/builder"
	"github.com/Snakdy/container-build-engine/pkg/containers"
	"github.com/Snakdy/container-build-engine/pkg/pipelines"
	"github.com/djcass44/all-your-base/internal/statements"
	"github.com/djcass44/all-your-base/pkg/packages/rpm"
	"github.com/hashicorp/go-getter"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainguard-dev/go-apk/pkg/fs"
	"github.com/djcass44/all-your-base/pkg/airutil"
	aybv1 "github.com/djcass44/all-your-base/pkg/api/v1"
	cacertificates "github.com/djcass44/all-your-base/pkg/ca-certificates"
	"github.com/djcass44/all-your-base/pkg/downloader"
	"github.com/djcass44/all-your-base/pkg/lockfile"
	"github.com/djcass44/all-your-base/pkg/packages/alpine"
	"github.com/djcass44/all-your-base/pkg/packages/debian"
	"github.com/go-logr/logr"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build an image",
	RunE:  build,
}

const (
	flagConfig = "config"

	flagSave  = "save"
	flagImage = "image"
	flagTag   = "tag"

	flagUid      = "uid"
	flagUsername = "username"

	flagCacheDir = "cache-dir"
	flagPlatform = "platform"

	flagSkipCACerts = "skip-ca-certificates"
)

const (
	defaultUsername = "somebody"
	defaultUid      = 1001
)

func init() {
	buildCmd.Flags().StringP(flagConfig, "c", "", "path to an image configuration file")

	buildCmd.Flags().String(flagSave, "", "path to save the image as a tar archive")
	buildCmd.Flags().String(flagImage, "", "oci image path (without tag) to push the image")
	buildCmd.Flags().StringArrayP(flagTag, "t", nil, "tags to push")

	buildCmd.Flags().Int(flagUid, defaultUid, "uid of the non-root user to create")
	buildCmd.Flags().String(flagUsername, defaultUsername, "username of the non-root user to create")

	buildCmd.Flags().String(flagCacheDir, "", "cache directory (defaults to user cache dir)")
	buildCmd.Flags().String(flagPlatform, "linux/amd64", "build platform")

	buildCmd.Flags().Bool(flagSkipCACerts, false, "skip running update-ca-certificates")

	_ = buildCmd.MarkFlagRequired(flagConfig)
	_ = buildCmd.MarkFlagFilename(flagConfig, ".yaml", ".yml")
	_ = buildCmd.MarkFlagDirname(flagCacheDir)

	buildCmd.MarkFlagsMutuallyExclusive(flagSave, flagImage)
	buildCmd.MarkFlagsRequiredTogether(flagImage, flagTag)
}

func build(cmd *cobra.Command, _ []string) error {
	log := logr.FromContextOrDiscard(cmd.Context())

	configPath, _ := cmd.Flags().GetString(flagConfig)
	localPath, _ := cmd.Flags().GetString(flagSave)
	ociPath, _ := cmd.Flags().GetString(flagImage)
	tags, _ := cmd.Flags().GetStringArray(flagTag)

	cacheDir, _ := cmd.Flags().GetString(flagCacheDir)
	cacheDir = getCacheDir(cacheDir)

	platform, _ := cmd.Flags().GetString(flagPlatform)
	skipCaCerts, _ := cmd.Flags().GetBool(flagSkipCACerts)

	username, _ := cmd.Flags().GetString(flagUsername)

	imgPlatform, err := v1.ParsePlatform(platform)
	if err != nil {
		log.Error(err, "failed to parse platform")
		return err
	}

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

	alpineKeeper, err := alpine.NewPackageKeeper(cmd.Context(), repoURLs(cfg.Spec.Repositories[strings.ToLower(string(aybv1.PackageAlpine))]), rootfs)
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

	// validate that the configuration file lines up
	// with what we expect from the lockfile
	if err := lockFile.Validate(cfg.Spec); err != nil {
		return err
	}

	pkgKeys := lockFile.SortedKeys()

	var pipelineStatements []pipelines.OrderedPipelineStatement

	// install packages
	for i, name := range pkgKeys {
		p := lockFile.Packages[name]

		pipelineStatements = append(pipelineStatements, pipelines.OrderedPipelineStatement{
			ID: fmt.Sprintf("pkg-%d", i),
			Options: map[string]any{
				"type":     string(p.Type),
				"name":     p.Name,
				"version":  p.Version,
				"resolved": p.Resolved,
			},
			Statement: statements.NewPackageStatement(alpineKeeper, debianKeeper, yumKeeper, dl),
			DependsOn: []string{statements.StatementEnv},
		})
	}

	baseImage := airutil.ExpandEnv(lockFile.Packages[""].Resolved)
	switch baseImage {
	case containers.MagicImageScratch:
	case "":
		log.Info("using scratch base as nothing was provided")
		baseImage = containers.MagicImageScratch
	default:
		baseImage = airutil.ExpandEnv(cfg.Spec.From)
	}

	// pull the base image
	baseImg, err := containers.Get(cmd.Context(), baseImage)
	if err != nil {
		return err
	}
	imgCfg, err := baseImg.ConfigFile()
	if err != nil {
		return err
	}

	// sort out environment variables
	envOpts := map[string]any{"HOME": "/home/somebody"}
	for _, kv := range imgCfg.Config.Env {
		k, v, _ := strings.Cut(kv, "=")
		envOpts[k] = v
	}
	for _, vars := range cfg.Spec.Env {
		envOpts[vars.Name] = os.Expand(vars.Value, expandMap(envOpts))
	}

	pipelineStatements = append(pipelineStatements, pipelines.OrderedPipelineStatement{
		ID:        statements.StatementEnv,
		Options:   envOpts,
		Statement: &pipelines.Env{},
	})

	// collect a list of all the file statements in case
	// something should be run after files are in place
	var fileDeps []string

	// download files
	for i, file := range cfg.Spec.Files {
		// expand paths using environment variables
		path := filepath.Clean(os.Expand(file.Path, expandMap(envOpts)))
		id := fmt.Sprintf("file-download-%d", i)
		pipelineStatements = append(pipelineStatements, pipelines.OrderedPipelineStatement{
			ID: id,
			Options: map[string]any{
				"uri":        airutil.ExpandEnv(file.URI),
				"path":       path,
				"executable": file.Executable,
				"sub-path":   file.SubPath,
			},
			Statement: &pipelines.File{},
			DependsOn: []string{statements.StatementEnv},
		})
		fileDeps = append(fileDeps, id)
	}

	// create links
	linkOpts := map[string]any{}
	for _, link := range cfg.Spec.Links {
		srcPath := filepath.Clean(link.Source)
		dstPath := filepath.Clean(link.Target)
		linkOpts[srcPath] = dstPath
	}
	pipelineStatements = append(pipelineStatements, pipelines.OrderedPipelineStatement{
		ID:        "symbolic-links",
		Options:   linkOpts,
		Statement: &pipelines.SymbolicLink{},
		DependsOn: fileDeps,
	})

	// update ca certificates
	if !skipCaCerts {
		if err := cacertificates.UpdateCertificates(cmd.Context(), rootfs); err != nil {
			return err
		}
	}

	entrypoint := cfg.Spec.Entrypoint
	if entrypoint == nil {
		entrypoint = []string{"/bin/sh"}
	}

	// package everything up as our final container image
	imageBuilder, err := builder.NewBuilder(cmd.Context(), baseImage, pipelineStatements, builder.Options{
		Username:        username,
		WorkingDir:      wd,
		Entrypoint:      entrypoint,
		Command:         cfg.Spec.Command,
		ForceEntrypoint: true,
		DirFS:           false,
		Metadata: builder.MetadataOptions{
			CreatedBy: "all-your-base",
		},
	})
	if err != nil {
		return err
	}
	img, err := imageBuilder.Build(cmd.Context(), imgPlatform)
	if err != nil {
		return err
	}

	if localPath != "" {
		return containers.Save(cmd.Context(), img, cfg.Name, localPath)
	}
	// push all tags
	for _, t := range tags {
		if err := containers.Push(cmd.Context(), img, fmt.Sprintf("%s:%s", ociPath, t)); err != nil {
			return err
		}
	}

	return nil
}

func expandMap(kv map[string]any) func(s string) string {
	return func(s string) string {
		for k, v := range kv {
			if k == s {
				return v.(string)
			}
		}
		return ""
	}
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
		s[i] = airutil.ExpandEnv(p[i].URL)
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
