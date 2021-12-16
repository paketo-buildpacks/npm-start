package main

import (
	"os"

	npmstart "github.com/paketo-buildpacks/npm-start"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func main() {
	logger := scribe.NewLogger(os.Stdout)
	projectPathParser := npmstart.NewProjectPathParser()

	packit.Run(
		npmstart.Detect(projectPathParser),
		npmstart.Build(projectPathParser, logger),
	)
}
