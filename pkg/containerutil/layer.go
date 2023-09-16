package containerutil

import (
	"archive/tar"
	"bytes"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"io"
	"os"
	"path"
	"path/filepath"
)

func NewLayer(appPath string, platform *v1.Platform) (v1.Layer, error) {
	layerBuf, err := tarDir(appPath, platform)
	if err != nil {
		return nil, fmt.Errorf("tarring data: %w", err)
	}
	layerBytes := layerBuf.Bytes()
	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewBuffer(layerBytes)), nil
	}, tarball.WithCompressedCaching, tarball.WithMediaType(types.OCILayer))
}

func tarDir(appPath string, platform *v1.Platform) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	if err := walkRecursive(tw, appPath, "/", v1.Time{}, platform); err != nil {
		return nil, err
	}
	return buf, nil
}

// walkRecursive performs a filepath.Walk of the given root directory adding it
// to the provided tar.Writer with root -> chroot.  All symlinks are dereferenced,
// which is what leads to recursion when we encounter a directory symlink.
func walkRecursive(tw *tar.Writer, root, chroot string, creationTime v1.Time, platform *v1.Platform) error {
	return filepath.Walk(root, func(hostPath string, info os.FileInfo, err error) error {
		if hostPath == root {
			return nil
		}
		if err != nil {
			return fmt.Errorf("filepath.Walk(%q): %w", root, err)
		}
		// Skip other directories.
		if info.Mode().IsDir() {
			return nil
		}
		newPath := path.Join(chroot, filepath.ToSlash(hostPath[len(root):]))

		evalPath, err := filepath.EvalSymlinks(hostPath)
		if err != nil {
			return fmt.Errorf("filepath.EvalSymlinks(%q): %w", hostPath, err)
		}

		// Chase symlinks.
		info, err = os.Stat(evalPath)
		if err != nil {
			return fmt.Errorf("os.Stat(%q): %w", evalPath, err)
		}
		// Skip other directories.
		if info.Mode().IsDir() {
			return walkRecursive(tw, evalPath, newPath, creationTime, platform)
		}

		// Open the file to copy it into the tarball.
		file, err := os.Open(evalPath)
		if err != nil {
			return fmt.Errorf("os.Open(%q): %w", evalPath, err)
		}
		defer file.Close()

		// Copy the file into the image tarball.
		header := &tar.Header{
			Name:     newPath,
			Size:     info.Size(),
			Typeflag: tar.TypeReg,
			Mode:     int64(info.Mode()),
			ModTime:  creationTime.Time,
		}
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("tar.Writer.WriteHeader(%q): %w", newPath, err)
		}
		if _, err := io.Copy(tw, file); err != nil {
			return fmt.Errorf("io.Copy(%q, %q): %w", newPath, evalPath, err)
		}
		return nil
	})
}
