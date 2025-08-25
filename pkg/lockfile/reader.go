package lockfile

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
)

func Read(ctx context.Context, cfgPath string) (*Lock, error) {
	log := logr.FromContextOrDiscard(ctx)
	lock, err := os.Open(Name(filepath.Clean(cfgPath)))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("missing lockfile")
		}
		log.Error(err, "failed to open lockfile")
		return nil, err
	}
	defer lock.Close()
	// read the lockfile
	var lockFile Lock
	if err := json.NewDecoder(lock).Decode(&lockFile); err != nil {
		log.Error(err, "failed to read lockfile")
		return nil, err
	}
	return &lockFile, nil
}

func Name(s string) string {
	return strings.TrimSuffix(s, filepath.Ext(s)) + "-lock.json"
}
