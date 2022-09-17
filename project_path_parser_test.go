package npmstart_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	npmstart "github.com/paketo-buildpacks/npm-start"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testProjectPathParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir        string
		projectDir        string
		projectPathParser npmstart.ProjectPathParser
	)

	it.Before(func() {
		workingDir = t.TempDir()

		projectDir = filepath.Join(workingDir, "custom", "path")
		err := os.MkdirAll(projectDir, os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		projectPathParser = npmstart.NewProjectPathParser()
		t.Setenv("BP_NODE_PROJECT_PATH", "custom/path")
	})

	context("Get", func() {
		it("returns the set project path", func() {
			result, err := projectPathParser.Get(workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(filepath.Join(workingDir, "custom", "path")))
		})
	})

	context("failure cases", func() {
		context("when the project path subdirectory isn't accessible", func() {
			it.Before(func() {
				Expect(os.Chmod(workingDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := projectPathParser.Get(workingDir)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the project path subdirectory does not exist", func() {
			it.Before(func() {
				t.Setenv("BP_NODE_PROJECT_PATH", "some-garbage")
			})

			it("returns an error", func() {
				_, err := projectPathParser.Get(workingDir)
				Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("expected value derived from BP_NODE_PROJECT_PATH [%s] to be an existing directory", "some-garbage"))))
			})
		})

	})
}
