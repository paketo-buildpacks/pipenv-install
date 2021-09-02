package pipenvinstall_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
	pipenvinstall "github.com/paketo-buildpacks/pipenv-install"
	"github.com/paketo-buildpacks/pipenv-install/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testInstallProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		packagesLayer     packit.Layer
		cacheLayer        packit.Layer
		packagesLayerPath string
		cacheLayerPath    string
		workingDir        string
		executions        []pexec.Execution
		executable        *fakes.Executable

		pipenvInstallProcess pipenvinstall.PipenvInstallProcess
	)

	it.Before(func() {
		var err error
		packagesLayerPath, err = ioutil.TempDir("", "packages")
		Expect(err).NotTo(HaveOccurred())

		packagesLayer = packit.Layer{Path: packagesLayerPath}
		packagesLayer, err = packagesLayer.Reset()
		Expect(err).NotTo(HaveOccurred())

		cacheLayerPath, err = ioutil.TempDir("", "cache")
		Expect(err).NotTo(HaveOccurred())

		cacheLayer = packit.Layer{Path: cacheLayerPath}
		cacheLayer, err = cacheLayer.Reset()
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "workingdir")
		Expect(err).NotTo(HaveOccurred())

		executions = []pexec.Execution{}
		executable = &fakes.Executable{}
		executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
			executions = append(executions, execution)
			// this is a stub for "pipenv install"
			if len(execution.Args) < 1 || execution.Args[0] != "install" {
				return nil
			}
			fmt.Fprintln(execution.Stdout, "stdout output")
			fmt.Fprintln(execution.Stderr, "stderr output")
			Expect(os.Mkdir(filepath.Join(packagesLayerPath, "some-virtualenv-dir"), os.ModePerm)).To(Succeed())
			f, err := os.Create(filepath.Join(packagesLayerPath, "some-virtualenv-dir", "pyvenv.cfg"))
			Expect(err).NotTo(HaveOccurred())
			f.Close()
			return nil
		}

		pipenvInstallProcess = pipenvinstall.NewPipenvInstallProcess(executable, scribe.NewEmitter(bytes.NewBuffer(nil)))
	})

	it.After(func() {
		Expect(os.RemoveAll(packagesLayerPath)).To(Succeed())
		Expect(os.RemoveAll(cacheLayerPath)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("Execute", func() {
		context("no lock file", func() {
			it.Before(func() {
				Expect(os.RemoveAll(filepath.Join(workingDir, "Pipfile.lock"))).To(Succeed())
			})

			it("runs installation", func() {
				err := pipenvInstallProcess.Execute(workingDir, packagesLayer, cacheLayer)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{
					"install",
					"--skip-lock",
				}))
				Expect(executable.ExecuteCall.Receives.Execution.Dir).To(Equal(workingDir))
				Expect(executable.ExecuteCall.Receives.Execution.Env).To(ContainElement("PIP_USER=1"))
				Expect(executable.ExecuteCall.Receives.Execution.Env).To(ContainElement(fmt.Sprintf("WORKON_HOME=%s", packagesLayerPath)))
				Expect(executable.ExecuteCall.Receives.Execution.Env).To(ContainElement(fmt.Sprintf("PIPENV_CACHE_DIR=%s", cacheLayerPath)))

				Expect(packagesLayer.SharedEnv).To(Equal(packit.Environment{
					"PATH.prepend":           filepath.Join(packagesLayerPath, "some-virtualenv-dir", "bin"),
					"PATH.delim":             ":",
					"PYTHONUSERBASE.default": filepath.Join(packagesLayerPath, "some-virtualenv-dir"),
				}))
			})
		})

		context("has lock file", func() {
			it.Before(func() {
				executions = []pexec.Execution{}
				Expect(os.WriteFile(filepath.Join(workingDir, "Pipfile.lock"), []byte{}, os.ModePerm)).To(Succeed())
			})

			it("runs installation", func() {
				err := pipenvInstallProcess.Execute(workingDir, packagesLayer, cacheLayer)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.CallCount).To(Equal(2))
				Expect(executions[0].Args).To(Equal([]string{
					"install",
					"--deploy",
				}))
				Expect(executions[0].Dir).To(Equal(workingDir))
				Expect(executions[0].Env).To(ContainElement("PIP_USER=1"))
				Expect(executions[0].Env).To(ContainElement(fmt.Sprintf("WORKON_HOME=%s", packagesLayerPath)))
				Expect(executions[0].Env).To(ContainElement(fmt.Sprintf("PIPENV_CACHE_DIR=%s", cacheLayerPath)))

				Expect(executions[1].Args).To(Equal([]string{
					"clean",
				}))
				Expect(executions[1].Dir).To(Equal(workingDir))
				Expect(executions[1].Env).To(ContainElement("PIP_USER=1"))
				Expect(executions[1].Env).To(ContainElement(fmt.Sprintf("WORKON_HOME=%s", packagesLayerPath)))
				Expect(executions[1].Env).To(ContainElement(fmt.Sprintf("PIPENV_CACHE_DIR=%s", cacheLayerPath)))

				Expect(packagesLayer.SharedEnv).To(Equal(packit.Environment{
					"PATH.prepend":           filepath.Join(packagesLayerPath, "some-virtualenv-dir", "bin"),
					"PATH.delim":             ":",
					"PYTHONUSERBASE.default": filepath.Join(packagesLayerPath, "some-virtualenv-dir"),
				}))
			})
		})

		context("failure cases", func() {
			context("when Pipfile.lock stat fails", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					err := pipenvInstallProcess.Execute(workingDir, packagesLayer, cacheLayer)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
