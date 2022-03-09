package pipenvinstall

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
)

//go:generate faux --interface Executable --output fakes/executable.go

// Executable defines the interface for invoking an executable.
type Executable interface {
	Execute(pexec.Execution) error
}

// PipenvInstallProcess implements the InstallProcess interface.
type PipenvInstallProcess struct {
	executable Executable
	logger     scribe.Emitter
}

// NewPipenvInstallProcess creates an instance of the PipenvInstallProcess given an Executable.
func NewPipenvInstallProcess(executable Executable, logger scribe.Emitter) PipenvInstallProcess {
	return PipenvInstallProcess{
		executable: executable,
		logger:     logger,
	}
}

// Execute installs the pipenv dependencies from workingDir/Pipfile into the
// targetLayer. The cacheLayer is used for the pipenv cache directory.
func (p PipenvInstallProcess) Execute(workingDir string, targetLayer, cacheLayer packit.Layer) error {
	targetPath := targetLayer.Path
	cachePath := cacheLayer.Path
	lockExists := true
	args := []string{
		"install",
		// --deploy is for checking Pipefile and lock are in sync
		"--deploy",
		// --system is not used because it does not let us write to a specific path
	}

	_, err := os.Stat(filepath.Join(workingDir, "Pipfile.lock"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			lockExists = false
			args = []string{
				"install",
				// Do not write out a Pipfile.lock. It's not useful and is expensive.
				"--skip-lock",
			}
		} else {
			return fmt.Errorf("failed to stat Pipfile.lock: %w", err)
		}
	}

	p.logger.Subprocess("Running 'pipenv %s'", strings.Join(args, " "))

	buffer := bytes.NewBuffer(nil)
	err = p.executable.Execute(pexec.Execution{
		Args: args,
		Env: append(os.Environ(),
			// Pipenv seems to disregard PYTHONUSERBASE.
			// Target dir set using WORKON_HOME which is a virtualenv setting.
			"PIP_USER=1",
			"PIP_IGNORE_INSTALLED=1",
			fmt.Sprintf("WORKON_HOME=%s", targetPath),
			fmt.Sprintf("PIPENV_CACHE_DIR=%s", cachePath)),
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
	})

	if err != nil {
		return fmt.Errorf("pipenv install failed:\n%s\nerror: %w", buffer.String(), err)
	}

	// if clean is run when no lock file exists, it will generate
	// one, which is an expensive operation.
	if lockExists {
		p.logger.Subprocess("Running 'pipenv clean'")
		buffer.Reset()
		err = p.executable.Execute(pexec.Execution{
			Args: []string{"clean"},
			Env: append(os.Environ(),
				"PIP_USER=1",
				fmt.Sprintf("WORKON_HOME=%s", targetPath),
				fmt.Sprintf("PIPENV_CACHE_DIR=%s", cachePath)),
			Dir:    workingDir,
			Stdout: buffer,
			Stderr: buffer,
		})
		if err != nil {
			return fmt.Errorf("pipenv clean failed:\n%s\nerror: %w", buffer.String(), err)
		}
	}

	return nil
}
