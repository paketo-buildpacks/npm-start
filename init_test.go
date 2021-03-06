package npmstart_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitGoBuild(t *testing.T) {
	suite := spec.New("npm-start", spec.Report(report.Terminal{}))
	suite("Build", testBuild)
	suite("Detect", testDetect)
	suite.Run(t)
}
