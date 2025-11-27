package integration_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testProjectPath(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker

		pullPolicy       = "never"
		extenderBuildStr = ""
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()

		if settings.Extensions.UbiNodejsExtension.Online != "" {
			pullPolicy = "always"
			extenderBuildStr = "[extender (build)] "
		}
	})

	context("when building an app with a custom project path set", func() {
		var (
			image     occam.Image
			container occam.Container

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("builds a working OCI image and runs given start cmd", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "project_path_app"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithExtensions(
					settings.Extensions.UbiNodejsExtension.Online,
				).
				WithBuildpacks(
					settings.Buildpacks.NodeEngine.Online,
					settings.Buildpacks.NPMInstall.Online,
					settings.Buildpacks.NPMStart.Online,
				).
				WithPullPolicy(pullPolicy).
				WithEnv(map[string]string{"BP_NODE_PROJECT_PATH": "server"}).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s%s \d+\.\d+\.\d+`, extenderBuildStr, settings.Buildpack.Name))))
			Expect(logs).To(ContainLines(
				extenderBuildStr+"  Assigning launch processes:",
				ContainSubstring("web (default): sh /workspace/server/start.sh"),
				extenderBuildStr+"",
			))

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				Expect(response.Body.Close()).To(Succeed())
			}()

			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := io.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("Hello, World!"))

			cLogs := func() fmt.Stringer {
				containerLogs, err := docker.Container.Logs.Execute(container.ID)
				Expect(err).NotTo(HaveOccurred())
				return containerLogs
			}

			Eventually(cLogs).Should(ContainSubstring("prestart"))
			Eventually(cLogs).Should(ContainSubstring("start"))
		})

		context("when BP_LIVE_RELOAD_ENABLED=true during the build", func() {
			it("makes the default process reloadable and watches the correct subdirectory", func() {
				var err error
				source, err = occam.Source(filepath.Join("testdata", "project_path_app"))
				Expect(err).NotTo(HaveOccurred())

				var logs fmt.Stringer
				image, logs, err = pack.WithNoColor().Build.
					WithExtensions(
						settings.Extensions.UbiNodejsExtension.Online,
					).
					WithBuildpacks(
						settings.Buildpacks.Watchexec.Online,
						settings.Buildpacks.NodeEngine.Online,
						settings.Buildpacks.NPMInstall.Online,
						settings.Buildpacks.NPMStart.Online,
					).
					WithPullPolicy(pullPolicy).
					WithEnv(map[string]string{
						"BP_NODE_PROJECT_PATH":   "server",
						"BP_LIVE_RELOAD_ENABLED": "true",
					}).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String())

				Expect(logs).To(ContainLines(
					MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name))))
				Expect(logs).To(ContainLines(
					extenderBuildStr+"  Assigning launch processes:",
					ContainSubstring("web (default): watchexec --restart --watch /workspace/server --ignore /workspace/server/package.json --ignore /workspace/server/package-lock.json --ignore /workspace/server/node_modules --shell none -- sh /workspace/server/start.sh"),
					ContainSubstring("no-reload:     sh /workspace/server/start.sh"),
					extenderBuildStr+"",
				))

				container, err = docker.Container.Run.
					WithEnv(map[string]string{"PORT": "8080"}).
					WithPublish("8080").
					WithPublishAll().
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(BeAvailable())

				response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					Expect(response.Body.Close()).To(Succeed())
				}()

				Expect(response.StatusCode).To(Equal(http.StatusOK))

				content, err := io.ReadAll(response.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(ContainSubstring("Hello, World!"))

				cLogs := func() fmt.Stringer {
					containerLogs, err := docker.Container.Logs.Execute(container.ID)
					Expect(err).NotTo(HaveOccurred())
					return containerLogs
				}

				Eventually(cLogs).Should(ContainSubstring("prestart"))
				Eventually(cLogs).Should(ContainSubstring("start"))
			})
		})
	})
}
