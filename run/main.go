package main

import (
	"os"

	"github.com/paketo-buildpacks/libreload-packit/watchexec"
	npmstart "github.com/paketo-buildpacks/npm-start"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func main() {
	projectPathParser := npmstart.NewProjectPathParser()
	logger := scribe.NewEmitter(os.Stdout).WithLevel(os.Getenv("BP_LOG_LEVEL"))

	reloader := watchexec.NewWatchexecReloader()

	packit.Run(
		npmstart.Detect(projectPathParser, reloader),
		npmstart.Build(
			projectPathParser,
			logger,
			reloader,
		),
	)
}
