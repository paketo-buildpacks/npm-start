package matchers

import (
	"fmt"
	"os"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
)

type BeAFileWithSubstringMatcher struct {
	expectedSubstring string
	actualContents    string
}

func BeAFileWithSubstring(expectedSubstring string) *BeAFileWithSubstringMatcher {
	return &BeAFileWithSubstringMatcher{
		expectedSubstring: expectedSubstring,
	}
}

func (matcher *BeAFileWithSubstringMatcher) Match(actual interface{}) (success bool, err error) {
	actualFilename, ok := actual.(string)
	if !ok {
		return false, fmt.Errorf("BeAFileWithSubstringMatcher matcher expects a file path")
	}

	bytes, err := os.ReadFile(actualFilename)
	if err != nil {
		return false, err
	}

	matcher.actualContents = string(bytes)

	return gomega.ContainSubstring(matcher.expectedSubstring).Match(matcher.actualContents)
}

func (matcher *BeAFileWithSubstringMatcher) FailureMessage(actual interface{}) string {
	return format.Message(actual, fmt.Sprintf("to have substring '%s', but were '%s'", matcher.expectedSubstring, matcher.actualContents))
}

func (matcher *BeAFileWithSubstringMatcher) NegatedFailureMessage(actual interface{}) string {
	return format.Message(actual, fmt.Sprintf("not to have substring: %s", matcher.expectedSubstring))
}
