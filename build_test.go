package pipenvinstall_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	pipenvinstall "github.com/paketo-buildpacks/pipenv-install"
	"github.com/paketo-buildpacks/pipenv-install/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		cnbDir     string

		buffer     *bytes.Buffer
		logEmitter scribe.Emitter

		installProcess      *fakes.InstallProcess
		sitePackagesProcess *fakes.SitePackagesProcess
		venvDirLocator      *fakes.VenvDirLocator
		sbomGenerator       *fakes.SBOMGenerator

		build        packit.BuildFunc
		buildContext packit.BuildContext
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.MkdirAll(filepath.Join(layersDir, "cache"), os.ModePerm)).To(Succeed())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		installProcess = &fakes.InstallProcess{}
		sitePackagesProcess = &fakes.SitePackagesProcess{}
		venvDirLocator = &fakes.VenvDirLocator{}
		sbomGenerator = &fakes.SBOMGenerator{}

		sitePackagesProcess.ExecuteCall.Returns.SitePackagesPath = "some-site-packages-path"
		venvDirLocator.LocateVenvDirCall.Returns.VenvDir = "some-venv-dir"
		sbomGenerator.GenerateCall.Returns.SBOM = sbom.SBOM{}

		buffer = bytes.NewBuffer(nil)
		logEmitter = scribe.NewEmitter(buffer)

		build = pipenvinstall.Build(
			installProcess,
			sitePackagesProcess,
			venvDirLocator,
			sbomGenerator,
			chronos.DefaultClock,
			logEmitter)

		buildContext = packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				Name:        "Some Buildpack",
				Version:     "some-version",
				SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
			},
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "site-packages",
					},
				},
			},
			Platform: packit.Platform{Path: "some-platform-path"},
			Layers:   packit.Layers{Path: layersDir},
			Stack:    "some-stack",
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	it("runs the build process and returns expected layers", func() {
		result, err := build(buildContext)
		Expect(err).NotTo(HaveOccurred())

		layers := result.Layers
		Expect(layers).To(HaveLen(1))

		packagesLayer := layers[0]
		Expect(packagesLayer.Name).To(Equal("packages"))
		Expect(packagesLayer.Path).To(Equal(filepath.Join(layersDir, "packages")))

		Expect(packagesLayer.Build).To(BeFalse())
		Expect(packagesLayer.Launch).To(BeFalse())
		Expect(packagesLayer.Cache).To(BeFalse())

		Expect(packagesLayer.BuildEnv).To(BeEmpty())
		Expect(packagesLayer.LaunchEnv).To(BeEmpty())
		Expect(packagesLayer.ProcessLaunchEnv).To(BeEmpty())

		Expect(packagesLayer.SharedEnv).To(HaveLen(4))
		Expect(packagesLayer.SharedEnv["PATH.prepend"]).To(Equal(filepath.Join("some-venv-dir", "bin")))
		Expect(packagesLayer.SharedEnv["PATH.delim"]).To(Equal(":"))
		Expect(packagesLayer.SharedEnv["PYTHONPATH.prepend"]).To(Equal("some-site-packages-path"))
		Expect(packagesLayer.SharedEnv["PYTHONPATH.delim"]).To(Equal(":"))

		Expect(packagesLayer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
			{
				Extension: sbom.Format(sbom.CycloneDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
			},
			{
				Extension: sbom.Format(sbom.SPDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
			},
		}))

		Expect(installProcess.ExecuteCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(installProcess.ExecuteCall.Receives.TargetLayer.Path).To(Equal(filepath.Join(layersDir, "packages")))
		Expect(installProcess.ExecuteCall.Receives.CacheLayer.Path).To(Equal(filepath.Join(layersDir, "cache")))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))

		Expect(sbomGenerator.GenerateCall.Receives.Dir).To(Equal(workingDir))
	})

	context("site-packages required at build and launch", func() {
		it.Before(func() {
			buildContext.Plan.Entries[0].Metadata = map[string]interface{}{
				"launch": true,
				"build":  true,
			}
		})

		it("layer's build, launch, cache flags must be set", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			layers := result.Layers
			Expect(layers).To(HaveLen(1))

			packagesLayer := layers[0]
			Expect(packagesLayer.Name).To(Equal("packages"))

			Expect(packagesLayer.Build).To(BeTrue())
			Expect(packagesLayer.Launch).To(BeTrue())
			Expect(packagesLayer.Cache).To(BeTrue())
		})
	})

	context("install process utilizes cache", func() {
		it.Before(func() {
			installProcess.ExecuteCall.Stub = func(_ string, _, cacheLayer packit.Layer) error {
				err := os.MkdirAll(filepath.Join(cacheLayer.Path, "something"), os.ModePerm)
				if err != nil {
					return fmt.Errorf("issue with stub call: %+v", err)
				}
				return nil
			}
			buildContext.Plan.Entries[0].Metadata = map[string]interface{}{
				"launch": true,
				"build":  true,
			}
		})

		it("result should include a cache layer", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			layers := result.Layers
			Expect(layers).To(HaveLen(2))

			packagesLayer := layers[0]
			Expect(packagesLayer.Name).To(Equal("packages"))

			cacheLayer := layers[1]
			Expect(cacheLayer.Name).To(Equal("cache"))
			Expect(cacheLayer.Path).To(Equal(filepath.Join(layersDir, "cache")))
			Expect(cacheLayer.Metadata["stack"]).To(Equal("some-stack"))

			Expect(cacheLayer.Build).To(BeFalse())
			Expect(cacheLayer.Launch).To(BeFalse())
			Expect(cacheLayer.Cache).To(BeTrue())
		})
	})

	context("if the stack id changes", func() {
		it.Before(func() {
			Expect(os.WriteFile(filepath.Join(layersDir, "cache", "some-cache"), []byte{}, 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layersDir, "cache.toml"), []byte(`
[metadata]
  stack = "some-stack"
`), 0644))
		})

		it("empties the cache layer", func() {
			_, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "0.0.1",
				},
				Platform: packit.Platform{
					Path: "some-platform-path",
				},
				Layers: packit.Layers{Path: layersDir},
				Stack:  "other-stack",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(layersDir, "cache", "some-cache")).NotTo(BeAnExistingFile())
		})
	})

	context("if the stack id is the same", func() {
		it.Before(func() {
			Expect(os.WriteFile(filepath.Join(layersDir, "cache", "some-cache"), []byte{}, 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layersDir, "cache.toml"), []byte(`
[metadata]
  stack = "some-stack"
`), 0644))
		})

		it("keeps the cache layer", func() {
			_, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "0.0.1",
				},
				Platform: packit.Platform{
					Path: "some-platform-path",
				},
				Layers: packit.Layers{Path: layersDir},
				Stack:  "some-stack",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(layersDir, "cache", "some-cache")).To(BeAnExistingFile())
		})
	})

	context("failure cases", func() {
		context("when the layers directory cannot be written to", func() {
			it.Before(func() {
				Expect(os.Chmod(layersDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the cache layer cannot be reset", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(layersDir, "cache"), os.ModePerm)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(layersDir, "cache", "some-cache"), []byte{}, 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(layersDir, "cache.toml"), []byte(`
[metadata]
  stack = "other-stack"
`), 0644))
				Expect(os.Chmod(filepath.Join(layersDir, "cache"), 0500)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(layersDir, "cache"), os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when install process returns an error", func() {
			it.Before(func() {
				installProcess.ExecuteCall.Returns.Error = errors.New("some-error")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("some-error")))
			})
		})

		context("when venv directory locator returns an error", func() {
			it.Before(func() {
				venvDirLocator.LocateVenvDirCall.Returns.Err = errors.New("some-venv-error")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("some-venv-error")))
			})
		})

		context("when site packages process locator returns an error", func() {
			it.Before(func() {
				sitePackagesProcess.ExecuteCall.Returns.Err = errors.New("some-site-error")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("some-site-error")))
			})
		})

		context("when generating the SBOM returns an error", func() {
			it.Before(func() {
				buildContext.BuildpackInfo.SBOMFormats = []string{"random-format"}
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(`unsupported SBOM format: 'random-format'`))
			})
		})

		context("when formatting the SBOM returns an error", func() {
			it.Before(func() {
				sbomGenerator.GenerateCall.Returns.Error = errors.New("failed to generate SBOM")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("failed to generate SBOM")))
			})
		})
	})
}
