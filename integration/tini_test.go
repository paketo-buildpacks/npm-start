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

func testTini(t *testing.T, context spec.G, it spec.S) {
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

	context("when BP_LAUNCH_WITH_TINI=true", func() {
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

		it("uses tini to launch the start script", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "app_with_start_cmd"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithExtensions(
					settings.Extensions.UbiNodejsExtension.Online,
				).
				WithBuildpacks(
					settings.Buildpacks.NodeEngine.Online,
					settings.Buildpacks.NPMInstall.Online,
					settings.Buildpacks.Tini.Online,
					settings.Buildpacks.NPMStart.Online,
				).
				WithEnv(map[string]string{
					"BP_LAUNCH_WITH_TINI": "true",
					"BP_NPM_START_SCRIPT": "start:node",
				}).
				WithPullPolicy(pullPolicy).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s%s \d+\.\d+\.\d+`, extenderBuildStr, settings.Buildpack.Name))))
			Expect(logs).To(ContainLines(
				extenderBuildStr+"  Using tini for process launching",
				extenderBuildStr+"    Warning: skipping prestart and/or poststart scripts because BP_LAUNCH_WITH_TINI is enabled",
				extenderBuildStr+"  Assigning launch processes:",
				extenderBuildStr+"    web (default): tini -g -- node hello.js",
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
			Expect(string(content)).To(ContainSubstring("hello world"))
		})
	})
}
