package npmstart_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	npmstart "github.com/paketo-buildpacks/npm-start"
	"github.com/paketo-buildpacks/npm-start/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		cnbDir     string
		buffer     *bytes.Buffer
		pathParser *fakes.PathParser

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.Mkdir(filepath.Join(workingDir, "some-project-dir"), os.ModePerm)).To(Succeed())
		err = os.WriteFile(filepath.Join(workingDir, "some-project-dir", "package.json"), []byte(`{
			"scripts": {
				"prestart": "some-prestart-command",
				"start": "some-start-command",
				"poststart": "some-poststart-command"
			}
		}`), 0600)
		Expect(err).NotTo(HaveOccurred())

		buffer = bytes.NewBuffer(nil)
		logger := scribe.NewEmitter(buffer)

		pathParser = &fakes.PathParser{}
		pathParser.GetCall.Returns.ProjectPath = filepath.Join(workingDir, "some-project-dir")

		build = npmstart.Build(pathParser, logger)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that builds correctly", func() {
		result, err := build(packit.BuildContext{
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Stack:      "some-stack",
			BuildpackInfo: packit.BuildpackInfo{
				Name:    "Some Buildpack",
				Version: "some-version",
			},
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{},
			},
			Layers: packit.Layers{Path: layersDir},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Plan).To(Equal(
			packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{},
			},
		))
		processes := result.Launch.Processes
		Expect(processes).To(HaveLen(1))
		process := processes[0]
		Expect(process.Type).To(Equal("web"))
		Expect(process.Command).To(Equal("sh"))
		Expect(process.Default).To(BeTrue())
		Expect(process.Direct).To(BeTrue())
		Expect(process.Args).To(HaveLen(1))
		Expect(process.Args[0]).To(ContainSubstring(fmt.Sprintf("%s/some-project-dir/start.sh", workingDir)))

		filename := process.Args[0]
		Expect(filename).To(BeARegularFile())
		content, err := os.ReadFile(filename)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("some-prestart-command && some-start-command && some-poststart-command"))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Assigning launch processes:"))
	})

	context("when BP_LIVE_RELOAD_ENABLED=true in the build environment", func() {
		it.Before(func() {
			os.Setenv("BP_LIVE_RELOAD_ENABLED", "true")
		})

		it.After(func() {
			os.Unsetenv("BP_LIVE_RELOAD_ENABLED")
		})

		it("adds a reloadable start command that ignores package manager files and makes it the default", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Launch.Processes).To(HaveLen(2))
			processWeb := result.Launch.Processes[0]
			Expect(processWeb.Type).To(Equal("web"))
			Expect(processWeb.Command).To(Equal("watchexec"))
			Expect(processWeb.Args).To(HaveLen(14))
			Expect(processWeb.Args).To(ContainElements(
				"--restart",
				"--shell", "none",
				"--watch", filepath.Join(workingDir, "some-project-dir"),
				"--ignore", filepath.Join(workingDir, "some-project-dir", "package.json"),
				"--ignore", filepath.Join(workingDir, "some-project-dir", "package-lock.json"),
				"--ignore", filepath.Join(workingDir, "some-project-dir", "node_modules"),
				"--",
				"sh",
			))
			Expect(processWeb.Args[13]).To(ContainSubstring(fmt.Sprintf("%s/some-project-dir/start.sh", workingDir)))
			Expect(processWeb.Default).To(BeTrue())
			Expect(processWeb.Direct).To(BeTrue())

			filename := processWeb.Args[13]
			Expect(filename).To(BeARegularFile())
			content, err := os.ReadFile(filename)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("some-start-command && some-poststart-command"))

			processNoReload := result.Launch.Processes[1]
			Expect(processNoReload.Type).To(Equal("no-reload"))
			Expect(processNoReload.Command).To(Equal("sh"))
			Expect(processNoReload.Args).To(HaveLen(1))
			Expect(processNoReload.Args[0]).To(ContainSubstring(fmt.Sprintf("%s/some-project-dir/start.sh", workingDir)))
			Expect(processNoReload.Default).To(BeFalse())
			Expect(processNoReload.Direct).To(BeTrue())

			Expect(pathParser.GetCall.Receives.Path).To(Equal(workingDir))
		})
	})

	context("when there is no prestart script", func() {
		it.Before(func() {
			err := os.WriteFile(filepath.Join(workingDir, "some-project-dir", "package.json"), []byte(`{
				"scripts": {
					"start": "some-start-command",
					"poststart": "some-poststart-command"
				}
			}`), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		it("specifies a valid start command", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan).To(Equal(
				packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{},
				},
			))
			processes := result.Launch.Processes
			Expect(processes).To(HaveLen(1))
			process := processes[0]
			Expect(process.Type).To(Equal("web"))
			Expect(process.Command).To(Equal("sh"))
			Expect(process.Default).To(BeTrue())
			Expect(process.Direct).To(BeTrue())
			Expect(process.Args).To(HaveLen(1))
			Expect(process.Args[0]).To(ContainSubstring(fmt.Sprintf("%s/some-project-dir/start.sh", workingDir)))

			filename := process.Args[0]
			Expect(filename).To(BeARegularFile())
			content, err := os.ReadFile(filename)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("some-start-command && some-poststart-command"))
		})
	})

	context("when there is no poststart script", func() {
		it.Before(func() {
			err := os.WriteFile(filepath.Join(workingDir, "some-project-dir", "package.json"), []byte(`{
				"scripts": {
					"prestart": "some-prestart-command",
					"start": "some-start-command"
				}
			}`), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		it("specifies a valid start command", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan).To(Equal(
				packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{},
				},
			))
			processes := result.Launch.Processes
			Expect(processes).To(HaveLen(1))
			process := processes[0]
			Expect(process.Type).To(Equal("web"))
			Expect(process.Command).To(Equal("sh"))
			Expect(process.Default).To(BeTrue())
			Expect(process.Direct).To(BeTrue())
			Expect(process.Args).To(HaveLen(1))
			Expect(process.Args[0]).To(ContainSubstring(fmt.Sprintf("%s/some-project-dir/start.sh", workingDir)))

			filename := process.Args[0]
			Expect(filename).To(BeARegularFile())
			content, err := os.ReadFile(filename)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("some-prestart-command && some-start-command"))
		})
	})

	context("when the project-path env var is not set", func() {
		it.Before(func() {
			pathParser.GetCall.Returns.ProjectPath = workingDir

			err := os.WriteFile(filepath.Join(workingDir, "package.json"), []byte(`{
				"scripts": {
					"prestart": "some-prestart-command",
					"start": "some-start-command",
					"poststart": "some-poststart-command"
				}
			}`), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(os.Remove(filepath.Join(workingDir, "package.json"))).To(Succeed())
		})

		it("returns a result with a valid start command", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan).To(Equal(
				packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{},
				},
			))
			processes := result.Launch.Processes
			Expect(processes).To(HaveLen(1))
			process := processes[0]
			Expect(process.Type).To(Equal("web"))
			Expect(process.Command).To(Equal("sh"))
			Expect(process.Default).To(BeTrue())
			Expect(process.Direct).To(BeTrue())
			Expect(process.Args).To(HaveLen(1))
			Expect(process.Args[0]).To(ContainSubstring(fmt.Sprintf("%s/start.sh", workingDir)))

			filename := process.Args[0]
			Expect(filename).To(BeARegularFile())
			content, err := os.ReadFile(filename)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("some-prestart-command && some-start-command && some-poststart-command"))

		})
	})

	context("failure cases", func() {
		context("when the package.json file does not exist", func() {
			it.Before(func() {
				Expect(os.Remove(filepath.Join(workingDir, "some-project-dir", "package.json"))).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})

		context("when the package.json is malformed", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "some-project-dir", "package.json"), []byte("%%%"), 0600)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("invalid character '%'")))
			})
		})

		context("when BP_LIVE_RELOAD_ENABLED is set to an invalid value", func() {
			it.Before(func() {
				os.Setenv("BP_LIVE_RELOAD_ENABLED", "not-a-bool")
			})

			it.After(func() {
				os.Unsetenv("BP_LIVE_RELOAD_ENABLED")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse BP_LIVE_RELOAD_ENABLED value not-a-bool")))
			})
		})
	})
}
