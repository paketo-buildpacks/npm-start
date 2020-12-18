package npmstart

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/scribe"
)

func Build(logger scribe.Logger) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		var pkg struct {
			Scripts struct {
				PostStart string `json:"poststart"`
				PreStart  string `json:"prestart"`
				Start     string `json:"start"`
			} `json:"scripts"`
		}

		file, err := os.Open(filepath.Join(context.WorkingDir, "package.json"))
		if err != nil {
			return packit.BuildResult{}, err
		}
		defer file.Close()

		err = json.NewDecoder(file).Decode(&pkg)
		if err != nil {
			return packit.BuildResult{}, err
		}

		command := "node server.js"
		if pkg.Scripts.Start != "" {
			command = pkg.Scripts.Start
		}

		if pkg.Scripts.PreStart != "" {
			command = fmt.Sprintf("%s && %s", pkg.Scripts.PreStart, command)
		}

		if pkg.Scripts.PostStart != "" {
			command = fmt.Sprintf("%s && %s", command, pkg.Scripts.PostStart)
		}

		logger.Process("Assigning launch processes")
		logger.Subprocess("web: %s", command)

		return packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{},
			},
			Launch: packit.LaunchMetadata{
				Processes: []packit.Process{
					{
						Type:    "web",
						Command: command,
					},
				},
			},
		}, nil
	}
}
