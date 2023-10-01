package ca_certificates

import (
	"context"
	memfs "github.com/chainguard-dev/go-apk/pkg/fs"
	"github.com/go-logr/logr"
	"io/fs"
	"path/filepath"
	"strings"
)

const (
	CertsDir = "/usr/share/ca-certificates"
	LocalCertsDir = "/usr/local/share/ca-certificates"
	CertBundle = "ca-certificates.crt"
	EtcCertsDir = "/etc/ssl/certs"
)

// UpdateCertificates attempts to replicate what the "update-ca-certificates"
// script does.
func UpdateCertificates(ctx context.Context, rootfs memfs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx)
	bundle := strings.Builder{}
	err := fs.WalkDir(rootfs, CertsDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".crt") {
			return nil
		}
		log.V(3).Info("found certificate", "name", d.Name())
		// todo check if the certificate should be removed
		
		// read the file
		data, err := rootfs.ReadFile(path)
		if err != nil {
			return err
		}
		
		// append the file to our builder
		bundle.Write(data)
		bundle.WriteString("\n")
		
		// todo create symlink
		return nil
	})
	if err != nil {
		return err
	}
	if err := rootfs.WriteFile(filepath.Join(EtcCertsDir, CertBundle), []byte(bundle.String()), 0644); err != nil {
		log.Error(err, "failed to walk directory")
		return err
	}
	
	return nil
}