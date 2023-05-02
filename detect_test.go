package pipenvinstall_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	pipenvinstall "github.com/paketo-buildpacks/pipenv-install"
	"github.com/paketo-buildpacks/pipenv-install/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect        = NewWithT(t).Expect
		detect        packit.DetectFunc
		lockParser    *fakes.Parser
		pipfileParser *fakes.Parser
		workingDir    string
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		err = os.WriteFile(filepath.Join(workingDir, "Pipfile"), []byte{}, 0644)
		Expect(err).NotTo(HaveOccurred())

		pipfileParser = &fakes.Parser{}
		lockParser = &fakes.Parser{}

		detect = pipenvinstall.Detect(pipfileParser, lockParser)
	})

	context("detection", func() {
		it("returns a build plan that provides site-packages", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: pipenvinstall.SitePackages},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: pipenvinstall.CPython,
						Metadata: pipenvinstall.BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: pipenvinstall.Pipenv,
						Metadata: pipenvinstall.BuildPlanMetadata{
							Build: true,
						},
					},
				},
			}))
			Expect(pipfileParser.ParseVersionCall.Receives.Path).To(Equal(workingDir))
		})

		context("when there is no Pipfile", func() {
			it.Before(func() {
				Expect(os.Remove(filepath.Join(workingDir, "Pipfile"))).To(Succeed())
			})

			it("fails detection", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(packit.Fail.WithMessage("no 'Pipfile' found")))
			})
		})

		context("when there is a Pipfile.lock", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(workingDir, "Pipfile.lock"), []byte{}, 0644)
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				Expect(os.Remove(filepath.Join(workingDir, "Pipfile.lock"))).To(Succeed())
			})

			it("calls Pipfile lock parser", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(lockParser.ParseVersionCall.Receives.Path).To(Equal(workingDir))
			})
		})

		context("failure cases", func() {
			context("when the Pipfile cannot be read", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := detect(packit.DetectContext{
						WorkingDir: workingDir,
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
