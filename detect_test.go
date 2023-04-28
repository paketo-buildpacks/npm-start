package npmstart_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	npmstart "github.com/paketo-buildpacks/npm-start"
	"github.com/paketo-buildpacks/npm-start/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string

		reloader *fakes.Reloader

		detect packit.DetectFunc
	)

	it.Before(func() {
		workingDir = t.TempDir()
		Expect(os.Mkdir(filepath.Join(workingDir, "custom"), os.ModePerm)).To(Succeed())
		t.Setenv("BP_NODE_PROJECT_PATH", "custom")

		reloader = &fakes.Reloader{}

		detect = npmstart.Detect(reloader)
	})

	context("when there is a package.json with a start script", func() {
		it.Before(func() {
			Expect(os.WriteFile(filepath.Join(workingDir, "custom", "package.json"), []byte(`{
				"scripts": {
					"start": "node server.js"
				}
			}`), 0600)).To(Succeed())
		})

		it("detects", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "node",
						Metadata: map[string]interface{}{
							"launch": true,
						},
					},
					{
						Name: "npm",
						Metadata: map[string]interface{}{
							"launch": true,
						},
					},
					{
						Name: "node_modules",
						Metadata: map[string]interface{}{
							"launch": true,
						},
					},
				},
			}))
		})

		context("when live reload is enabled", func() {
			it.Before(func() {
				reloader.ShouldEnableLiveReloadCall.Returns.Bool = true
			})

			it("requires watchexec at launch", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan).To(Equal(packit.BuildPlan{
					Requires: []packit.BuildPlanRequirement{
						{
							Name: "node",
							Metadata: map[string]interface{}{
								"launch": true,
							},
						},
						{
							Name: "npm",
							Metadata: map[string]interface{}{
								"launch": true,
							},
						},
						{
							Name: "node_modules",
							Metadata: map[string]interface{}{
								"launch": true,
							},
						},
						{
							Name: "watchexec",
							Metadata: map[string]interface{}{
								"launch": true,
							},
						},
					},
				}))
			})
		})
	})

	context("when there is a package.json without a start script", func() {
		it.Before(func() {
			Expect(os.WriteFile(filepath.Join(workingDir, "custom", "package.json"), []byte(`{
				"scripts": {
					"prestart":  "npm run lint",
					"poststart": "npm run test"
				}
			}`), 0600)).To(Succeed())
		})

		it.After(func() {
			Expect(os.RemoveAll(workingDir)).To(Succeed())
		})

		it("fails detection", func() {
			_, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).To(MatchError(ContainSubstring(npmstart.NoStartScriptError)))
		})
	})

	context("when there is no package.json", func() {
		it("fails detection", func() {
			_, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).To(MatchError(packit.Fail.WithMessage("no 'package.json' found in project path %s", filepath.Join(workingDir, "custom"))))
		})
	})

	context("failure cases", func() {
		context("the workspace directory cannot be accessed", func() {
			it.Before(func() {
				Expect(os.Chmod(workingDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the project path cannot be found", func() {
			it.Before(func() {
				t.Setenv("BP_NODE_PROJECT_PATH", "does-not-exist")
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(ContainSubstring("could not find project path")))
			})
		})

		context("the reloader returns an error", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "custom", "package.json"), []byte(`{
					"scripts": {
						"start":  "node server.js"
					}
				}`), 0600)).To(Succeed())

				reloader.ShouldEnableLiveReloadCall.Returns.Error = errors.New("reloader error")
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError("reloader error"))
			})
		})
	})
}
