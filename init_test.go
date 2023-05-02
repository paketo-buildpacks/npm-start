package npmstart_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitNPMStart(t *testing.T) {
	suite := spec.New("npm-start", spec.Report(report.Terminal{}), spec.Sequential())
	suite("Build", testBuild)
	suite("Detect", testDetect)
	suite("ProjectPathParser", testProjectPathParser)
	suite("PackageJsonParser", testPackageJsonParser)
	suite.Run(t)
}
