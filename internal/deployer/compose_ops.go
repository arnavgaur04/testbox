package deployer

import (
	"fmt"
	"os/exec"
)

// composeOutput runs a compose command and returns its output.
// It tries 'docker compose' (v2) first, then 'docker-compose' (v1).
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

// Status returns 'compose ps' output for the given compose file.
func Status(repoPath string, composeFilePath string) (string, error) {
	return composeOutput(repoPath, composeFilePath, "ps")
}

// Logs returns 'compose logs' output.
// service is optional. Empty means all services.
// tail controls --tail lines. Use "all" or a number string. Default "100".
func Logs(repoPath string, composeFilePath string, service string, tail string) (string, error) {
	if tail == "" {
		tail = "100"
	}
	args := []string{"logs", "--tail", tail}
	if service != "" {
		args = append(args, service)
	}
	return composeOutput(repoPath, composeFilePath, args...)
}
