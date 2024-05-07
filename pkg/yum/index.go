package yum

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/djcass44/all-your-base/pkg/yum/yumindex"
	"github.com/djcass44/all-your-base/pkg/yum/yumrepo"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/hashicorp/go-getter"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func NewIndex(ctx context.Context, repository string) (*yumindex.Metadata, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("repo", repository)
	log.V(1).Info("downloading index")

	repoData, err := getMetadata(ctx, repository)
	if err != nil {
		return nil, err
	}
	primaryURL := repoData.PrimaryXML()
	if primaryURL == "" {
		return nil, errors.New("missing primary XML url")
	}
	if !strings.HasPrefix(primaryURL, "http") {
		primaryURL = fmt.Sprintf("%s/%s", repository, primaryURL)
	}
	dst := fmt.Sprintf("%s/%s-primary.xml", os.TempDir(), uuid.NewString())
	log.V(1).Info("downloading primary index", "src", primaryURL, "dst", dst)
	client := &getter.Client{
		Ctx:             ctx,
		Src:             primaryURL,
		Dst:             dst,
		Mode:            getter.ClientModeFile,
		DisableSymlinks: true,
	}
	if err := client.Get(); err != nil {
		return nil, fmt.Errorf("downloading primary index: %w", err)
	}
	f, err := os.Open(filepath.Clean(dst))
	if err != nil {
		return nil, fmt.Errorf("opening index: %w", err)
	}
	defer f.Close()
	var index yumindex.Metadata
	if err := xml.NewDecoder(f).Decode(&index); err != nil {
		return nil, fmt.Errorf("decoding xml index: %w", err)
	}
	index.Source = repository
	return &index, nil
}

func getMetadata(ctx context.Context, repository string) (*yumrepo.RepoData, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("repo", repository)
	log.V(1).Info("downloading repository metadata")

	target := fmt.Sprintf("%s/repodata/repomd.xml", repository)
	resp, err := http.Get(target)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Info("failed to download repomd.xml", "url", target)
		return nil, fmt.Errorf("http response failed with code: %d", resp.StatusCode)
	}
	log.V(1).Info("successfully downloaded repository metadata", "code", resp.StatusCode)
	var metadata yumrepo.RepoData
	if err := xml.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}
