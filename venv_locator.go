package pipenvinstall

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
)

type VenvLocator struct {
}

func NewVenvLocator() VenvLocator {
	return VenvLocator{}
}

// It would have been cleaner to run "pipenv --venv"
// and extract out the exact virtual env dir,
// but it doesn't seem to work.
// So we look for the dir with pyvenv.cfg in $WORKON_HOME

func (v VenvLocator) LocateVenvDir(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", packit.Fail.WithMessage("reading target directory %s failed:\nerror: %w", path, err)
	}

	venvDir := ""
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		_, err := os.Stat(filepath.Join(path, entry.Name(), "pyvenv.cfg"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", packit.Fail.WithMessage("pipenv virtual env dir lookup failed in target %s: %w", path, err)
		}
		venvDir = entry.Name()
		break
	}

	if venvDir == "" {
		return "", packit.Fail.WithMessage("pipenv virtual env directory not found in target %s", path)
	}

	return filepath.Join(path, venvDir), nil
}
