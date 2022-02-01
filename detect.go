package npmstart

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/paketo-buildpacks/packit/v2"
)

//go:generate faux --interface PathParser --output fakes/path_parser.go
type PathParser interface {
	Get(path string) (projectPath string, err error)
}

const NoStartScriptError = "no start script in package.json"

func Detect(projectPathParser PathParser) packit.DetectFunc {
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

		shouldReload, err := checkLiveReloadEnabled()
		if err != nil {
			return packit.DetectResult{}, err
		}

		if shouldReload {
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

func checkLiveReloadEnabled() (bool, error) {
	if reload, ok := os.LookupEnv("BP_LIVE_RELOAD_ENABLED"); ok {
		shouldEnableReload, err := strconv.ParseBool(reload)
		if err != nil {
			return false, fmt.Errorf("failed to parse BP_LIVE_RELOAD_ENABLED value %s: %w", reload, err)
		}
		return shouldEnableReload, nil
	}
	return false, nil
}
