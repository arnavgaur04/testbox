package deployer

import (
	"fmt"
	"testbox/pkg/models"
)

// SuiteResult holds one suite outcome.
type SuiteResult struct {
	Name   string
	Passed bool
	Output string
}

// Runs one suite. Copies files first.
// Passed mirrors the exec exit code.
func RunSuite(repoPath string, files []string, suite models.TestSuite) (SuiteResult, error) {
	res := SuiteResult{Name: suite.Name}
	for _, c := range suite.Copy {
		parts := models.SplitCopyPair(c)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return res, fmt.Errorf("bad copy %q for %s", c, suite.Name)
		}
		// Wipe dest first. Repeat runs nest.
		rm := []string{"exec", "-T", suite.Service, "rm", "-rf", parts[1]}
		if out, err := composeStream(repoPath, files, rm...); err != nil {
			return res, fmt.Errorf("wipe failed for %s: %w\nOutput: %s", suite.Name, err, out)
		}
		dst := suite.Service + ":" + parts[1]
		if out, err := composeStream(repoPath, files, "cp", parts[0], dst); err != nil {
			return res, fmt.Errorf("copy failed for %s: %w\nOutput: %s", suite.Name, err, out)
		}
	}
	out, err := composeStream(repoPath, files, "exec", "-T", suite.Service, "sh", "-c", suite.Command)
	res.Output = out
	res.Passed = err == nil
	if err != nil {
		return res, fmt.Errorf("suite %s failed: %w", suite.Name, err)
	}
	return res, nil
}
