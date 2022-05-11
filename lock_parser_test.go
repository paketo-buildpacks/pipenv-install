package pipenvinstall_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	pipenvinstall "github.com/paketo-buildpacks/pipenv-install"
	"github.com/sclevine/spec"
)

func testLockParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		parser     pipenvinstall.PipfileLockParser
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		parser = pipenvinstall.NewPipfileLockParser()
	})

	context("Calling ParseVersion", func() {
		context("when Pipfile.lock is valid and specifies a CPython version", func() {

			it.Before(func() {
				Expect(os.WriteFile(
					filepath.Join(workingDir, "Pipfile.lock"),
					[]byte(`{
    "_meta": {
        "hash": {
            "sha256": "6f803d4df721681c56a93ac01ee9234098df1c5aa13b1543ae64c4d77ea38a87"
        },
        "pipfile-spec": 6,
        "requires": {
					"python_version" : "3.8"
				},
        "sources": [
            {
                "url": "https://pypi.python.org/simple",
                "verify_ssl": true
            }
        ]
    }
}`), os.ModePerm)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Remove(filepath.Join(workingDir, "Pipfile.lock"))).To(Succeed())
			})
			it("parses the CPython version", func() {
				version, err := parser.ParseVersion(workingDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(version).To(Equal("3.8"))
			})
		})

		context("failure cases", func() {
			context("when Pipfile.lock file cannot be read", func() {
				it.Before(func() {
					Expect(os.WriteFile(
						filepath.Join(workingDir, "Pipfile.lock"),
						[]byte(`{}`), os.ModePerm)).To(Succeed())
					Expect(os.Chmod(filepath.Join(workingDir, "Pipfile.lock"), 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Remove(filepath.Join(workingDir, "Pipfile.lock"))).To(Succeed())
				})

				it("returns an error", func() {
					_, err := parser.ParseVersion(workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the contents of the Pipfile.lock file are malformed", func() {
				it.Before(func() {
					Expect(os.WriteFile(
						filepath.Join(workingDir, "Pipfile.lock"),
						[]byte(`%%%%%%%%`), os.ModePerm)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Remove(filepath.Join(workingDir, "Pipfile.lock"))).To(Succeed())
				})

				it("returns an error", func() {
					_, err := parser.ParseVersion(workingDir)
					Expect(err).To(MatchError(ContainSubstring("invalid character")))
				})
			})
		})
	})
}
