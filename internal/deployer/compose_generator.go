package deployer

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	testmcp "testbox/internal/mcp"
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
{{- if .Mcp.Enabled }}
  mcp:
{{- if .Mcp.Image }}
    image: {{ .Mcp.Image }}
{{- else }}
    build:
      context: {{ .Mcp.BuildContext }}
      args:
        TESTBOX_REF: {{ .McpRef }}
{{- end }}
    command:
{{- range .Mcp.Command }}
      - "{{ . }}"
{{- end }}
    ports:
      - "{{ .Mcp.HostPort }}:{{ .Mcp.ContainerPort }}"
{{- if .Mcp.DependsOn }}
    depends_on:
{{- range .Mcp.DependsOn }}
      - {{ . }}
{{- end }}
{{- end }}
{{- if and .Mcp.Image (or .Mcp.HasOpenAPI .Mcp.HasTools) }}
    volumes:
{{- if .Mcp.HasOpenAPI }}
      - "{{ .Mcp.StageDir }}/openapi.yaml:/mcp/openapi.yaml:ro"
{{- end }}
{{- if .Mcp.HasTools }}
      - "{{ .Mcp.StageDir }}/tools.yaml:/mcp/tools.yaml:ro"
{{- end }}
{{- end }}
{{- end }}
`

// GenerateComposeFile creates a docker-compose.local.yaml in the .testbox directory.
// mcpRef is the testbox git ref used to build the mcp image. Ignored when mcp is off.
func GenerateComposeFile(repoPath string, config *models.MakeYaml, mcpRef string) error {
	testboxDir := filepath.Join(repoPath, ".testbox")
	if err := os.MkdirAll(testboxDir, 0755); err != nil {
		return fmt.Errorf("failed to create .testbox directory: %w", err)
	}
	composeFilePath := filepath.Join(testboxDir, "docker-compose.local.yaml")

	// MCP service is opt-in. Off by default.
	mcpInfo, err := mcpServiceInfo(repoPath, config)
	if err != nil {
		return err
	}

	// Prepare data for the template
	data := struct {
		ImageName string
		Services  []models.LocalServiceConfig
		Mcp       *testmcp.ServiceInfo
		McpRef    string
	}{
		ImageName: config.Docker.ImageName,
		Services:  config.LocalDeploy.Services,
		Mcp:       mcpInfo,
		McpRef:    mcpRef,
	}
	// Template checks .Mcp.Enabled. Nil-safe default when mcp is off.
	if data.Mcp == nil {
		data.Mcp = &testmcp.ServiceInfo{}
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

// mcpServiceInfo stages the mcp build context when mcp is enabled.
// Returns nil when mcp is off. Target uses the first local service.
func mcpServiceInfo(repoPath string, config *models.MakeYaml) (*testmcp.ServiceInfo, error) {
	if config.Mcp == nil || !config.Mcp.Enabled {
		return nil, nil
	}
	if len(config.LocalDeploy.Services) == 0 {
		return nil, fmt.Errorf("mcp needs at least one local_deploy service as target")
	}
	first := config.LocalDeploy.Services[0]
	target := fmt.Sprintf("http://%s:%d", first.ServiceName, first.ContainerPort)
	dependsOn := make([]string, 0, len(config.LocalDeploy.Services))
	for _, s := range config.LocalDeploy.Services {
		dependsOn = append(dependsOn, s.ServiceName)
	}
	return testmcp.WriteBuildContext(repoPath, config.Mcp.OpenAPI, config.Mcp.ToolsFile, target, config.Mcp.Port, config.Mcp.Image, dependsOn)
}

// Up runs the given compose file in detached mode.
// It uses 'docker compose' (v2). It falls back to 'docker-compose' (v1) if v2 is missing.
func Up(repoPath string, composeFilePath string) error {
	if _, err := os.Stat(composeFilePath); err != nil {
		return fmt.Errorf("compose file not found: %s", composeFilePath)
	}
	output, err := composeOutput(repoPath, composeFilePath, "up", "-d")
	if err != nil {
		return fmt.Errorf("compose up failed: %w\nOutput: %s", err, output)
	}
	fmt.Println("Containers started successfully.")
	return nil
}

// Down stops the services in the given compose file.
func Down(repoPath string, composeFilePath string) error {
	if _, err := os.Stat(composeFilePath); err != nil {
		return fmt.Errorf("compose file not found: %s", composeFilePath)
	}
	output, err := composeOutput(repoPath, composeFilePath, "down")
	if err != nil {
		return fmt.Errorf("compose down failed: %w\nOutput: %s", err, output)
	}
	fmt.Println("Containers stopped successfully.")
	return nil
}
