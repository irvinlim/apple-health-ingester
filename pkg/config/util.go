package config

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

func fileExists(fs afero.Fs, path string, failOnError bool) (bool, error) {
	stat, err := fs.Stat(path)
	if err != nil {
		// If failOnError is false we can ignore ENOENT.
		if !failOnError && os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to stat %s", path)
	}
	if stat.IsDir() {
		if failOnError {
			return false, errors.Errorf("%s is a directory", path)
		}
		// Show a warning if it is a directory instead of a file.
		log.WithField("path", path).Warning("warning: config file path is a directory, skipping")
		return false, nil
	}
	return true, nil
}

// expandUser expands the given path (such as "~/.ssh") to refer to the path
// located within the home directory specific to the platform that the program
// is currently running on.
//
// For example on macOS, this expands to "/Users/johndoe/.ssh".
func expandUser(path string) string {
	usr, _ := user.Current()
	dir := usr.HomeDir
	// Exact match.
	if path == "~" {
		return dir
	}
	// Only match prefixes.
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(dir, path[2:])
	}
	return path
}
