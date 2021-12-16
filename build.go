package npmstart

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
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

		processes := []packit.Process{
			{
				Type:    "web",
				Command: command,
				Default: true,
			},
		}

		shouldReload, err := checkLiveReloadEnabled()
		if err != nil {
			return packit.BuildResult{}, err
		}

		if shouldReload {
			processes = []packit.Process{
				{
					Type: "web",
					Command: strings.Join([]string{
						"watchexec",
						"--restart",
						fmt.Sprintf("--watch %s", filepath.Join(context.WorkingDir, projectPath)),
						fmt.Sprintf("--ignore %s", filepath.Join(context.WorkingDir, projectPath, "package.json")),
						fmt.Sprintf("--ignore %s", filepath.Join(context.WorkingDir, projectPath, "package-lock.json")),
						fmt.Sprintf("--ignore %s", filepath.Join(context.WorkingDir, projectPath, "node_modules")),
						fmt.Sprintf(`"%s"`, command),
					}, " "),
					Default: true,
				},
				{
					Type:    "no-reload",
					Command: command,
				},
			}
		}

		logger.Process("Assigning launch processes")
		for _, process := range processes {
			logger.Subprocess("%s: %s", process.Type, process.Command)
		}
		logger.Break()

		return packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{},
			},
			Launch: packit.LaunchMetadata{
				Processes: processes,
			},
		}, nil
	}
}
