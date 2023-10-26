package npmstart

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	libnodejs "github.com/paketo-buildpacks/libnodejs"
	"github.com/paketo-buildpacks/libreload-packit"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func Build(logger scribe.Emitter, reloader Reloader) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		projectPath, err := libnodejs.FindProjectPath(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		pkg, err := libnodejs.ParsePackageJSON(projectPath)
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

		// ubuntu uses dash as the default shell while ubi uses bash as the default shell
		// The version of bash on the current ubi images does not work properly with the
		// signal handling added in the script. Running with bash -c and escaping quotes in
		// the command changes the behavior so that it matches that when running with dash
		// This is fixed in more recent versions of bash ( 5.x and greater) but it will be some
		// time before ubi (and ubuntu it seems) will use that new a version of bash.  This work
		// around is needed until then.
		etcOsReleaseFileContent, err := os.ReadFile(filepath.Join("/etc/os-release"))
		if err == nil {
			re := regexp.MustCompile(`ID=(rhel|"rhel")`)

			match := re.FindStringSubmatch(string(etcOsReleaseFileContent))
			if match != nil {
				arg = fmt.Sprintf("bash -c \"%s\"", strings.Replace(arg, `"`, `\"`, -1))
			}
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
