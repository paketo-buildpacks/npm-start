package npmstart_test

import (
	"encoding/json"
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

		projectPathParser *fakes.PathParser
		reloader          *fakes.Reloader

		detect packit.DetectFunc
	)

	it.Before(func() {
		workingDir = t.TempDir()
		Expect(os.Mkdir(filepath.Join(workingDir, "custom"), os.ModePerm)).To(Succeed())

		projectPathParser = &fakes.PathParser{}
		projectPathParser.GetCall.Returns.ProjectPath = filepath.Join(workingDir, "custom")

		reloader = &fakes.Reloader{}

		detect = npmstart.Detect(projectPathParser, reloader)
	})

	context("when there is a package.json with a start script", func() {
		it.Before(func() {
			content := npmstart.PackageJson{Scripts: npmstart.PackageScripts{
				Start: "node server.js",
			}}

			bytes, err := json.Marshal(content)
			Expect(err).To(BeNil())

			Expect(os.WriteFile(filepath.Join(workingDir, "custom", "package.json"), bytes, 0600)).To(Succeed())
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
			Expect(projectPathParser.GetCall.Receives.Path).To(Equal(filepath.Join(workingDir)))
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
			content := npmstart.PackageJson{Scripts: npmstart.PackageScripts{
				PreStart:  "npm run lint",
				PostStart: "npm run test",
			}}

			bytes, err := json.Marshal(content)
			Expect(err).To(BeNil())

			Expect(os.WriteFile(filepath.Join(workingDir, "custom", "package.json"), bytes, 0600)).To(Succeed())
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
			Expect(err).To(MatchError(packit.Fail.WithMessage(`no "package.json" found in project path %s`, filepath.Join(workingDir, "custom"))))
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
				Expect(err).To(MatchError(ContainSubstring("failed to stat package.json:")))
			})
		})

		context("when the project path cannot be found", func() {
			it.Before(func() {
				projectPathParser.GetCall.Returns.Err = errors.New("some-error")
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError("some-error"))
			})
		})

		context("the reloader returns an error", func() {
			it.Before(func() {
				content := npmstart.PackageJson{Scripts: npmstart.PackageScripts{
					Start: "node server.js",
				}}

				bytes, err := json.Marshal(content)
				Expect(err).To(BeNil())

				Expect(os.WriteFile(filepath.Join(workingDir, "custom", "package.json"), bytes, 0600)).To(Succeed())

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
