package integration_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
	"github.com/paketo-buildpacks/packit/pexec"
)

func testGracefulShutdown(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when building an image from an app that has a SIGTERM handler", func() {
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

		it("builds a working OCI image and gracefully shuts down", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "graceful_shutdown_app"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithBuildpacks(
					nodeBuildpack,
					tiniBuildpack,
					npmBuildpack,
					buildpack,
				).
				WithNoPull().
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			container, err = docker.Container.Run.Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort()))
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()

			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("hello world"))

			Expect(dockerStop(container.ID)).NotTo(HaveOccurred())

			cLogs := func() fmt.Stringer {
				containerLogs, err := docker.Container.Logs.Execute(container.ID)
				Expect(err).NotTo(HaveOccurred())
				return containerLogs
			}

			// this works due to tini
			Eventually(cLogs).Should(ContainSubstring("echo from SIGTERM handler"))
		})
	})
}

func dockerStop(containerID string) error {
	stderr := bytes.NewBuffer(nil)
	exec := pexec.NewExecutable("docker")
	err := exec.Execute(pexec.Execution{
		Args:   []string{"container", "stop", containerID},
		Stderr: stderr,
	})
	if err != nil {
		return fmt.Errorf("failed to stop docker container: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return nil
}
