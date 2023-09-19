package fileutil

import (
	fullfs "github.com/chainguard-dev/go-apk/pkg/fs"
	"os"
)

func IsSymbolicLink(rootfs fullfs.FullFS, path string) (bool, error) {
	info, err := rootfs.Lstat(path)
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return true, nil
	}
	return false, nil
}
