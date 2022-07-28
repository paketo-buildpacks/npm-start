package npmstart

import (
	"fmt"
	"os"
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

		command := "sh"
		arg := pkg.Scripts.Start

		if pkg.Scripts.PreStart != "" {
			arg = fmt.Sprintf("%s && %s", pkg.Scripts.PreStart, arg)
		}

		if pkg.Scripts.PostStart != "" {
			arg = fmt.Sprintf("%s && %s", arg, pkg.Scripts.PostStart)
		}

		// Ideally we would like the lifecycle to support setting a custom working
		// directory to run the launch process.  Until that happens we will cd in.
		if projectPath != context.WorkingDir {
			arg = fmt.Sprintf("cd %s && %s", projectPath, arg)
		}

		script, err := createStartupScript(fmt.Sprintf(StartupScript, arg), projectPath, context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		args := []string{script}
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

func createStartupScript(script, projectPath, workingDir string) (string, error) {
	targetDir := workingDir
	if projectPath != workingDir {
		targetDir = projectPath
	}

	f, err := os.CreateTemp(targetDir, "start.sh")
	if err != nil {
		return "", err
	}
	err = f.Chmod(0744)
	if err != nil {
		return "", err
	}

	_, err = f.WriteString(script)
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}
