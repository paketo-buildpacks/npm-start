package npmstart

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/scribe"
)

func Build(pathParser PathParser, logger scribe.Logger) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		var pkg struct {
			Scripts struct {
				PostStart string `json:"poststart"`
				PreStart  string `json:"prestart"`
				Start     string `json:"start"`
			} `json:"scripts"`
		}

		projectPath, err := pathParser.Get(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		file, err := os.Open(filepath.Join(context.WorkingDir, projectPath, "package.json"))
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("unable to open package.json: %w", err)
		}
		defer file.Close()

		err = json.NewDecoder(file).Decode(&pkg)
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("unable to decode package.json: %w", err)
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

		// Ideally we would like the lifecycle to support setting a custom working
		// directory to run the launch process.  Until that happens we will cd in.
		if projectPath != "" {
			command = fmt.Sprintf("cd %s && %s", projectPath, command)
		}

		logger.Process("Assigning launch processes")
		logger.Subprocess("web: %s", command)
		logger.Break()

		return packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{},
			},
			Launch: packit.LaunchMetadata{
				Processes: []packit.Process{
					{
						Type:    "web",
						Command: command,
						Default: true,
					},
				},
			},
		}, nil
	}
}
