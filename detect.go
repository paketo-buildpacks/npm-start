package npmstart

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
)

func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		_, err := os.Stat(filepath.Join(context.WorkingDir, "package.json"))
		if err != nil {
			if os.IsNotExist(err) {
				return packit.DetectResult{}, packit.Fail
			}
			return packit.DetectResult{}, fmt.Errorf("failed to stat package.json: %w", err)
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Requires: []packit.BuildPlanRequirement{
					{
						Name: Node,
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
					{
						Name: Npm,
						Metadata: map[string]interface{}{
							"launch": true,
						},
					},
					{
						Name: Tini,
						Metadata: map[string]interface{}{
							"launch": true,
						},
					},
				},
			},
		}, nil
	}
}
