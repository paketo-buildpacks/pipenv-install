package main

import (
	"os"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/draft"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
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
