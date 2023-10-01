package ca_certificates

import (
	"bufio"
	"context"
	memfs "github.com/chainguard-dev/go-apk/pkg/fs"
	"github.com/go-logr/logr"
	"io/fs"
	"k8s.io/utils/strings/slices"
	"os"
	"path/filepath"
	"strings"
)

const (
	CertsConf = "/etc/ca-certificates.conf"
	CertsDir = "/usr/share/ca-certificates"
	LocalCertsDir = "/usr/local/share/ca-certificates"
	CertBundle = "ca-certificates.crt"
	EtcCertsDir = "/etc/ssl/certs"
)

// UpdateCertificates attempts to replicate what the "update-ca-certificates"
// script does.
func UpdateCertificates(ctx context.Context, rootfs memfs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx)

	// read all the environment variables
	certsConf := getEnv("CERTSCONF", CertsConf)
	certsDir := getEnv("CERTSDIR", CertsDir)
	localCertsDir := getEnv("LOCALCERTSDIR", LocalCertsDir)
	certBundle := getEnv("CERTBUNDLE", CertBundle)
	etcCertsDir := getEnv("ETCCERTSDIR", EtcCertsDir)

	blockedCerts, err := readBlockedCerts(ctx, certsConf, rootfs)
	if err != nil {
		return err
	}

	bundle := strings.Builder{}
	if err := walkCertDir(ctx, certsDir, &bundle, blockedCerts, rootfs); err != nil {
		return err
	}
	if err := walkCertDir(ctx, localCertsDir, &bundle, blockedCerts, rootfs); err != nil {
		return err
	}
	if err := rootfs.WriteFile(filepath.Join(etcCertsDir, certBundle), []byte(bundle.String()), 0644); err != nil {
		log.Error(err, "failed to save certificate bundle")
		return err
	}

	return nil
}

// getEnv is a simple call to os.Getenv
// that returns a default value if nothing
// has been set.
func getEnv(s, def string) string {
	val := os.Getenv(s)
	if val == "" {
		return def
	}
	return val
}

// readBlockedCerts reads the ca-certificates.conf file and returns the list of
// certificates that have been blocked (i.e. all starting with '!')
func readBlockedCerts(ctx context.Context, path string, rootfs memfs.FullFS) ([]string, error) {
	log := logr.FromContextOrDiscard(ctx)
	log.V(2).Info("reading ca-certificates configuration file")

	f, err := rootfs.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.V(2).Info("certificates configuration file does not exist")
			return nil, nil
		}
		return nil, err
	}

	var blocked []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "!") {
			blocked = append(blocked, strings.TrimPrefix(line, "!"))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	log.V(2).Info("successfully loaded blocked certificates", "count", len(blocked))
	return blocked, nil
}

// walkCertDir walks a given directory and searches for '.crt' files.
// If files are found and they're aren't blocked, they're added to the bundle
func walkCertDir(ctx context.Context, dir string, bundle *strings.Builder, blocked []string, rootfs memfs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx)
	log.V(2).Info("walking directory", "dir", dir)
	return fs.WalkDir(rootfs, dir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".crt") {
			return nil
		}
		log.V(3).Info("found certificate", "name", d.Name())

		// check if the certificate should be removed
		if slices.Contains(blocked, d.Name()) {
			log.V(1).Info("skipping blocked certificate", "name", d.Name())
			return nil
		}
		
		// read the file
		data, err := rootfs.ReadFile(path)
		if err != nil {
			return err
		}

		// append the file to our builder

		// include the filename in a comment
		bundle.WriteString("# ")
		bundle.WriteString(d.Name())
		bundle.WriteString("\n")
		bundle.Write(data)
		bundle.WriteString("\n")

		// todo create symlink
		return nil
	})
}