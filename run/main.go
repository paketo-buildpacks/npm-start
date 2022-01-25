package main

import (
	"os"

	npmstart "github.com/paketo-buildpacks/npm-start"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func main() {
	projectPathParser := npmstart.NewProjectPathParser()

	packit.Run(
		npmstart.Detect(projectPathParser),
		npmstart.Build(
			projectPathParser,
			scribe.NewEmitter(os.Stdout),
		),
	)
}
