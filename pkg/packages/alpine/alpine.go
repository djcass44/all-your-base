package alpine

import (
	"context"
	"github.com/chainguard-dev/go-apk/pkg/apk"
	"github.com/djcass44/all-your-base/pkg/archiveutil"
	"github.com/djcass44/all-your-base/pkg/lockfile"
	"github.com/go-logr/logr"
	"os"
)

var testKeys = map[string][]byte{
	"alpine-devel@lists.alpinelinux.org-616ae350.rsa.pub": []byte(`
-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAyduVzi1mWm+lYo2Tqt/0
XkCIWrDNP1QBMVPrE0/ZlU2bCGSoo2Z9FHQKz/mTyMRlhNqTfhJ5qU3U9XlyGOPJ
piM+b91g26pnpXJ2Q2kOypSgOMOPA4cQ42PkHBEqhuzssfj9t7x47ppS94bboh46
xLSDRff/NAbtwTpvhStV3URYkxFG++cKGGa5MPXBrxIp+iZf9GnuxVdST5PGiVGP
ODL/b69sPJQNbJHVquqUTOh5Ry8uuD2WZuXfKf7/C0jC/ie9m2+0CttNu9tMciGM
EyKG1/Xhk5iIWO43m4SrrT2WkFlcZ1z2JSf9Pjm4C2+HovYpihwwdM/OdP8Xmsnr
DzVB4YvQiW+IHBjStHVuyiZWc+JsgEPJzisNY0Wyc/kNyNtqVKpX6dRhMLanLmy+
f53cCSI05KPQAcGj6tdL+D60uKDkt+FsDa0BTAobZ31OsFVid0vCXtsbplNhW1IF
HwsGXBTVcfXg44RLyL8Lk/2dQxDHNHzAUslJXzPxaHBLmt++2COa2EI1iWlvtznk
Ok9WP8SOAIj+xdqoiHcC4j72BOVVgiITIJNHrbppZCq6qPR+fgXmXa+sDcGh30m6
9Wpbr28kLMSHiENCWTdsFij+NQTd5S47H7XTROHnalYDuF1RpS+DpQidT5tUimaT
JZDr++FjKrnnijbyNF8b98UCAwEAAQ==
-----END PUBLIC KEY-----`),
	"alpine-devel@lists.alpinelinux.org-6165ee59.rsa.pub": []byte(`
-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAutQkua2CAig4VFSJ7v54
ALyu/J1WB3oni7qwCZD3veURw7HxpNAj9hR+S5N/pNeZgubQvJWyaPuQDm7PTs1+
tFGiYNfAsiibX6Rv0wci3M+z2XEVAeR9Vzg6v4qoofDyoTbovn2LztaNEjTkB+oK
tlvpNhg1zhou0jDVYFniEXvzjckxswHVb8cT0OMTKHALyLPrPOJzVtM9C1ew2Nnc
3848xLiApMu3NBk0JqfcS3Bo5Y2b1FRVBvdt+2gFoKZix1MnZdAEZ8xQzL/a0YS5
Hd0wj5+EEKHfOd3A75uPa/WQmA+o0cBFfrzm69QDcSJSwGpzWrD1ScH3AK8nWvoj
v7e9gukK/9yl1b4fQQ00vttwJPSgm9EnfPHLAtgXkRloI27H6/PuLoNvSAMQwuCD
hQRlyGLPBETKkHeodfLoULjhDi1K2gKJTMhtbnUcAA7nEphkMhPWkBpgFdrH+5z4
Lxy+3ek0cqcI7K68EtrffU8jtUj9LFTUC8dERaIBs7NgQ/LfDbDfGh9g6qVj1hZl
k9aaIPTm/xsi8v3u+0qaq7KzIBc9s59JOoA8TlpOaYdVgSQhHHLBaahOuAigH+VI
isbC9vmqsThF2QdDtQt37keuqoda2E6sL7PUvIyVXDRfwX7uMDjlzTxHTymvq2Ck
htBqojBnThmjJQFgZXocHG8CAwEAAQ==
-----END PUBLIC KEY-----`),
}

type PackageKeeper struct {
	indices []apk.NamedIndex
}

func NewPackageKeeper(ctx context.Context, repositories []string) (*PackageKeeper, error) {
	log := logr.FromContextOrDiscard(ctx)
	indices, err := apk.GetRepositoryIndexes(ctx, repositories, testKeys, "x86_64", apk.WithIgnoreSignatures(false))
	if err != nil {
		return nil, err
	}

	log.Info("loaded indices", "count", len(indices))
	for _, i := range indices {
		log.Info("added index", "count", i.Count(), "name", i.Name(), "source", i.Source())
	}

	return &PackageKeeper{
		indices: indices,
	}, nil
}

func (*PackageKeeper) Unpack(ctx context.Context, pkg, rootfs string) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("pkg", pkg, "rootfs", rootfs)
	log.Info("unpacking apk")

	f, err := os.Open(pkg)
	if err != nil {
		log.Error(err, "failed to open file")
		return err
	}
	defer f.Close()

	if err := archiveutil.Guntar(ctx, f, rootfs); err != nil {
		return err
	}
	return nil
}

func (p *PackageKeeper) Resolve(ctx context.Context, pkg string) ([]lockfile.Package, error) {
	resolver := apk.NewPkgResolver(ctx, p.indices)

	// resolve the package
	repoPkg, repoPkgDeps, _, err := resolver.GetPackageWithDependencies(pkg, nil)
	if err != nil {
		return nil, err
	}

	// collect the urls for each package
	names := make([]lockfile.Package, len(repoPkgDeps)+1)
	names[0] = lockfile.Package{
		Name:      repoPkg.Name,
		Resolved:  repoPkg.Url(),
		Integrity: repoPkg.ChecksumString(),
	}
	for i := range repoPkgDeps {
		names[i+1] = lockfile.Package{
			Name:      repoPkgDeps[i].Name,
			Resolved:  repoPkgDeps[i].Url(),
			Integrity: repoPkgDeps[i].ChecksumString(),
		}
	}

	return names, nil
}
