package pipenvinstall_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

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

		clock      chronos.Clock
		timeStamp  time.Time
		buffer     *bytes.Buffer
		logEmitter scribe.Emitter

		entryResolver       *fakes.EntryResolver
		installProcess      *fakes.InstallProcess
		sitePackagesProcess *fakes.SitePackagesProcess
		venvDirLocator      *fakes.VenvDirLocator
		sbomGenerator       *fakes.SBOMGenerator

		build        packit.BuildFunc
		buildContext packit.BuildContext
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = ioutil.TempDir("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		entryResolver = &fakes.EntryResolver{}
		installProcess = &fakes.InstallProcess{}
		sitePackagesProcess = &fakes.SitePackagesProcess{}
		venvDirLocator = &fakes.VenvDirLocator{}
		sbomGenerator = &fakes.SBOMGenerator{}

		sitePackagesProcess.ExecuteCall.Returns.SitePackagesPath = "some-site-packages-path"
		venvDirLocator.LocateVenvDirCall.Returns.VenvDir = "some-venv-dir"
		sbomGenerator.GenerateCall.Returns.SBOM = sbom.SBOM{}

		buffer = bytes.NewBuffer(nil)
		logEmitter = scribe.NewEmitter(buffer)

		timeStamp = time.Now()
		clock = chronos.NewClock(func() time.Time {
			return timeStamp
		})

		build = pipenvinstall.Build(
			entryResolver,
			installProcess,
			sitePackagesProcess,
			venvDirLocator,
			sbomGenerator,
			clock,
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

		Expect(packagesLayer.Metadata).To(HaveLen(1))
		Expect(packagesLayer.Metadata["built_at"]).To(Equal(timeStamp.Format(time.RFC3339Nano)))

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

		Expect(entryResolver.MergeLayerTypesCall.Receives.Name).To(Equal("site-packages"))
		Expect(entryResolver.MergeLayerTypesCall.Receives.Entries).To(Equal([]packit.BuildpackPlanEntry{
			{Name: "site-packages"},
		}))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))

		Expect(sbomGenerator.GenerateCall.Receives.Dir).To(Equal(workingDir))
	})

	context("site-packages required at build and launch", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			entryResolver.MergeLayerTypesCall.Returns.Build = true
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
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			entryResolver.MergeLayerTypesCall.Returns.Build = true
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

			Expect(cacheLayer.Build).To(BeFalse())
			Expect(cacheLayer.Launch).To(BeFalse())
			Expect(cacheLayer.Cache).To(BeTrue())
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
