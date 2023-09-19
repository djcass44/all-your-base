package containerutil

import (
	"archive/tar"
	"bytes"
	"fmt"
	fullfs "github.com/chainguard-dev/go-apk/pkg/fs"
	"github.com/djcass44/all-your-base/pkg/fileutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"io"
	"io/fs"
	"os"
	"strings"
)

func NewLayer(fs fullfs.FullFS, platform *v1.Platform) (v1.Layer, error) {
	layerBuf, err := tarDir(fs, platform)
	if err != nil {
		return nil, fmt.Errorf("tarring data: %w", err)
	}
	layerBytes := layerBuf.Bytes()
	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewBuffer(layerBytes)), nil
	}, tarball.WithCompressedCaching, tarball.WithMediaType(types.OCILayer))
}

func tarDir(fs fullfs.FullFS, platform *v1.Platform) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	if err := walkRecursive(fs, tw, "/", v1.Time{}, platform); err != nil {
		return nil, err
	}
	return buf, nil
}

// walkRecursive performs a filepath.Walk of the given root directory adding it
// to the provided tar.Writer with root -> chroot.  All symlinks are dereferenced,
// which is what leads to recursion when we encounter a directory symlink.
func walkRecursive(rootfs fullfs.FullFS, tw *tar.Writer, root string, creationTime v1.Time, platform *v1.Platform) error {
	return fs.WalkDir(rootfs, root, func(hostPath string, d os.DirEntry, err error) error {
		if hostPath == root {
			return nil
		}
		if err != nil {
			return fmt.Errorf("fs.WalkDir(%q): %w", root, err)
		}
		// create directory shells
		if d.IsDir() {
			header := &tar.Header{
				Name:     hostPath,
				Typeflag: tar.TypeDir,
				Mode:     0755,
				ModTime:  creationTime.Time,
			}
			if err := tw.WriteHeader(header); err != nil {
				return fmt.Errorf("tar.Writer.WriteHeader(%q): %w", hostPath, err)
			}
			return nil
		}

		evalPath := hostPath
		ok, err := fileutil.IsSymbolicLink(rootfs, hostPath)
		if err != nil {
			return fmt.Errorf("fileutil.IsSymbolicLink(%q): %w", hostPath, err)
		}
		if ok {
			evalPath, err = rootfs.Readlink(hostPath)
			if err != nil {
				return fmt.Errorf("fs.Readlink(%q): %w", hostPath, err)
			}
		}

		// Chase symlinks.
		info, err := fs.Stat(rootfs, evalPath)
		if err != nil {
			return fmt.Errorf("fs.Stat(%q): %w", evalPath, err)
		}
		// Skip other directories.
		if info.Mode().IsDir() {
			return walkRecursive(rootfs, tw, hostPath, creationTime, platform)
		}

		// Open the file to copy it into the tarball.
		file, err := rootfs.Open(evalPath)
		if err != nil {
			return fmt.Errorf("os.Open(%q): %w", evalPath, err)
		}
		defer file.Close()

		// hacky method of setting the uid...
		uid := 0
		if strings.HasPrefix(hostPath, "/home/somebody") {
			uid = 1001
		}

		// Copy the file into the image tarball.
		header := &tar.Header{
			Name:     hostPath,
			Size:     info.Size(),
			Typeflag: tar.TypeReg,
			Uid:      uid,
			Gid:      0,
			Mode:     int64(info.Mode()),
			ModTime:  creationTime.Time,
		}
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("tar.Writer.WriteHeader(%q): %w", hostPath, err)
		}
		if _, err := io.Copy(tw, file); err != nil {
			return fmt.Errorf("io.Copy(%q, %q): %w", hostPath, evalPath, err)
		}
		return nil
	})
}
