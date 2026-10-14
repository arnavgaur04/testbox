package deployer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"testbox/pkg/models"
)

// ComposeTemplate defines the structure for our auto-generated docker-compose file.
// Note: No 'version' key. It is obsolete in Compose v2.
const ComposeTemplate = `services:
{{- range .Services }}
  {{ .ServiceName }}:
    image: {{ $.ImageName }}
    ports:
      - "{{ .Port }}:{{ .ContainerPort }}"
{{- end }}
`

// GenerateComposeFile creates a docker-compose.local.yaml in the .testbox directory
func GenerateComposeFile(repoPath string, config *models.MakeYaml) error {
	testboxDir := filepath.Join(repoPath, ".testbox")
	if err := os.MkdirAll(testboxDir, 0755); err != nil {
		return fmt.Errorf("failed to create .testbox directory: %w", err)
	}
	composeFilePath := filepath.Join(testboxDir, "docker-compose.local.yaml")

	// Prepare data for the template
	data := struct {
		ImageName string
		Services  []models.LocalServiceConfig
	}{
		ImageName: config.Docker.ImageName,
		Services:  config.LocalDeploy.Services,
	}

	// Create the file
	file, err := os.Create(composeFilePath)
	if err != nil {
		return fmt.Errorf("failed to create compose file: %w", err)
	}
	defer file.Close()

	// Parse and execute the template
	tmpl, err := template.New("compose").Parse(ComposeTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("Generated compose file at: %s\n", composeFilePath)
	return nil
}

// Up runs the given compose file in detached mode.
// It uses 'docker compose' (v2). It falls back to 'docker-compose' (v1) if v2 is missing.
func Up(repoPath string, composeFilePath string) error {
	if _, err := os.Stat(composeFilePath); err != nil {
		return fmt.Errorf("compose file not found: %s", composeFilePath)
	}

	// Try v2 first.
	cmd := exec.Command("docker", "compose", "-f", composeFilePath, "up", "-d")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err == nil {
		fmt.Println("Containers started successfully.")
		return nil
	}

	// Fall back to v1.
	cmd = exec.Command("docker-compose", "-f", composeFilePath, "up", "-d")
	cmd.Dir = repoPath
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compose up failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Println("Containers started successfully.")
	return nil
}

// Down stops the services in the given compose file.
func Down(repoPath string, composeFilePath string) error {
	if _, err := os.Stat(composeFilePath); err != nil {
		return fmt.Errorf("compose file not found: %s", composeFilePath)
	}

	cmd := exec.Command("docker", "compose", "-f", composeFilePath, "down")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err == nil {
		fmt.Println("Containers stopped successfully.")
		return nil
	}

	cmd = exec.Command("docker-compose", "-f", composeFilePath, "down")
	cmd.Dir = repoPath
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compose down failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Println("Containers stopped successfully.")
	return nil
}
