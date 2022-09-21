package npmstart

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/libreload-packit"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func Build(pathParser PathParser, logger scribe.Emitter, reloader Reloader) packit.BuildFunc {
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
		originalProcess := packit.Process{
			Type:    "web",
			Command: command,
			Args:    args,
			Default: true,
			Direct:  true,
		}
		var processes []packit.Process

		if shouldEnableReload, err := reloader.ShouldEnableLiveReload(); err != nil {
			return packit.BuildResult{}, err
		} else if shouldEnableReload {
			nonReloadableProcess, reloadableProcess := reloader.TransformReloadableProcesses(originalProcess, libreload.ReloadableProcessSpec{
				WatchPaths: []string{projectPath},
				IgnorePaths: []string{
					filepath.Join(projectPath, "package.json"),
					filepath.Join(projectPath, "package-lock.json"),
					filepath.Join(projectPath, "node_modules"),
				},
			})
			nonReloadableProcess.Type = "no-reload"
			reloadableProcess.Type = "web"
			processes = append(processes, reloadableProcess, nonReloadableProcess)
		} else {
			processes = append(processes, originalProcess)
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

	path := filepath.Join(targetDir, "start.sh")
	err := os.WriteFile(path, []byte(script), 0644)
	if err != nil {
		return "", err
	}

	return path, nil
}
