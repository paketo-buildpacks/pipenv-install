package main

import (
	"os"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	pipenvinstall "github.com/paketo-buildpacks/pipenv-install"
)

type Generator struct{}

func (f Generator) Generate(dir string) (sbom.SBOM, error) {
	return sbom.Generate(dir)
}

func main() {
	logger := scribe.NewEmitter(os.Stdout).WithLevel(os.Getenv("BP_LOG_LEVEL"))

	packit.Run(
		pipenvinstall.Detect(
			pipenvinstall.NewPipfileParser(),
			pipenvinstall.NewPipfileLockParser(),
		),
		pipenvinstall.Build(
			pipenvinstall.NewPipenvInstallProcess(pexec.NewExecutable("pipenv"), logger),
			pipenvinstall.NewSiteProcess(pexec.NewExecutable("python")),
			pipenvinstall.NewVenvLocator(),
			Generator{},
			chronos.DefaultClock,
			logger,
		),
	)
}
