package linuxutil

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// NewUser adds an entry to the /etc/passwd file to create a new Linux
// user.
func NewUser(ctx context.Context, rootfs, username string, uid int) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("rootfs", rootfs, "username", username, "uid", uid)
	log.Info("creating user")

	path := filepath.Join(rootfs, "etc", "passwd")
	ok, err := containsUser(path, username, uid)
	if err != nil {
		log.Error(err, "failed to check if user already exists")
		return err
	}
	if ok {
		log.Info("user already exists")
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Error(err, "failed to establish directory structure")
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Error(err, "failed to open passwd file")
		return err
	}
	if _, err := file.WriteString(fmt.Sprintf("%s:x:%d:0:Linux User,,,:/home/%s:/sbin/nologin\n", username, uid, username)); err != nil {
		log.Error(err, "failed to write to passwd file")
		return err
	}

	// create the home directory.
	// hopefully the permission bits are correct - https://superuser.com/a/165465
	if err := os.MkdirAll(filepath.Join(rootfs, "home", username), 0775); err != nil {
		log.Error(err, "failed to create home directory")
		return err
	}

	return nil
}

// containsUser checks if a given /etc/passwd file contains a user.
// It checks for a match based the username or uid.
func containsUser(path, username string, uid int) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	br := bufio.NewScanner(f)
	for br.Scan() {
		s := br.Text()
		if strings.Contains(s, fmt.Sprintf("%s:x:", username)) {
			return true, nil
		}
		if strings.Contains(s, fmt.Sprintf(":x:%d:0:", uid)) {
			return true, nil
		}
	}
	if err := br.Err(); err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	return false, nil
}
