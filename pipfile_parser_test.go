package pipenvinstall_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	pipenvinstall "github.com/paketo-buildpacks/pipenv-install"
	"github.com/sclevine/spec"
)

func testPipfileParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		parser     pipenvinstall.PipfileParser
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		parser = pipenvinstall.NewPipfileParser()
	})

	context("Calling ParseVersion", func() {
		context("when Pipfile is valid and specifies a CPython version", func() {

			it.Before(func() {
				Expect(ioutil.WriteFile(
					filepath.Join(workingDir, "Pipfile"),
					[]byte(`
[requires]
python_version = "3.8"
`), os.ModePerm)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Remove(filepath.Join(workingDir, "Pipfile"))).To(Succeed())
			})

			it("parses the Python version", func() {
				version, err := parser.ParseVersion(workingDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(version).To(Equal("3.8"))
			})
		})

		context("failure cases", func() {
			context("when Pipfile file cannot be read", func() {
				it.Before(func() {
					Expect(ioutil.WriteFile(
						filepath.Join(workingDir, "Pipfile"),
						[]byte(`{}`), os.ModePerm)).To(Succeed())
					Expect(os.Chmod(filepath.Join(workingDir, "Pipfile"), 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Remove(filepath.Join(workingDir, "Pipfile"))).To(Succeed())
				})

				it("returns an error", func() {
					_, err := parser.ParseVersion(workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the contents of the Pipfile file are malformed", func() {
				it.Before(func() {
					Expect(ioutil.WriteFile(
						filepath.Join(workingDir, "Pipfile"),
						[]byte(`%%%%%%%%`), os.ModePerm)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Remove(filepath.Join(workingDir, "Pipfile"))).To(Succeed())
				})

				it("returns an error", func() {
					_, err := parser.ParseVersion(workingDir)
					Expect(err).To(MatchError(ContainSubstring("parsing error")))
				})
			})
		})
	})
}
