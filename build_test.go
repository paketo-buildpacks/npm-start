package npmstart_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/libreload-packit"
	npmstart "github.com/paketo-buildpacks/npm-start"
	"github.com/paketo-buildpacks/npm-start/fakes"
	"github.com/paketo-buildpacks/npm-start/matchers"
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
		reloader   *fakes.Reloader

		startScript string

		buildContext packit.BuildContext
		build        packit.BuildFunc
	)

	it.Before(func() {
		layersDir = t.TempDir()
		cnbDir = t.TempDir()
		workingDir = t.TempDir()

		Expect(os.Mkdir(filepath.Join(workingDir, "some-project-dir"), os.ModePerm)).To(Succeed())
		err := os.WriteFile(filepath.Join(workingDir, "some-project-dir", "package.json"), []byte(`{
			"scripts": {
				"prestart": "some-prestart-command",
				"start": "some-start-command",
				"poststart": "some-poststart-command"
			}
		}`), 0600)
		Expect(err).NotTo(HaveOccurred())

		buffer = bytes.NewBuffer(nil)
		logger := scribe.NewEmitter(buffer)

		t.Setenv("BP_NODE_PROJECT_PATH", "some-project-dir")

		reloader = &fakes.Reloader{}

		startScript = fmt.Sprintf("%s/some-project-dir/start.sh", workingDir)

		buildContext = packit.BuildContext{
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
		}

		build = npmstart.Build(logger, reloader)
	})

	it("returns a result that builds correctly", func() {
		result, err := build(buildContext)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Plan).To(Equal(
			packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{},
			},
		))

		Expect(result.Launch.Processes).To(ConsistOf(packit.Process{
			Type:    "web",
			Command: "sh",
			Default: true,
			Direct:  true,
			Args:    []string{startScript},
		}))

		Expect(startScript).To(matchers.BeAFileWithSubstring("some-prestart-command && some-start-command && some-poststart-command"))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Assigning launch processes:"))
	})

	context("when live reload is enabled", func() {
		it.Before(func() {
			reloader.ShouldEnableLiveReloadCall.Returns.Bool = true
			reloader.TransformReloadableProcessesCall.Returns.Reloadable = packit.Process{
				Type:    "Reloadable",
				Command: "Reloadable",
			}
			reloader.TransformReloadableProcessesCall.Returns.NonReloadable = packit.Process{
				Type:    "NonReloadable",
				Command: "NonReloadable",
			}
		})

		it("adds a reloadable start command that ignores package manager files and makes it the default", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(reloader.TransformReloadableProcessesCall.Receives.OriginalProcess).To(Equal(packit.Process{
				Type:    "web",
				Command: "sh",
				Default: true,
				Direct:  true,
				Args:    []string{startScript},
			}))

			Expect(reloader.TransformReloadableProcessesCall.Receives.Spec).To(Equal(libreload.ReloadableProcessSpec{
				IgnorePaths: []string{
					filepath.Join(workingDir, "some-project-dir", "package.json"),
					filepath.Join(workingDir, "some-project-dir", "package-lock.json"),
					filepath.Join(workingDir, "some-project-dir", "node_modules"),
				},
				WatchPaths: []string{filepath.Join(workingDir, "some-project-dir")},
			}))

			Expect(result.Launch.Processes).To(ConsistOf(packit.Process{
				Type:    "web",
				Command: "Reloadable",
			}, packit.Process{
				Type:    "no-reload",
				Command: "NonReloadable",
			}))

			Expect(startScript).To(matchers.BeAFileWithSubstring("some-prestart-command && some-start-command && some-poststart-command"))

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
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan).To(Equal(
				packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{},
				},
			))

			Expect(result.Launch.Processes).To(ConsistOf(packit.Process{
				Type:    "web",
				Command: "sh",
				Default: true,
				Direct:  true,
				Args:    []string{startScript},
			}))

			Expect(startScript).To(matchers.BeAFileWithSubstring("some-start-command && some-poststart-command"))
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
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan).To(Equal(
				packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{},
				},
			))

			Expect(result.Launch.Processes).To(ConsistOf(packit.Process{
				Type:    "web",
				Command: "sh",
				Default: true,
				Direct:  true,
				Args:    []string{startScript},
			}))

			Expect(startScript).To(matchers.BeAFileWithSubstring("some-prestart-command && some-start-command"))
		})
	})

	context("when the project-path env var is not set", func() {
		it.Before(func() {
			startScript = fmt.Sprintf("%s/start.sh", workingDir)

			err := os.WriteFile(filepath.Join(workingDir, "package.json"), []byte(`{
				"scripts": {
					"prestart": "some-prestart-command",
					"start": "some-start-command",
					"poststart": "some-poststart-command"
				}
			}`), 0600)
			Expect(err).NotTo(HaveOccurred())

			t.Setenv("BP_NODE_PROJECT_PATH", "")
		})

		it.After(func() {
			Expect(os.Remove(filepath.Join(workingDir, "package.json"))).To(Succeed())
		})

		it("returns a result with a valid start command", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan).To(Equal(
				packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{},
				},
			))

			Expect(result.Launch.Processes).To(ConsistOf(packit.Process{
				Type:    "web",
				Command: "sh",
				Default: true,
				Direct:  true,
				Args:    []string{startScript},
			}))

			Expect(startScript).To(matchers.BeAFileWithSubstring("some-prestart-command && some-start-command && some-poststart-command"))
		})
	})

	context("failure cases", func() {
		context("when the package.json file does not exist", func() {
			it.Before(func() {
				Expect(os.Remove(filepath.Join(workingDir, "some-project-dir", "package.json"))).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})

		context("when the package.json is malformed", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "some-project-dir", "package.json"), []byte("%%%"), 0600)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("invalid character '%'")))
			})
		})

		context("when the reloader returns an error", func() {
			it.Before(func() {
				reloader.ShouldEnableLiveReloadCall.Returns.Error = errors.New("some error")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError("some error"))
			})
		})
	})
}
