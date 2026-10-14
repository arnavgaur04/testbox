package docker

import (
	"fmt"
	"os/exec"
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

// Runs docker build for the config.
func (b *Builder) Build(config models.DockerConfig) error {
	args := BuildArgsForTest(config)

	cmd := exec.Command("docker", args...)

	// Run in the context directory.
	cmd.Dir = config.Context

	fmt.Printf("Running: docker %v in %s\n", args, cmd.Dir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
