package main

import (
	"os"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	pipenvinstall "github.com/paketo-buildpacks/pipenv-install"
)

func main() {
	planner := draft.NewPlanner()
	logger := scribe.NewEmitter(os.Stdout)
	installProcess := pipenvinstall.NewPipenvInstallProcess(
		pexec.NewExecutable("pipenv"),
		logger,
	)
	pipfileParser := pipenvinstall.NewPipfileParser()
	lockParser := pipenvinstall.NewPipfileLockParser()

	packit.Run(
		pipenvinstall.Detect(
			pipfileParser,
			lockParser,
		),
		pipenvinstall.Build(
			planner,
			installProcess,
			pipenvinstall.NewSiteProcess(pexec.NewExecutable("python")),
			pipenvinstall.NewVenvLocator(),
			chronos.DefaultClock,
			logger,
		),
	)
}
