package npmstart

import (
	"fmt"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func Build(pathParser PathParser, logger scribe.Emitter) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		projectPath, err := pathParser.Get(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		var pkg *PackageJson

		pkg, err = NewPackageJsonFromPath(filepath.Join(projectPath, "package.json"))
		if err != nil {
			return packit.BuildResult{}, err
		}

		command := "node"
		arg := fmt.Sprintf("node %s", filepath.Join(context.WorkingDir, "server.js"))

		if pkg.Scripts.Start != "" {
			command = "bash"
			arg = pkg.Scripts.Start
		}

		if pkg.Scripts.PreStart != "" {
			command = "bash"
			arg = fmt.Sprintf("%s && %s", pkg.Scripts.PreStart, arg)
		}

		if pkg.Scripts.PostStart != "" {
			command = "bash"
			arg = fmt.Sprintf("%s && %s", arg, pkg.Scripts.PostStart)
		}

		// Ideally we would like the lifecycle to support setting a custom working
		// directory to run the launch process.  Until that happens we will cd in.
		if projectPath != context.WorkingDir {
			command = "bash"
			arg = fmt.Sprintf("cd %s && %s", projectPath, arg)
		}

		args := []string{arg}
		switch command {
		case "bash":
			args = []string{"-c", arg}
		case "node":
			args = []string{filepath.Join(context.WorkingDir, "server.js")}
		}

		processes := []packit.Process{
			{
				Type:    "web",
				Command: command,
				Args:    args,
				Default: true,
				Direct:  true,
			},
		}

		shouldReload, err := checkLiveReloadEnabled()
		if err != nil {
			return packit.BuildResult{}, err
		}

		if shouldReload {
			processes = []packit.Process{
				{
					Type:    "web",
					Command: "watchexec",
					Args: append([]string{
						"--restart",
						"--shell", "none",
						"--watch", projectPath,
						"--ignore", filepath.Join(projectPath, "package.json"),
						"--ignore", filepath.Join(projectPath, "package-lock.json"),
						"--ignore", filepath.Join(projectPath, "node_modules"),
						"--",
						command,
					}, args...),
					Default: true,
					Direct:  true,
				},
				{
					Type:    "no-reload",
					Command: command,
					Args:    args,
					Direct:  true,
				},
			}
		}

		logger.LaunchProcesses(processes)

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
