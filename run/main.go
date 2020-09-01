package main

import (
	"os"

	npmstart "github.com/paketo-buildpacks/npm-start"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/scribe"
)

func main() {
	logger := scribe.NewLogger(os.Stdout)

	packit.Run(
		npmstart.Detect(),
		npmstart.Build(logger),
	)
}
