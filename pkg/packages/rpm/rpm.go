package rpm

import (
	"chainguard.dev/apko/pkg/apk/fs"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/cavaliergopher/rpm"
	v1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/lockfile"
	"github.com/djcass44/all-your-base/pkg/yum"
	"github.com/djcass44/all-your-base/pkg/yum/yumindex"
	"github.com/go-logr/logr"
	"github.com/sassoftware/go-rpmutils/cpio"
	"github.com/ulikunitz/xz"
	"golang.org/x/exp/maps"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

type PackageKeeper struct {
	indices []*yumindex.Metadata
}

func NewPackageKeeper(ctx context.Context, repositories []string) (*PackageKeeper, error) {
	log := logr.FromContextOrDiscard(ctx)

	var indices []*yumindex.Metadata
	for _, repo := range repositories {
		idx, err := yum.NewIndex(ctx, repo)
		if err != nil {
			return nil, err
		}
		log.V(2).Info("added index", "count", idx.Packages, "source", repo)
		indices = append(indices, idx)
	}
	return &PackageKeeper{
		indices: indices,
	}, nil
}

func (p *PackageKeeper) Unpack(ctx context.Context, pkgFile string, rootfs fs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("pkg", pkgFile)
	log.V(4).Info("unpacking rpm")

	f, err := os.Open(pkgFile)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	pkg, err := rpm.Read(f)
	if err != nil {
		return fmt.Errorf("reading package header: %w", err)
	}

	compression := pkg.PayloadCompression()
	log.V(6).Info("detected payload compression", "compression", compression, "supported", supportedRPMCompressionTypes)
	if !slices.Contains(supportedRPMCompressionTypes, compression) {
		return fmt.Errorf("unsupported compression: %s", compression)
	}

	var reader io.Reader

	switch compression {
	case compressionXZ:
		xzReader, err := xz.NewReader(f)
		if err != nil {
			return fmt.Errorf("creating xz reader: %w", err)
		}
		reader = xzReader
	case compressionGzip:
		gzipReader, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("creating gzip reader: %w", err)
		}
		reader = gzipReader
	}

	if format := pkg.PayloadFormat(); format != "cpio" {
		return fmt.Errorf("unsupported payload format: %s", format)
	}

	return p.Extract(ctx, rootfs, reader)
}

// Extract the contents of a cpio stream from r to the destination directory dest
func (p *PackageKeeper) Extract(ctx context.Context, rootfs fs.FullFS, rs io.Reader) error {
	log := logr.FromContextOrDiscard(ctx)

	linkMap := make(map[int][]string)

	stream := cpio.NewReader(rs)

	for {
		entry, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading stream: %w", err)
		}

		// sanitize path
		target := path.Clean(entry.Filename())
		for strings.HasPrefix(target, "../") {
			target = target[3:]
		}
		target = filepath.Join("/", filepath.FromSlash(target))
		if !strings.HasPrefix(target, string(filepath.Separator)) && "/" != target {
			// this shouldn't happen due to the sanitization above but always check
			return fmt.Errorf("invalid cpio path %q", entry.Filename())
		}
		// create the parent directory if it doesn't exist.
		if dir := filepath.Dir(entry.Filename()); dir != "" {
			if _, err := rootfs.Stat(dir); err != nil {
				if os.IsNotExist(err) {
					log.V(2).Info("creating parent directory", "path", dir)
					if err := rootfs.MkdirAll(dir, 0o755); err != nil {
						return fmt.Errorf("creating directory: %w", err)
					}
				} else {
					return fmt.Errorf("checking directory: %w", err)
				}
			}
		}
		// FIXME: Need a makedev implementation in go.

		switch entry.Mode() &^ 07777 {
		case cpio.S_ISCHR:
			// FIXME: skipping due to lack of makedev.
			continue
		case cpio.S_ISBLK:
			// FIXME: skipping due to lack of makedev.
			continue
		case cpio.S_ISDIR:
			log.V(8).Info("creating directory", "path", target)
			m := os.FileMode(entry.Mode()).Perm()
			if err := rootfs.Mkdir(target, m); err != nil && !os.IsExist(err) {
				return fmt.Errorf("creating dir: %w", err)
			}
		case cpio.S_ISFIFO:
			// skip
			continue
		case cpio.S_ISLNK:
			buf := make([]byte, entry.Filesize())
			if _, err := stream.Read(buf); err != nil {
				return fmt.Errorf("reading symlink name: %w", err)
			}
			filename := string(buf)
			log.V(7).Info("creating symlink", "path", target)
			if err := rootfs.Symlink(filename, target); err != nil {
				if os.IsExist(err) {
					log.V(7).Info("skipping symlink since the target already exists", "path", target)
					continue
				}
				return fmt.Errorf("creating symlink: %w", err)
			}
		case cpio.S_ISREG:
			log.V(8).Info("creating file", "path", target)
			// save hardlinks until after the target is written
			if entry.Nlink() > 1 && entry.Filesize() == 0 {
				l, ok := linkMap[entry.Ino()]
				if !ok {
					l = make([]string, 0)
				}
				l = append(l, target)
				linkMap[entry.Ino()] = l
				continue
			}

			f, err := rootfs.Create(target)
			if err != nil {
				return fmt.Errorf("creating file '%s': %w", target, err)
			}
			written, err := io.Copy(f, stream)
			if err != nil {
				return fmt.Errorf("copying file: %w", err)
			}
			if written != int64(entry.Filesize()) {
				return fmt.Errorf("short write")
			}
			if err := f.Close(); err != nil {
				return err
			}

			// fix permissions
			fileMode := os.FileMode(entry.Mode()).Perm()
			log.V(9).Info("updating file permissions", "file", target, "permissions", fileMode)
			if err := rootfs.Chmod(target, fileMode); err != nil {
				return fmt.Errorf("chmodding file %s: %w", target, err)
			}

			// Create hardlinks after the file content is written.
			if entry.Nlink() > 1 && entry.Filesize() > 0 {
				l, ok := linkMap[entry.Ino()]
				if !ok {
					return fmt.Errorf("hardlinks missing")
				}

				for _, t := range l {
					log.V(2).Info("creating hardlink", "target", target, "path", t)
					if err := rootfs.Link(target, t); err != nil {
						if os.IsExist(err) {
							log.V(2).Info("skipping hardlink since the target already exists", "target", target, "path", t)
							continue
						}
						return fmt.Errorf("creating hardlink: %w", err)
					}
				}
			}
		default:
			return fmt.Errorf("unknown file mode 0%o for %s", entry.Mode(), entry.Filename())
		}
	}

	return nil
}

func (p *PackageKeeper) Resolve(ctx context.Context, pkg string) ([]lockfile.Package, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("pkg", pkg)
	// dedupe packages
	packages := map[string]lockfile.Package{}
	for _, idx := range p.indices {
		for _, p := range idx.Package {
			if p.Name == pkg {
				log.V(3).Info("fetching dependencies", "pkg", p.Name)
				dependencies := idx.GetProviders(ctx, p.Format.Requires.Entry.GetValues(), nil)
				for _, dep := range dependencies {
					packages[fmt.Sprintf("%s-%s", dep.Name, dep.Version.Ver)] = lockfile.Package{
						Name:      dep.Name,
						Type:      v1.PackageRPM,
						Version:   dep.Version.Ver,
						Resolved:  strings.TrimSuffix(idx.Source, "/") + "/" + strings.TrimPrefix(dep.Location.Href, "/"),
						Integrity: dep.Checksum.Text,
						Direct:    false,
					}
					log.V(4).Info("collecting package", "name", dep.Name, "version", dep.Version.Ver)
				}
				packages[fmt.Sprintf("%s-%s", p.Name, p.Version.Ver)] = lockfile.Package{
					Name:      p.Name,
					Type:      v1.PackageRPM,
					Version:   p.Version.Ver,
					Resolved:  strings.TrimSuffix(idx.Source, "/") + "/" + strings.TrimPrefix(p.Location.Href, "/"),
					Integrity: p.Checksum.Text,
					Direct:    true,
				}
				log.V(4).Info("collecting package", "name", p.Name, "version", p.Version.Ver)
			}
		}
	}
	results := maps.Values(packages)
	if len(results) > 0 {
		return results, nil
	}
	return nil, fmt.Errorf("package could not be found in any index: %s", pkg)
}
