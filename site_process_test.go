package pipenvinstall_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/pipenv-install/fakes"
	pipenvinstall "github.com/paketo-buildpacks/pipenv-install"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testSiteProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layerPath  string
		executable *fakes.Executable

		process pipenvinstall.SiteProcess
	)

	it.Before(func() {
		var err error
		layerPath, err = ioutil.TempDir("", "layer")
		Expect(err).NotTo(HaveOccurred())

		executable = &fakes.Executable{}
		executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
			if execution.Stdout != nil {
				fmt.Fprintln(execution.Stdout, filepath.Join(layerPath, "/pip/lib/python/site-packages"))
			}
			return nil
		}

		process = pipenvinstall.NewSiteProcess(executable)
	})

	it.After(func() {
		Expect(os.RemoveAll(layerPath)).To(Succeed())
	})

	context("Execute", func() {
		context("there are site packages in the pipenv layer", func() {
			it("returns the full path to the packages", func() {
				sitePackagesPath, err := process.Execute(layerPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution.Env).To(Equal(append(os.Environ(), fmt.Sprintf("PYTHONUSERBASE=%s", layerPath))))
				Expect(executable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{"-m", "site", "--user-site"}))

				Expect(sitePackagesPath).To(Equal(filepath.Join(layerPath, "pip", "lib", "python", "site-packages")))
			})
		})

		context("failure cases", func() {
			context("site package lookup fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						fmt.Fprintln(execution.Stdout, "stdout output")
						fmt.Fprintln(execution.Stderr, "stderr output")
						return errors.New("locating site packages failed")
					}
				})

				it("returns an error", func() {
					_, err := process.Execute(layerPath)
					Expect(err).To(MatchError(ContainSubstring("failed to locate site packages:")))
					Expect(err).To(MatchError(ContainSubstring("stderr output")))
					Expect(err).To(MatchError(ContainSubstring("error: locating site packages failed")))
				})
			})

			context("when the site process returns nothing", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = nil
				})

				it("returns an error", func() {
					_, err := process.Execute(layerPath)
					Expect(err).To(MatchError("failed to locate site packages: output is empty"))
				})
			})
		})
	})
}
