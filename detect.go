package pipenvinstall

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2"
)

// BuildPlanMetadata is the buildpack specific data included in build plan
// requirements.
type BuildPlanMetadata struct {
	// Build denotes the dependency is needed at build-time.
	Build bool `toml:"build"`
	// Version denotes the version to request.
	Version string `toml:"version"`
	// VersionSource denotes the source of version request.
	VersionSource string `toml:"version-source"`
}

//go:generate faux --interface Parser --output fakes/parser.go

// Interface to parse python version out of Pipfile.lock.
type Parser interface {
	ParseVersion(path string) (version string, err error)
}

// Detect will return a packit.DetectFunc that will be invoked during the
// detect phase of the buildpack lifecycle.
//
// Detection will contribute a Build Plan that provides site-packages,
// and requires cpython and pipenv at build.
func Detect(pipfileParser, pipfileLockParser Parser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		_, err := os.Stat(filepath.Join(context.WorkingDir, "Pipfile"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return packit.DetectResult{}, packit.Fail
			}

			return packit.DetectResult{}, err
		}

		cpythonRequirement := packit.BuildPlanRequirement{
			Name: CPython,
			Metadata: BuildPlanMetadata{
				Build: true,
			},
		}

		lockFileExists, err := fileExists(filepath.Join(context.WorkingDir, "Pipfile.lock"))
		if err != nil {
			return packit.DetectResult{}, packit.Fail.WithMessage("failed trying to stat Pipfile.lock: %w", err)
		}

		if lockFileExists {
			cpythonVersion, err := pipfileLockParser.ParseVersion(context.WorkingDir)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return packit.DetectResult{}, err
				}
			}

			if cpythonVersion != "" {
				cpythonRequirement.Metadata = BuildPlanMetadata{
					Build:         true,
					Version:       cpythonVersion,
					VersionSource: "Pipfile.lock",
				}
			}
		} else {
			cpythonVersion, err := pipfileParser.ParseVersion(context.WorkingDir)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return packit.DetectResult{}, err
				}
			}

			if cpythonVersion != "" {
				cpythonRequirement.Metadata = BuildPlanMetadata{
					Build:         true,
					Version:       cpythonVersion,
					VersionSource: "Pipfile",
				}
			}
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: SitePackages},
				},
				Requires: []packit.BuildPlanRequirement{
					cpythonRequirement,
					{
						Name: Pipenv,
						Metadata: BuildPlanMetadata{
							Build: true,
						},
					},
				},
			},
		}, nil
	}
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
