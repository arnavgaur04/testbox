package deployer

import (
	"fmt"
	"os"
	"os/exec"
)

// Runs a compose command.
// Tries v2 first, then v1.
func composeOutput(repoPath string, composeFilePath string, opArgs ...string) (string, error) {
	v2Args := append([]string{"compose", "-f", composeFilePath}, opArgs...)
	cmd := exec.Command("docker", v2Args...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err == nil {
		return string(output), nil
	}
	v1Err := string(output)

	v1Args := append([]string{"-f", composeFilePath}, opArgs...)
	cmd = exec.Command("docker-compose", v1Args...)
	cmd.Dir = repoPath
	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("compose %v failed (v2: %s): %w\nOutput: %s", opArgs, v1Err, err, string(output))
	}
	return string(output), nil
}

// Returns 'compose ps' output.
func Status(repoPath string, composeFilePath string) (string, error) {
	if _, err := os.Stat(composeFilePath); err != nil {
		return "", fmt.Errorf("compose file not found: %s", composeFilePath)
	}
	return composeOutput(repoPath, composeFilePath, "ps")
}

// Returns 'compose logs' output.
// Empty service means all.
func Logs(repoPath string, composeFilePath string, service string, tail string) (string, error) {
	if _, err := os.Stat(composeFilePath); err != nil {
		return "", fmt.Errorf("compose file not found: %s", composeFilePath)
	}
	if tail == "" {
		tail = "100"
	}
	args := []string{"logs", "--tail", tail}
	if service != "" {
		args = append(args, service)
	}
	return composeOutput(repoPath, composeFilePath, args...)
}
