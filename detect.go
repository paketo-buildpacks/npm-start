package npmstart

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/libreload-packit"
	"github.com/paketo-buildpacks/packit/v2"
)

type Reloader libreload.Reloader

//go:generate faux --interface Reloader --output fakes/reloader.go

//go:generate faux --interface PathParser --output fakes/path_parser.go
type PathParser interface {
	Get(path string) (projectPath string, err error)
}

const NoStartScriptError = "no start script in package.json"

func Detect(projectPathParser PathParser, reloader Reloader) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		projectPath, err := projectPathParser.Get(context.WorkingDir)
		if err != nil {
			return packit.DetectResult{}, err
		}

		_, err = os.Stat(filepath.Join(projectPath, "package.json"))
		if err != nil {
			if os.IsNotExist(err) {
				return packit.DetectResult{}, packit.Fail
			}
			return packit.DetectResult{}, fmt.Errorf("failed to stat package.json: %w", err)
		}

		var pkg *PackageJson
		if pkg, err = NewPackageJsonFromPath(filepath.Join(projectPath, "package.json")); err != nil {
			return packit.DetectResult{}, err
		}

		if !pkg.hasStartCommand() {
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
