package main

import (
	npmstart "github.com/paketo-buildpacks/npm-start"
	"github.com/paketo-buildpacks/packit"
)

func main() {
	packit.Run(npmstart.Detect(), npmstart.Build())
}
