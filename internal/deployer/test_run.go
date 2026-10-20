package deployer

import (
	"context"
	"errors"
	"fmt"
	"testbox/pkg/models"
	"time"
)

// SuiteResult holds one suite outcome.
type SuiteResult struct {
	Name     string
	Passed   bool
	Output   string
	Duration time.Duration
}

// Runs one suite. Copies files first.
// Passed mirrors the exec exit code.
func RunSuite(repoPath string, files []string, suite models.TestSuite, timeout time.Duration) (res SuiteResult, err error) {
	res.Name = suite.Name
	start := time.Now()
	defer func() {
		res.Duration = time.Since(start)
	}()
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	for _, c := range suite.Copy {
		parts := models.SplitCopyPair(c)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return res, fmt.Errorf("bad copy %q for %s", c, suite.Name)
		}
		// Wipe dest first. Repeat runs nest.
		rm := []string{"exec", "-T", suite.Service, "rm", "-rf", parts[1]}
		if out, err := composeStreamContext(ctx, repoPath, files, rm...); err != nil {
			return res, fmt.Errorf("wipe failed for %s: %w\nOutput: %s", suite.Name, err, out)
		}
		dst := suite.Service + ":" + parts[1]
		if out, err := composeStreamContext(ctx, repoPath, files, "cp", parts[0], dst); err != nil {
			return res, fmt.Errorf("copy failed for %s: %w\nOutput: %s", suite.Name, err, out)
		}
	}
	out, err := composeStreamContext(ctx, repoPath, files, "exec", "-T", suite.Service, "sh", "-c", suite.Command)
	res.Output = out
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		res.Passed = false
		return res, fmt.Errorf("suite %s timed out after %s", suite.Name, timeout)
	}
	res.Passed = err == nil
	if err != nil {
		return res, fmt.Errorf("suite %s failed: %w", suite.Name, err)
	}
	return res, nil
}
