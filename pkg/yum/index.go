package yum

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/carlmjohnson/requests"
	"github.com/djcass44/all-your-base/pkg/requestutil"
	"github.com/djcass44/all-your-base/pkg/yum/yumindex"
	"github.com/djcass44/all-your-base/pkg/yum/yumrepo"
	"github.com/go-logr/logr"
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
	var buf bytes.Buffer
	log.V(4).Info("downloading primary index", "src", primaryURL)

	if err := requests.URL(primaryURL).Handle(requestutil.WithGzip(&buf)).Fetch(ctx); err != nil {
		return nil, fmt.Errorf("downloading primary index: %w", err)
	}
	var index yumindex.Metadata
	if err := xml.NewDecoder(&buf).Decode(&index); err != nil {
		return nil, fmt.Errorf("decoding xml index: %w", err)
	}
	index.Source = repository
	return &index, nil
}

func getMetadata(ctx context.Context, repository string) (*yumrepo.RepoData, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("repo", repository)
	log.V(4).Info("downloading repository metadata")

	target := fmt.Sprintf("%s/repodata/repomd.xml", repository)

	var buf bytes.Buffer
	if err := requests.URL(target).Handle(requestutil.WithGzip(&buf)).Fetch(ctx); err != nil {
		log.Info("failed to download repomd.xml", "url", target)
		return nil, err
	}
	log.V(6).Info("successfully downloaded repository metadata")
	var metadata yumrepo.RepoData
	if err := xml.NewDecoder(&buf).Decode(&metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}
