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

var (
	buildpack     string
	nodeBuildpack string
	tiniBuildpack string
	npmBuildpack  string
	buildpackInfo struct {
		Buildpack struct {
			ID   string
			Name string
		}
	}
)

func TestIntegration(t *testing.T) {
	var (
		Expect = NewWithT(t).Expect
		err    error
	)

	var config struct {
		NodeEngine string `json:"node-engine"`
		Npm        string `json:"npm"`
		Tini       string `json:"tini"`
	}

	file, err := os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	Expect(json.NewDecoder(file).Decode(&config)).To(Succeed())

	file, err = os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())

	_, err = toml.DecodeReader(file, &buildpackInfo)
	Expect(err).NotTo(HaveOccurred())

	root, err := filepath.Abs("./..")
	Expect(err).NotTo(HaveOccurred())

	buildpackStore := occam.NewBuildpackStore()

	buildpack, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).ToNot(HaveOccurred())

	nodeBuildpack, err = buildpackStore.Get.
		Execute(config.NodeEngine)
	Expect(err).ToNot(HaveOccurred())

	npmBuildpack, err = buildpackStore.Get.
		Execute(config.Npm)
	Expect(err).ToNot(HaveOccurred())

	tiniBuildpack, err = buildpackStore.Get.
		Execute(config.Tini)
	Expect(err).ToNot(HaveOccurred())

	SetDefaultEventuallyTimeout(10 * time.Second)

	suite := spec.New("Integration", spec.Parallel(), spec.Report(report.Terminal{}))
	suite("SimpleApp", testSimpleApp)
	suite.Run(t)
}
