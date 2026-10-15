package docker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testbox/pkg/models"
)

// Builder handles Docker build operations.
type Builder struct{}

// Returns build args. Split for tests.
// No Docker needed to test this.
func BuildArgsForTest(config models.DockerConfig) []string {
	args := []string{"build", "-t", config.ImageName, "-f", config.Dockerfile}
	for _, arg := range config.BuildArgs {
		args = append(args, "--build-arg", arg)
	}
	if config.Target != "" {
		args = append(args, "--target", config.Target)
	}
	args = append(args, config.Context)
	return args
}

// Defaults BuildKit on for builds.
// Respects parent env when set.
func buildEnv(parent []string) []string {
	for _, e := range parent {
		if strings.HasPrefix(e, "DOCKER_BUILDKIT=") {
			return parent
		}
	}
	return append(parent, "DOCKER_BUILDKIT=1")
}

// Runs docker build for the config.
func (b *Builder) Build(config models.DockerConfig) error {
	args := BuildArgsForTest(config)

	cmd := exec.Command("docker", args...)
	cmd.Env = buildEnv(os.Environ())

	// Run in the context directory.
	cmd.Dir = config.Context

	fmt.Printf("Running: docker %v in %s\n", args, cmd.Dir)

	// Stream live. Keep copy for errors.
	var buf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &buf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &buf)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w\nOutput: %s", err, buf.String())
	}

	return nil
}
