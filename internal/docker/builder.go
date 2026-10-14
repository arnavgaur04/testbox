package docker

import (
	"fmt"
	"os/exec"
	"testbox/pkg/models"
)

// Builder handles Docker build operations.
type Builder struct{}

// BuildArgsForTest returns the docker build args. Split out for unit tests.
// No Docker needed to test this function.
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

// Build executes a docker build command based on the provided configuration.
func (b *Builder) Build(config models.DockerConfig) error {
	args := BuildArgsForTest(config)

	cmd := exec.Command("docker", args...)
	
	// We run the command in the context directory
	cmd.Dir = config.Context

	fmt.Printf("Running: docker %v in %s\n", args, cmd.Dir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
