package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var settings struct {
	Buildpacks struct {
		NodeEngine struct {
			Online string
		}
		NPMInstall struct {
			Online string
		}
		NPMStart struct {
			Online string
		}
		Watchexec struct {
			Online string
		}
	}
	Buildpack struct {
		ID   string
		Name string
	}
	Config struct {
		NodeEngine         string `json:"node-engine"`
		NPMInstall         string `json:"npm-install"`
		Watchexec          string `json:"watchexec"`
		UbiNodejsExtension string `json:"ubi-nodejs-extension"`
	}

	Extensions struct {
		UbiNodejsExtension struct {
			Online string
		}
	}
}

func TestIntegration(t *testing.T) {
	var docker = occam.NewDocker()

	Expect := NewWithT(t).Expect

	file, err := os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	Expect(json.NewDecoder(file).Decode(&settings.Config)).To(Succeed())

	file, err = os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())

	_, err = toml.NewDecoder(file).Decode(&settings.Buildpack)
	Expect(err).NotTo(HaveOccurred())

	root, err := filepath.Abs("./..")
	Expect(err).NotTo(HaveOccurred())

	buildpackStore := occam.NewBuildpackStore()

	pack := occam.NewPack()

	builder, err := pack.Builder.Inspect.Execute()
	Expect(err).NotTo(HaveOccurred())

	if builder.BuilderName == "paketocommunity/builder-ubi-buildpackless-base" {
		settings.Extensions.UbiNodejsExtension.Online, err = buildpackStore.Get.
			Execute(settings.Config.UbiNodejsExtension)
		Expect(err).ToNot(HaveOccurred())
	}

	settings.Buildpacks.NPMStart.Online, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.NodeEngine.Online, err = buildpackStore.Get.
		Execute(settings.Config.NodeEngine)
	Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.NPMInstall.Online, err = buildpackStore.Get.
		Execute(settings.Config.NPMInstall)
	Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.Watchexec.Online = settings.Config.Watchexec
	err = docker.Pull.Execute(settings.Buildpacks.Watchexec.Online)
	if err != nil {
		t.Fatalf("Failed to pull %s: %s", settings.Buildpacks.Watchexec.Online, err)
	}

	SetDefaultEventuallyTimeout(10 * time.Second)

	suite := spec.New("Integration", spec.Parallel(), spec.Report(report.Terminal{}))
	suite("GracefulShutdown", testGracefulShutdown)
	suite("ProjectPath", testProjectPath)
	suite("ReproducibleBuilds", testReproducibleBuilds)
	suite("StartCommand", testAppWithStartCmd)
	suite.Run(t)
}
