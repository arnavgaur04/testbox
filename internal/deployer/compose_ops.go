package deployer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Runs compose command. Merges -f.
// Tries v2 first, then v1.
func composeOutput(repoPath string, files []string, opArgs ...string) (string, error) {
	if err := checkFiles(files); err != nil {
		return "", err
	}
	flagArgs := fileFlags(files, opArgs)
	v2Args := append([]string{"compose"}, flagArgs...)
	cmd := exec.Command("docker", v2Args...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err == nil {
		return string(output), nil
	}
	v1Err := string(output)

	v1Args := fileFlags(files, opArgs)
	cmd = exec.Command("docker-compose", v1Args...)
	cmd.Dir = repoPath
	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("compose %v failed (v2: %s): %w\nOutput: %s", opArgs, v1Err, err, string(output))
	}
	return string(output), nil
}

// Streams a compose command live.
// Full output returned for errors.
func composeStream(repoPath string, files []string, opArgs ...string) (string, error) {
	if err := checkFiles(files); err != nil {
		return "", err
	}
	run := func(prog string, args []string) (string, error) {
		return streamCommand(repoPath, prog, args)
	}
	flagArgs := fileFlags(files, opArgs)
	v2Args := append([]string{"compose"}, flagArgs...)
	out, err := run("docker", v2Args)
	if err == nil {
		return out, nil
	}
	v1Err := out
	out, err = run("docker-compose", flagArgs)
	if err != nil {
		return "", fmt.Errorf("compose %v failed (v2: %s): %w\nOutput: %s", opArgs, v1Err, err, out)
	}
	return out, nil
}

// Runs compose ps. JSON only.
// Stderr stays out of the JSON.
func composePSOutput(repoPath string, files []string) (string, error) {
	if err := checkFiles(files); err != nil {
		return "", err
	}
	run := func(prog string, args []string) (string, error) {
		cmd := exec.Command(prog, args...)
		cmd.Dir = repoPath
		out, err := cmd.Output()
		return string(out), err
	}
	flagArgs := fileFlags(files, []string{"ps", "--format", "json"})
	v2Args := append([]string{"compose"}, flagArgs...)
	out, err := run("docker", v2Args)
	if err == nil {
		return out, nil
	}
	out, err = run("docker-compose", flagArgs)
	if err != nil {
		return "", fmt.Errorf("compose ps failed: %w", err)
	}
	return out, nil
}

// Returns 'compose ps' output.
func Status(repoPath string, files []string) (string, error) {
	return composeOutput(repoPath, files, "ps")
}

// Returns 'compose logs' output.
// Empty service means all.
func Logs(repoPath string, files []string, service string, tail string) (string, error) {
	if tail == "" {
		tail = "100"
	}
	args := []string{"logs", "--tail", tail}
	if service != "" {
		args = append(args, service)
	}
	return composeOutput(repoPath, files, args...)
}

// Missing file fails before Docker.
// Saves a confusing daemon error.
func checkFiles(files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("no compose file given")
	}
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("compose file not found: %s", f)
		}
	}
	return nil
}

// Streams a command. Keeps full output.
func streamCommand(repoPath string, prog string, args []string) (string, error) {
	var buf bytes.Buffer
	cmd := exec.Command(prog, args...)
	cmd.Dir = repoPath
	cmd.Stdout = io.MultiWriter(os.Stdout, &buf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &buf)
	err := cmd.Run()
	return buf.String(), err
}

// Builds -f flags for each file.
func fileFlags(files []string, opArgs []string) []string {
	args := []string{}
	for _, f := range files {
		args = append(args, "-f", f)
	}
	return append(args, opArgs...)
}
