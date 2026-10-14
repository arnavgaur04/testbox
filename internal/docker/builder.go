package docker

import (
	"fmt"
	"os/exec"
	"testbox/pkg/models"
)

// Builder handles Docker build operations.
type Builder struct{}

// Build executes a docker build command based on the provided configuration.
func (b *Builder) Build(config models.DockerConfig) error {
	// Prepare arguments for: docker build -t <image_name> -f <dockerfile> <context>
	args := []string{"build", "-t", config.ImageName, "-f", config.Dockerfile}

	// Add build args if present
	for _, arg := range config.BuildArgs {
		args = append(args, "--build-arg", arg)
	}

	// Add target if present
	if config.Target != "" {
		args = append(args, "--target", config.Target)
	}

	// Append context at the end
	args = append(args, config.Context)

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
