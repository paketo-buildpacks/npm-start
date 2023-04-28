package npmstart

import (
	"fmt"
	"os"

	"github.com/paketo-buildpacks/libnodejs"
	"github.com/paketo-buildpacks/libreload-packit"
	"github.com/paketo-buildpacks/packit/v2"
)

type Reloader libreload.Reloader

//go:generate faux --interface Reloader --output fakes/reloader.go

const NoStartScriptError = "no start script in package.json"

func Detect(reloader Reloader) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		projectPath, err := libnodejs.FindProjectPath(context.WorkingDir)
		if err != nil {
			return packit.DetectResult{}, err
		}

		pkg, err := libnodejs.ParsePackageJSON(projectPath)
		if err != nil {
			if os.IsNotExist(err) {
			        return packit.DetectResult{}, packit.Fail.WithMessage("no 'package.json' found in project path %s", projectPath)
			}
			return packit.DetectResult{}, fmt.Errorf("failed to open package.json: %w", err)
		}

		if !pkg.HasStartScript() {
			return packit.DetectResult{}, packit.Fail.WithMessage(NoStartScriptError)
		}

		requirements := []packit.BuildPlanRequirement{
			{
				Name: Node,
				Metadata: map[string]interface{}{
					"launch": true,
				},
			},
			{
				Name: Npm,
				Metadata: map[string]interface{}{
					"launch": true,
				},
			},
			{
				Name: NodeModules,
				Metadata: map[string]interface{}{
					"launch": true,
				},
			},
		}

		if shouldReload, err := reloader.ShouldEnableLiveReload(); err != nil {
			return packit.DetectResult{}, err
		} else if shouldReload {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: "watchexec",
				Metadata: map[string]interface{}{
					"launch": true,
				},
			})
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Requires: requirements,
			},
		}, nil
	}
}
