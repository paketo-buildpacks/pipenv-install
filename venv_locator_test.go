package pipenvinstall_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	pipenvinstall "github.com/paketo-buildpacks/pipenv-install"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testVenvLocator(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layerPath string

		process pipenvinstall.VenvLocator
	)

	it.Before(func() {
		var err error
		layerPath, err = ioutil.TempDir("", "layer")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.Mkdir(filepath.Join(layerPath, "some-virtualenv-dir"), os.ModePerm)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(layerPath, "some-virtualenv-dir", "pyvenv.cfg"), nil, os.ModePerm)).To(Succeed())

		Expect(os.Mkdir(filepath.Join(layerPath, "some-other-dir"), os.ModePerm)).To(Succeed())

		process = pipenvinstall.NewVenvLocator()
	})

	it.After(func() {
		Expect(os.RemoveAll(layerPath)).To(Succeed())
	})

	context("LocateVenvDir", func() {
		it("returns the full path to the packages", func() {
			venvDir, err := process.LocateVenvDir(layerPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(venvDir).To(Equal(filepath.Join(layerPath, "some-virtualenv-dir")))
		})

		context("failure cases", func() {
			context("when reading the root directory fails", func() {
				it.Before(func() {
					Expect(os.Chmod(layerPath, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(layerPath, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := process.LocateVenvDir(layerPath)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when reading a subdirectory fails", func() {
				it.Before(func() {
					Expect(os.Chmod(filepath.Join(layerPath, "some-virtualenv-dir"), 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(filepath.Join(layerPath, "some-virtualenv-dir"), os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := process.LocateVenvDir(layerPath)
					Expect(err).To(MatchError(ContainSubstring("lookup failed")))
				})
			})

			context("when there is no virtual env directory", func() {
				var emptyLayerPath string
				it.Before(func() {
					var err error
					emptyLayerPath, err = ioutil.TempDir("", "layer")
					Expect(err).NotTo(HaveOccurred())
				})

				it.After(func() {
					Expect(os.RemoveAll(emptyLayerPath)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := process.LocateVenvDir(emptyLayerPath)
					Expect(err).To(MatchError(ContainSubstring("virtual env directory not found")))
				})
			})
		})
	})
}
